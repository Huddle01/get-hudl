package runtime

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"gopkg.in/yaml.v3"
)

type Paging struct {
	Total      any    `json:"total,omitempty" yaml:"total,omitempty"`
	NextCursor string `json:"next_cursor,omitempty" yaml:"next_cursor,omitempty"`
	HasMore    any    `json:"has_more,omitempty" yaml:"has_more,omitempty"`
	Page       any    `json:"page,omitempty" yaml:"page,omitempty"`
	Limit      any    `json:"limit,omitempty" yaml:"limit,omitempty"`
}

func NormalizeList(items any, paging *Paging) map[string]any {
	out := map[string]any{"items": items}
	if paging != nil {
		out["paging"] = paging
	}
	return out
}

func PrintValue(w io.Writer, mode string, value any) error {
	switch mode {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(value)
	case "yaml":
		data, err := yaml.Marshal(value)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	case "name":
		return printName(w, value)
	case "table", "wide", "":
		return printTable(w, value)
	default:
		return fmt.Errorf("unsupported output mode %q", mode)
	}
}

func printName(w io.Writer, value any) error {
	items, ok := value.(map[string]any)
	if ok {
		if rawItems, found := items["items"]; found {
			if list, okList := rawItems.([]map[string]any); okList {
				for _, item := range list {
					fmt.Fprintln(w, item["id"])
				}
				return nil
			}
			if generic, okGeneric := rawItems.([]any); okGeneric {
				for _, item := range generic {
					row := objectToMap(item)
					fmt.Fprintln(w, firstNonEmpty(row, "id", "name"))
				}
				return nil
			}
		}
	}

	row := objectToMap(value)
	fmt.Fprintln(w, firstNonEmpty(row, "id", "name"))
	return nil
}

func printTable(w io.Writer, value any) error {
	if listWrapper, ok := value.(map[string]any); ok {
		if items, found := listWrapper["items"]; found {
			return printRows(w, normalizeItems(items))
		}
	}
	return printObject(w, objectToMap(value))
}

func printRows(w io.Writer, rows []map[string]any) error {
	if len(rows) == 0 {
		_, err := fmt.Fprintln(w, "No results")
		return err
	}

	headers := collectHeaders(rows)
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(headers, "\t"))
	for _, row := range rows {
		values := make([]string, 0, len(headers))
		for _, header := range headers {
			values = append(values, scalarString(row[header]))
		}
		fmt.Fprintln(tw, strings.Join(values, "\t"))
	}
	return tw.Flush()
}

func printObject(w io.Writer, row map[string]any) error {
	headers := collectHeaders([]map[string]any{row})
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	for _, header := range headers {
		fmt.Fprintf(tw, "%s\t%s\n", header, scalarString(row[header]))
	}
	return tw.Flush()
}

func normalizeItems(items any) []map[string]any {
	switch typed := items.(type) {
	case []map[string]any:
		return typed
	case []any:
		out := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, objectToMap(item))
		}
		return out
	default:
		return []map[string]any{objectToMap(items)}
	}
}

func objectToMap(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return flattenMap("", typed)
	default:
		data, _ := json.Marshal(typed)
		var decoded map[string]any
		_ = json.Unmarshal(data, &decoded)
		return flattenMap("", decoded)
	}
}

func flattenMap(prefix string, input map[string]any) map[string]any {
	out := make(map[string]any)
	for key, value := range input {
		combined := key
		if prefix != "" {
			combined = prefix + "." + key
		}
		nested, ok := value.(map[string]any)
		if ok {
			for nestedKey, nestedValue := range flattenMap(combined, nested) {
				out[nestedKey] = nestedValue
			}
			continue
		}
		out[combined] = value
	}
	return out
}

func collectHeaders(rows []map[string]any) []string {
	seen := map[string]bool{}
	priority := []string{"id", "name", "status", "region.name", "region", "created_at", "updated_at", "floating_ip", "ip", "hostname"}
	var headers []string
	for _, key := range priority {
		for _, row := range rows {
			if _, ok := row[key]; ok && !seen[key] {
				seen[key] = true
				headers = append(headers, key)
			}
		}
	}
	var tail []string
	for _, row := range rows {
		for key := range row {
			if !seen[key] {
				seen[key] = true
				tail = append(tail, key)
			}
		}
	}
	sort.Strings(tail)
	return append(headers, tail...)
}

func scalarString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case []string:
		return strings.Join(typed, ",")
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, scalarString(item))
		}
		return strings.Join(parts, ",")
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprint(typed)
		}
		return string(data)
	}
}

func firstNonEmpty(row map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := row[key]; ok {
			text := scalarString(value)
			if text != "" {
				return text
			}
		}
	}
	return ""
}
