package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/octobit/heimdall-cli/internal/config"
)

func TestLoadFromEnvVars(t *testing.T) {
	t.Setenv("HEIMDALL_API_KEY", "hm_read_testkey")
	t.Setenv("HEIMDALL_BASE_URL", "https://api.example.com")

	profile, err := config.Load("", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if profile.APIKey != "hm_read_testkey" {
		t.Errorf("expected api key from env, got %q", profile.APIKey)
	}
	if profile.BaseURL != "https://api.example.com" {
		t.Errorf("expected base url from env, got %q", profile.BaseURL)
	}
}

func TestLoadFromConfigFile(t *testing.T) {
	dir := t.TempDir()
	cfgContent := `
default_profile: myapp
profiles:
  myapp:
    api_key: hm_read_from_file
    base_url: https://api.myapp.com
`
	cfgPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgPath, []byte(cfgContent), 0600)

	profile, err := config.Load(cfgPath, "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if profile.APIKey != "hm_read_from_file" {
		t.Errorf("expected api key from file, got %q", profile.APIKey)
	}
}

func TestLoadSpecificProfile(t *testing.T) {
	dir := t.TempDir()
	cfgContent := `
default_profile: myapp
profiles:
  myapp:
    api_key: hm_read_myapp
    base_url: https://api.myapp.com
  other:
    api_key: hm_read_other
    base_url: https://api.other.com
`
	cfgPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgPath, []byte(cfgContent), 0600)

	profile, err := config.Load(cfgPath, "other")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if profile.APIKey != "hm_read_other" {
		t.Errorf("expected other profile key, got %q", profile.APIKey)
	}
}

func TestLoadMissingProfile(t *testing.T) {
	dir := t.TempDir()
	cfgContent := `
profiles:
  myapp:
    api_key: hm_read_myapp
    base_url: https://api.myapp.com
`
	cfgPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgPath, []byte(cfgContent), 0600)

	_, err := config.Load(cfgPath, "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing profile")
	}
}

func TestEnvVarTakesPrecedenceOverFile(t *testing.T) {
	t.Setenv("HEIMDALL_API_KEY", "hm_read_fromenv")
	dir := t.TempDir()
	cfgContent := `
profiles:
  default:
    api_key: hm_read_fromfile
    base_url: https://api.example.com
`
	cfgPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgPath, []byte(cfgContent), 0600)

	profile, err := config.Load(cfgPath, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.APIKey != "hm_read_fromenv" {
		t.Errorf("env var should take precedence, got %q", profile.APIKey)
	}
}
