package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Huddle01/get-hudl/internal/config"
	"github.com/Huddle01/get-hudl/internal/runtime"
	"github.com/spf13/cobra"
)

var errPrinted = errors.New("error already rendered")

func addGlobalFlags(cmd *cobra.Command, opts *runtime.GlobalOptions) {
	cmd.PersistentFlags().StringVar(&opts.Region, "region", "", "Override the active region")
	cmd.PersistentFlags().StringVar(&opts.Workspace, "workspace", "", "Override the active workspace")
	cmd.PersistentFlags().StringVarP(&opts.Output, "output", "o", "", "Output format: table, json, yaml, wide, name")
	cmd.PersistentFlags().StringVar(&opts.APIKey, "api-key", "", "Override the API key for this invocation")
	cmd.PersistentFlags().DurationVar(&opts.Timeout, "timeout", 30*time.Second, "Request timeout")
	cmd.PersistentFlags().BoolVar(&opts.Verbose, "verbose", false, "Print request diagnostics to stderr")
	cmd.PersistentFlags().BoolVar(&opts.NoColor, "no-color", false, "Disable ANSI color output")
	cmd.PersistentFlags().BoolVar(&opts.Quiet, "quiet", false, "Suppress non-data informational output")
}

func addMutateFlags(cmd *cobra.Command, opts *runtime.MutateOptions, withFile bool) {
	cmd.Flags().StringVar(&opts.IdempotencyKey, "idempotency-key", "", "Idempotency key for safe retries")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Print the resolved request without sending it")
	cmd.Flags().BoolVar(&opts.Interactive, "interactive", false, "Prompt for missing values on a TTY")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip destructive confirmations")
	if withFile {
		cmd.Flags().StringVar(&opts.File, "file", "", "Load request values from a JSON/YAML file or `-` for stdin")
	}
}

func outputMode(app *runtime.App) string {
	if app.Config.Output != "" {
		return app.Config.Output
	}
	if app.IsTTYOut {
		return "table"
	}
	return "json"
}

func renderError(app *runtime.App, err error) error {
	if errors.Is(err, errPrinted) {
		return err
	}

	mode := outputMode(app)
	if httpErr, ok := err.(*runtime.HTTPError); ok && mode == "json" {
		payload := map[string]any{
			"error": map[string]any{
				"message":     httpErr.Message,
				"status_code": httpErr.StatusCode,
			},
		}
		if httpErr.RequestID != "" {
			payload["error"].(map[string]any)["request_id"] = httpErr.RequestID
		}
		if httpErr.Body != nil {
			payload["error"].(map[string]any)["body"] = httpErr.Body
		}
		_ = runtime.PrintValue(app.Stderr, "json", payload)
		return errPrinted
	}

	if mode == "json" {
		_ = runtime.PrintValue(app.Stderr, "json", map[string]any{"error": map[string]any{"message": err.Error()}})
		return errPrinted
	}

	_, _ = fmt.Fprintf(app.Stderr, "Error: %s\n", err.Error())
	return errPrinted
}

func executeResult(app *runtime.App, value any, paging *runtime.Paging) error {
	if paging != nil {
		value = runtime.NormalizeList(value, paging)
	}
	return runtime.PrintValue(app.Stdout, outputMode(app), value)
}

func handleRequest(app *runtime.App, req runtime.Request, extract func(map[string]any) (any, *runtime.Paging, error), mut runtime.MutateOptions) error {
	if mut.DryRun {
		payload := map[string]any{
			"backend":         req.Backend,
			"method":          req.Method,
			"path":            req.Path,
			"query":           req.Query,
			"body":            req.Body,
			"idempotency_key": chooseID(req.IdempotencyKey),
		}
		return runtime.PrintValue(app.Stdout, "json", payload)
	}

	raw, err := app.HTTP.Do(req)
	if err != nil {
		return renderError(app, err)
	}
	value, paging, err := extract(raw)
	if err != nil {
		return renderError(app, err)
	}
	if err := executeResult(app, value, paging); err != nil {
		return renderError(app, err)
	}
	return nil
}

func extractByKey(key string) func(map[string]any) (any, *runtime.Paging, error) {
	return func(raw map[string]any) (any, *runtime.Paging, error) {
		if key == "" {
			return raw, nil, nil
		}
		value, ok := raw[key]
		if !ok {
			return raw, nil, nil
		}
		return value, nil, nil
	}
}

func extractCloudList(key string) func(map[string]any) (any, *runtime.Paging, error) {
	return func(raw map[string]any) (any, *runtime.Paging, error) {
		if value, ok := raw[key]; ok {
			return value, nil, nil
		}
		if data, ok := raw["data"].(map[string]any); ok {
			if value, found := data[key]; found {
				return value, nil, nil
			}
		}
		return []any{}, nil, nil
	}
}

func extractGPUList(raw map[string]any) (any, *runtime.Paging, error) {
	data, ok := raw["data"].(map[string]any)
	if !ok {
		return []any{}, nil, nil
	}
	items := data["data"]
	paging := &runtime.Paging{}
	if meta, ok := data["meta"].(map[string]any); ok {
		paging.Total = meta["total"]
		paging.NextCursor = asString(meta["next_cursor"])
		paging.HasMore = meta["has_more"]
		paging.Page = meta["page"]
		paging.Limit = meta["limit"]
	}
	return items, paging, nil
}

func extractGPUData(raw map[string]any) (any, *runtime.Paging, error) {
	if value, ok := raw["data"]; ok {
		return value, nil, nil
	}
	return raw, nil, nil
}

