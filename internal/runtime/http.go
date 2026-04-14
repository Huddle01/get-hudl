package runtime

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Huddle01/get-hudl/internal/config"
	"golang.org/x/term"
)

type Backend string

const (
	BackendCloud Backend = "cloud"
	BackendGPU   Backend = "gpu"
)

type Request struct {
	Backend        Backend
	Method         string
	Path           string
	Query          map[string]string
	Body           any
	Mutating       bool
	IdempotencyKey string
}

type HTTPError struct {
	StatusCode int    `json:"status_code" yaml:"status_code"`
	Message    string `json:"message" yaml:"message"`
	Body       any    `json:"body,omitempty" yaml:"body,omitempty"`
	RequestID  string `json:"request_id,omitempty" yaml:"request_id,omitempty"`
	URL        string `json:"url,omitempty" yaml:"url,omitempty"`
}

func (e *HTTPError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("request failed with status %d", e.StatusCode)
}

type Client struct {
	httpClient  *http.Client
	cloudAPIKey string
	gpuAPIKey   string
	cloudBase   string
	gpuBase     string
	stderr      io.Writer
	verbose     bool
}

func NewApp(stdin io.Reader, stdout io.Writer, stderr io.Writer, opts GlobalOptions, resolved config.Resolved) *App {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	gpuKey := resolved.GPUAPIKey
	if gpuKey == "" {
		gpuKey = resolved.APIKey // fallback to shared key
	}
	client := &Client{
		httpClient:  &http.Client{Timeout: timeout},
		cloudAPIKey: resolved.APIKey,
		gpuAPIKey:   gpuKey,
		cloudBase:   resolved.CloudBase,
		gpuBase:     resolved.GPUBase,
		stderr:      stderr,
		verbose:     opts.Verbose,
	}

	return &App{
		Stdin:    stdin,
		Stdout:   stdout,
		Stderr:   stderr,
		Options:  opts,
		Config:   resolved,
		HTTP:     client,
		IsTTYOut: fileIsTTY(stdout),
		IsTTYIn:  fileIsTTY(stdin),
	}
}

func fileIsTTY(stream any) bool {
	if file, ok := stream.(interface{ Fd() uintptr }); ok {
		return term.IsTerminal(int(file.Fd()))
	}
	return false
}

func (c *Client) Do(req Request) (map[string]any, error) {
	apiKey := c.cloudAPIKey
	if req.Backend == BackendGPU {
		apiKey = c.gpuAPIKey
	}
	if apiKey == "" {
		if req.Backend == BackendGPU {
			return nil, fmt.Errorf("no GPU API key configured; run `hudl login --gpu-token <key>` or set HUDL_GPU_API_KEY")
		}
		return nil, fmt.Errorf("no API key configured; run `hudl login --token <key>` or set HUDL_API_KEY")
	}

	base := c.cloudBase
	if req.Backend == BackendGPU {
		base = c.gpuBase
	}

	u, err := url.Parse(strings.TrimRight(base, "/") + req.Path)
	if err != nil {
		return nil, err
	}

	query := u.Query()
	for key, value := range req.Query {
		if value != "" {
			query.Set(key, value)
		}
	}
	u.RawQuery = query.Encode()

	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, err = json.Marshal(req.Body)
		if err != nil {
			return nil, err
		}
	}

	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		httpReq, err := http.NewRequest(req.Method, u.String(), bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Accept", "application/json")
		httpReq.Header.Set("X-API-Key", apiKey)
		if req.Backend == BackendGPU {
			httpReq.Header.Set("Authorization", "Bearer "+apiKey)
		}
		if req.Body != nil {
			httpReq.Header.Set("Content-Type", "application/json")
		}
		if req.Mutating {
			key := req.IdempotencyKey
			if key == "" {
				key = randomID("hudl")
			}
			httpReq.Header.Set("Idempotency-Key", key)
		}

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			lastErr = err
			if !shouldRetry(err, 0) {
				return nil, err
			}
			sleepForAttempt(attempt)
			continue
		}

		raw, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}

		if c.verbose {
			fmt.Fprintf(c.stderr, "[hudl] %s %s -> %d\n", req.Method, u.String(), resp.StatusCode)
		}

		if resp.StatusCode >= 400 {
			lastErr = buildHTTPError(resp, raw, u.String())
			if shouldRetry(lastErr, resp.StatusCode) {
				sleepForAttempt(attempt)
				continue
			}
			return nil, lastErr
		}

		if len(raw) == 0 {
			return map[string]any{"ok": true}, nil
		}

		var decoded map[string]any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return map[string]any{"raw": string(raw)}, nil
		}
		return decoded, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("request failed")
	}
	return nil, lastErr
}

func buildHTTPError(resp *http.Response, raw []byte, requestURL string) error {
	errValue := &HTTPError{
		StatusCode: resp.StatusCode,
		URL:        requestURL,
		RequestID:  resp.Header.Get("X-Request-Id"),
	}

	var decoded map[string]any
	if len(raw) > 0 && json.Unmarshal(raw, &decoded) == nil {
		errValue.Body = decoded
		if text, ok := decoded["error"].(string); ok && text != "" {
			errValue.Message = text
		} else if text, ok := decoded["message"].(string); ok && text != "" {
			errValue.Message = text
		} else if text, ok := decoded["code"].(string); ok && text != "" {
			errValue.Message = text
		}
	}
	if errValue.Message == "" {
		errValue.Message = strings.TrimSpace(string(raw))
	}
	if errValue.Message == "" {
		errValue.Message = resp.Status
	}
	return errValue
}

func shouldRetry(err error, statusCode int) bool {
	if statusCode == http.StatusTooManyRequests || statusCode >= 500 {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}

func sleepForAttempt(attempt int) {
	base := math.Pow(2, float64(attempt)) * 150
	jitter := rand.Float64() * 100
	time.Sleep(time.Duration(base+jitter) * time.Millisecond)
}

func randomID(prefix string) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	buf := make([]byte, 12)
	for i := range buf {
		buf[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return prefix + "_" + string(buf)
}
