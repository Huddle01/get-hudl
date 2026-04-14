package runtime

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadRequestMap(path string) (map[string]any, error) {
	if path == "" {
		return map[string]any{}, nil
	}

	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, err
	}

	var out map[string]any
	if json.Valid(data) {
		if err := json.Unmarshal(data, &out); err != nil {
			return nil, err
		}
		return out, nil
	}

	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func MergeRequest(base map[string]any, overlays ...map[string]any) map[string]any {
	if base == nil {
		base = map[string]any{}
	}
	for _, overlay := range overlays {
		for key, value := range overlay {
			if value == nil {
				continue
			}
			existingMap, okExisting := base[key].(map[string]any)
			newMap, okNew := value.(map[string]any)
			if okExisting && okNew {
				base[key] = MergeRequest(existingMap, newMap)
				continue
			}
			base[key] = value
		}
	}
	return base
}

func PromptString(in io.Reader, out io.Writer, label string, current string, required bool) (string, error) {
	reader := bufio.NewReader(in)
	if current != "" {
		fmt.Fprintf(out, "%s [%s]: ", label, current)
	} else {
		fmt.Fprintf(out, "%s: ", label)
	}
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		value = current
	}
	if required && value == "" {
		return "", fmt.Errorf("%s is required", label)
	}
	return value, nil
}

func PromptBool(in io.Reader, out io.Writer, label string, current *bool) (bool, error) {
	reader := bufio.NewReader(in)
	suffix := "y/N"
	if current != nil && *current {
		suffix = "Y/n"
	}
	fmt.Fprintf(out, "%s [%s]: ", label, suffix)
	value, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" && current != nil {
		return *current, nil
	}
	return value == "y" || value == "yes" || value == "true", nil
}

func PromptCSV(in io.Reader, out io.Writer, label string, current []string, required bool) ([]string, error) {
	joined := strings.Join(current, ",")
	value, err := PromptString(in, out, label, joined, required)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(value, ",")
	var outParts []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			outParts = append(outParts, part)
		}
	}
	if required && len(outParts) == 0 {
		return nil, fmt.Errorf("%s is required", label)
	}
	return outParts, nil
}

func MustJSON(data any) string {
	buf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buf)
	enc.SetIndent("", "  ")
	_ = enc.Encode(data)
	return strings.TrimSpace(buf.String())
}
