package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

const (
	defaultCloudBaseURL = "https://cloud.huddleapis.com/api/v1"
	defaultGPUBaseURL   = "https://gpu.huddleapis.com/api/v1"
)

type APIConfig struct {
	CloudBaseURL string `toml:"cloud_base_url" json:"cloud_base_url,omitempty"`
	GPUBaseURL   string `toml:"gpu_base_url" json:"gpu_base_url,omitempty"`
}

type DefaultsConfig struct {
	VM         map[string]any `toml:"vm" json:"vm,omitempty"`
	Volume     map[string]any `toml:"volume" json:"volume,omitempty"`
	SG         map[string]any `toml:"sg" json:"sg,omitempty"`
	GPU        map[string]any `toml:"gpu" json:"gpu,omitempty"`
	GPUVolume  map[string]any `toml:"gpu_volume" json:"gpu_volume,omitempty"`
	GPUWebhook map[string]any `toml:"gpu_webhook" json:"gpu_webhook,omitempty"`
}

type File struct {
	APIKey    string         `toml:"api_key" json:"api_key,omitempty"`
	Workspace string         `toml:"workspace" json:"workspace,omitempty"`
	Region    string         `toml:"region" json:"region,omitempty"`
	Output    string         `toml:"output" json:"output,omitempty"`
	API       APIConfig      `toml:"api" json:"api,omitempty"`
	Defaults  DefaultsConfig `toml:"defaults" json:"defaults,omitempty"`
}

type Env struct {
	APIKey    string
	Workspace string
	Region    string
	Output    string
	CloudBase string
	GPUBase   string
}

type Flags struct {
	APIKey    string
	Workspace string
	Region    string
	Output    string
	CloudBase string
	GPUBase   string
}

type Resolved struct {
	APIKey      string
	Workspace   string
	Region      string
	Output      string
	CloudBase   string
	GPUBase     string
	Defaults    DefaultsConfig
	UserPath    string
	ProjectPath string
}

func Load(flags Flags) (Resolved, error) {
	userPath, err := UserConfigPath()
	if err != nil {
		return Resolved{}, err
	}

	projectPath, err := ProjectConfigPath()
	if err != nil {
		return Resolved{}, err
	}

	userCfg, err := loadFile(userPath)
	if err != nil {
		return Resolved{}, err
	}

	projectCfg, err := loadFile(projectPath)
	if err != nil {
		return Resolved{}, err
	}

	env := readEnv()
	resolved := Resolved{
		CloudBase:   defaultCloudBaseURL,
		GPUBase:     defaultGPUBaseURL,
		UserPath:    userPath,
		ProjectPath: projectPath,
	}

	mergeResolved(&resolved, userCfg)
	mergeResolved(&resolved, projectCfg)
	mergeResolved(&resolved, fileFromEnv(env))
	mergeResolved(&resolved, fileFromFlags(flags))

	return resolved, nil
}

func UserConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".hudl", "config.toml"), nil
}

func ProjectConfigPath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(wd, "hudl.toml"), nil
}

func SaveUserConfig(update func(*File) error) error {
	path, err := UserConfigPath()
	if err != nil {
		return err
	}

	cfg, err := loadFile(path)
	if err != nil {
		return err
	}

	if err := update(&cfg); err != nil {
		return err
	}

	if cfg.API.CloudBaseURL == "" {
		cfg.API.CloudBaseURL = defaultCloudBaseURL
	}
	if cfg.API.GPUBaseURL == "" {
		cfg.API.GPUBaseURL = defaultGPUBaseURL
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

func ClearUserAuth() error {
	return SaveUserConfig(func(cfg *File) error {
		cfg.APIKey = ""
		return nil
	})
}

func loadFile(path string) (File, error) {
	var cfg File
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse %s: %w", path, err)
	}

	return cfg, nil
}

func readEnv() Env {
	return Env{
		APIKey:    strings.TrimSpace(os.Getenv("HUDL_API_KEY")),
		Workspace: strings.TrimSpace(os.Getenv("HUDL_WORKSPACE")),
		Region:    strings.TrimSpace(os.Getenv("HUDL_REGION")),
		Output:    strings.TrimSpace(os.Getenv("HUDL_OUTPUT")),
		CloudBase: strings.TrimSpace(os.Getenv("HUDL_CLOUD_BASE_URL")),
		GPUBase:   strings.TrimSpace(os.Getenv("HUDL_GPU_BASE_URL")),
	}
}

func fileFromEnv(env Env) File {
	return File{
		APIKey:    env.APIKey,
		Workspace: env.Workspace,
		Region:    env.Region,
		Output:    env.Output,
		API: APIConfig{
			CloudBaseURL: env.CloudBase,
			GPUBaseURL:   env.GPUBase,
		},
	}
}

func fileFromFlags(flags Flags) File {
	return File{
		APIKey:    flags.APIKey,
		Workspace: flags.Workspace,
		Region:    flags.Region,
		Output:    flags.Output,
		API: APIConfig{
			CloudBaseURL: flags.CloudBase,
			GPUBaseURL:   flags.GPUBase,
		},
	}
}

func mergeResolved(resolved *Resolved, cfg File) {
	if cfg.APIKey != "" {
		resolved.APIKey = cfg.APIKey
	}
	if cfg.Workspace != "" {
		resolved.Workspace = cfg.Workspace
	}
	if cfg.Region != "" {
		resolved.Region = cfg.Region
	}
	if cfg.Output != "" {
		resolved.Output = cfg.Output
	}
	if cfg.API.CloudBaseURL != "" {
		resolved.CloudBase = cfg.API.CloudBaseURL
	}
	if cfg.API.GPUBaseURL != "" {
		resolved.GPUBase = cfg.API.GPUBaseURL
	}
	if len(cfg.Defaults.VM) > 0 {
		resolved.Defaults.VM = cloneMap(cfg.Defaults.VM)
	}
	if len(cfg.Defaults.Volume) > 0 {
		resolved.Defaults.Volume = cloneMap(cfg.Defaults.Volume)
	}
	if len(cfg.Defaults.SG) > 0 {
		resolved.Defaults.SG = cloneMap(cfg.Defaults.SG)
	}
	if len(cfg.Defaults.GPU) > 0 {
		resolved.Defaults.GPU = cloneMap(cfg.Defaults.GPU)
	}
	if len(cfg.Defaults.GPUVolume) > 0 {
		resolved.Defaults.GPUVolume = cloneMap(cfg.Defaults.GPUVolume)
	}
	if len(cfg.Defaults.GPUWebhook) > 0 {
		resolved.Defaults.GPUWebhook = cloneMap(cfg.Defaults.GPUWebhook)
	}
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
