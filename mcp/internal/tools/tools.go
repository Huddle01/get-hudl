package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/Huddle01/get-hudl/internal/config"
	"github.com/Huddle01/get-hudl/internal/runtime"
	"github.com/Huddle01/get-hudl/mcp/internal/server"
)

// RegisterAll registers every Huddle01 Cloud tool on the MCP server.
func RegisterAll(srv *server.Server) {
	registerAuthTools(srv)
	registerContextTools(srv)
	registerVMTools(srv)
	registerVolumeTools(srv)
	registerFloatingIPTools(srv)
	registerSecurityGroupTools(srv)
	registerNetworkTools(srv)
	registerKeyTools(srv)
	registerLookupTools(srv)
	registerGPUTools(srv)
	registerGPUWaitlistTools(srv)
	registerGPUImageTools(srv)
	registerGPUVolumeTools(srv)
	registerGPUSSHKeyTools(srv)
	registerGPUAPIKeyTools(srv)
	registerGPUWebhookTools(srv)
	registerGPURegionTools(srv)
}

// cachedApp holds the singleton App loaded once at first use.
// Config is re-read only on auth/context mutations.
var (
	cachedApp *runtime.App
	cacheMu   sync.Mutex
)

func loadApp() (*runtime.App, error) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if cachedApp != nil {
		return cachedApp, nil
	}
	return reloadAppLocked()
}

func reloadApp() (*runtime.App, error) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	return reloadAppLocked()
}

func reloadAppLocked() (*runtime.App, error) {
	resolved, err := config.Load(config.Flags{})
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	app := runtime.NewApp(nil, io.Discard, io.Discard, runtime.GlobalOptions{}, resolved)
	cachedApp = app
	return app, nil
}

// invalidateCache forces the next request to re-read config from disk.
// Call this after mutations to auth or context.
func invalidateCache() {
	cacheMu.Lock()
	cachedApp = nil
	cacheMu.Unlock()
}

func do(req runtime.Request) (map[string]any, error) {
	app, err := loadApp()
	if err != nil {
		return nil, err
	}
	return app.HTTP.Do(req)
}

func requireRegion() (string, error) {
	app, err := loadApp()
	if err != nil {
		return "", err
	}
	if app.Config.Region == "" {
		return "", fmt.Errorf("region is required; set HUDL_REGION or run `hudl ctx region <region>`")
	}
	return app.Config.Region, nil
}

func cloudRequest(method, path string, query map[string]string, body any, mutating bool) (map[string]any, error) {
	region, err := requireRegion()
	if err != nil {
		return nil, err
	}
	if query == nil {
		query = map[string]string{}
	}
	query["region"] = region
	return do(runtime.Request{Backend: runtime.BackendCloud, Method: method, Path: path, Query: query, Body: body, Mutating: mutating})
}

func gpuRequest(method, path string, query map[string]string, body any, mutating bool) (map[string]any, error) {
	return do(runtime.Request{Backend: runtime.BackendGPU, Method: method, Path: path, Query: query, Body: body, Mutating: mutating})
}

// wrapError converts runtime.HTTPError into a structured JSON error that
// preserves status code, request ID, and body for AI agent consumption.
func wrapError(err error) error {
	httpErr, ok := err.(*runtime.HTTPError)
	if !ok {
		return err
	}
	payload := map[string]any{
		"status_code": httpErr.StatusCode,
		"message":     httpErr.Message,
	}
	if httpErr.RequestID != "" {
		payload["request_id"] = httpErr.RequestID
	}
	if httpErr.Body != nil {
		payload["body"] = httpErr.Body
	}
	data, jsonErr := json.Marshal(payload)
	if jsonErr != nil {
		return err
	}
	return fmt.Errorf("%s", string(data))
}

// Extract helpers — mirrors the CLI extractors.

func extractData(raw map[string]any) any {
	if d, ok := raw["data"]; ok {
		return d
	}
	return raw
}

func extractKey(raw map[string]any, key string) any {
	if key == "" {
		return raw
	}
	if v, ok := raw[key]; ok {
		return v
	}
	return raw
}

func extractCloudList(raw map[string]any, key string) any {
	if v, ok := raw[key]; ok {
		return v
	}
	if data, ok := raw["data"].(map[string]any); ok {
		if v, found := data[key]; found {
			return v
		}
	}
	return []any{}
}

// extractGPUListWithMeta returns both items and pagination metadata.
func extractGPUListWithMeta(raw map[string]any) map[string]any {
	data, ok := raw["data"].(map[string]any)
	if !ok {
		return map[string]any{"items": []any{}}
	}
	result := map[string]any{"items": data["data"]}
	if meta, ok := data["meta"].(map[string]any); ok {
		result["meta"] = meta
	}
	return result
}

func setQuery(q map[string]string, key, value string) {
	if value != "" {
		q[key] = value
	}
}

func setBody(body map[string]any, key, value string) {
	if value != "" {
		body[key] = value
	}
}

func setBodyInt(body map[string]any, key string, value int) {
	if value != 0 {
		body[key] = value
	}
}

func setBodyStringArray(body map[string]any, key string, values []string) {
	if len(values) > 0 {
		body[key] = values
	}
}

func intStr(n int) string {
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("%d", n)
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