func regionFromApp(app *runtime.App, required bool) (string, error) {
	if app.Config.Region == "" && required {
		return "", fmt.Errorf("region is required; pass --region or set HUDL_REGION")
	}
	return app.Config.Region, nil
}

func ensureConfirmation(app *runtime.App, mut runtime.MutateOptions, prompt string) error {
	if !mut.Interactive || !app.IsTTYIn {
		if !mut.Yes {
			return fmt.Errorf("%s; re-run with --yes or --interactive", prompt)
		}
		return nil
	}
	answer, err := runtime.PromptBool(app.Stdin, app.Stderr, prompt, nil)
	if err != nil {
		return err
	}
	if !answer {
		return fmt.Errorf("operation cancelled")
	}
	return nil
}

func mustLoadRequest(mut runtime.MutateOptions) (map[string]any, error) {
	if mut.File == "" {
		return map[string]any{}, nil
	}
	return runtime.LoadRequestMap(mut.File)
}

func chooseID(id string) string {
	if id != "" {
		return id
	}
	return "<generated>"
}

func appFromCommand(cmd *cobra.Command) *runtime.App {
	return runtime.FromContext(cmd.Context())
}

func newAppPersistentPreRun(stdin io.Reader, stdout, stderr io.Writer, opts *runtime.GlobalOptions) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		resolved, err := config.Load(config.Flags{
			APIKey:    opts.APIKey,
			Workspace: opts.Workspace,
			Region:    opts.Region,
			Output:    opts.Output,
			CloudBase: opts.CloudBase,
			GPUBase:   opts.GPUBase,
		})
		if err != nil {
			return err
		}
		app := runtime.NewApp(stdin, stdout, stderr, *opts, resolved)
		cmd.SetContext(runtime.WithApp(cmd.Context(), app))
		return nil
	}
}

func asString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func asMap(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	default:
		raw, _ := json.Marshal(typed)
		var decoded map[string]any
		_ = json.Unmarshal(raw, &decoded)
		return decoded
	}
}

func stringSlice(values []string) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func parsePositiveInt(value string, field string) (int, error) {
	number, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number", field)
	}
	if number <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", field)
	}
	return number, nil
}

func normalizeRegions(raw map[string]any) ([]map[string]any, error) {
	rows := make([]map[string]any, 0, len(raw))
	for key, value := range raw {
		rows = append(rows, map[string]any{
			"code":    key,
			"enabled": value,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return asString(rows[i]["code"]) < asString(rows[j]["code"]) })
	return rows, nil
}

func normalizeCloudImageGroups(raw any) []map[string]any {
	items, ok := raw.([]any)
	if !ok {
		return []map[string]any{}
	}
	rows := make([]map[string]any, 0)
	for _, item := range items {
		group := asMap(item)
		distro := asString(group["distro"])
		if versions, ok := group["versions"].([]any); ok {
			for _, version := range versions {
				v := asMap(version)
				rows = append(rows, map[string]any{
					"distro":  distro,
					"id":      v["id"],
					"version": v["version"],
				})
			}
		}
	}
	return rows
}

func stringArrayFlag(cmd *cobra.Command, name string) []string {
	values, _ := cmd.Flags().GetStringArray(name)
	return values
}

func setMapString(body map[string]any, key, value string) {
	if strings.TrimSpace(value) != "" {
		body[key] = strings.TrimSpace(value)
	}
}

func setMapStringArray(body map[string]any, key string, values []string) {
	if len(values) == 0 {
		return
	}
	body[key] = values
}

func setMapInt(body map[string]any, key string, value int) {
	if value != 0 {
		body[key] = value
	}
}

func setMapBool(body map[string]any, key string, value *bool) {
	if value != nil {
		body[key] = *value
	}
}

func cloneRequest(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	clone := make(map[string]any, len(input))
	for key, value := range input {
		clone[key] = value
	}
	return clone
}

func hasValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(typed) != ""
	case []string:
		return len(typed) > 0
	case []any:
		return len(typed) > 0
	case int:
		return typed != 0
	case int64:
		return typed != 0
	case float64:
		return typed != 0
	default:
		return true
	}
}

func stringList(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text := asString(item)
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func setQuery(query map[string]string, key, value string) {
	if strings.TrimSpace(value) != "" {
		query[key] = strings.TrimSpace(value)
	}
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func intString(value int) string {
	if value == 0 {
		return ""
	}
	return strconv.Itoa(value)
}

func completeCloudResource(path string, key string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		cfg, err := config.Load(config.Flags{})
		if err != nil || cfg.APIKey == "" || cfg.Region == "" {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		app := runtime.NewApp(nil, io.Discard, io.Discard, runtime.GlobalOptions{Timeout: 10 * time.Second}, cfg)
		raw, err := app.HTTP.Do(runtime.Request{
			Backend: runtime.BackendCloud,
			Method:  "GET",
			Path:    path,
			Query:   map[string]string{"region": cfg.Region},
		})
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		items, _, err := extractCloudList(key)(raw)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		var completions []string
		for _, item := range completeItems(items) {
			id := firstMatch(item, "id", "name")
			if id != "" {
				completions = append(completions, id)
			}
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

func completeItems(items any) []map[string]any {
	switch typed := items.(type) {
	case []any:
		out := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, asMap(item))
		}
		return out
	case []map[string]any:
		return typed
	default:
		if items == nil {
			return nil
		}
		return []map[string]any{asMap(items)}
	}
}

func firstMatch(row map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := row[key]; ok {
			text := asString(value)
			if text != "" {
				return text
			}
		}
	}
	return ""
}
