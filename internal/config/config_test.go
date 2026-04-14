package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPrecedence(t *testing.T) {
	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", homeDir)
	t.Setenv("HUDL_REGION", "env-region")
	t.Setenv("HUDL_OUTPUT", "yaml")

	if err := os.WriteFile(filepath.Join(projectDir, "hudl.toml"), []byte("workspace = \"project-ws\"\nregion = \"project-region\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := SaveUserConfig(func(cfg *File) error {
		cfg.APIKey = "user-key"
		cfg.Workspace = "user-ws"
		cfg.Region = "user-region"
		cfg.Output = "json"
		cfg.API.CloudBaseURL = "https://cloud.example.test"
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(cwd) }()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}

	resolved, err := Load(Flags{
		Region:    "flag-region",
		Workspace: "flag-ws",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resolved.APIKey != "user-key" {
		t.Fatalf("expected api key from user config, got %q", resolved.APIKey)
	}
	if resolved.Workspace != "flag-ws" {
		t.Fatalf("expected workspace from flags, got %q", resolved.Workspace)
	}
	if resolved.Region != "flag-region" {
		t.Fatalf("expected region from flags, got %q", resolved.Region)
	}
	if resolved.Output != "yaml" {
		t.Fatalf("expected output from env, got %q", resolved.Output)
	}
	if resolved.CloudBase != "https://cloud.example.test" {
		t.Fatalf("expected cloud base from user config, got %q", resolved.CloudBase)
	}
}

func TestSaveUserConfigPermissions(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	if err := SaveUserConfig(func(cfg *File) error {
		cfg.APIKey = "test-key"
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	path, err := UserConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("expected permissions 0600, got %#o", got)
	}
}
