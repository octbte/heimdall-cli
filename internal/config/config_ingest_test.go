package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/octobit/heimdall-cli/internal/config"
)

func TestLoadIngestAPIKeyFromEnv(t *testing.T) {
	t.Setenv("HEIMDALL_API_KEY", "hm_read_test")
	t.Setenv("HEIMDALL_INGEST_API_KEY", "hm_live_envkey")
	t.Setenv("HEIMDALL_BASE_URL", "https://example.com")

	p, err := config.Load("", "")
	if err != nil {
		t.Fatal(err)
	}
	if p.IngestAPIKey != "hm_live_envkey" {
		t.Errorf("got %q, want hm_live_envkey", p.IngestAPIKey)
	}
}

func TestLoadIngestAPIKeyFromFile(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	cfgContent := `
default_profile: default
profiles:
  default:
    api_key: hm_read_test
    base_url: https://example.com
    ingest_api_key: hm_live_fromfile
`
	if err := os.WriteFile(cfgFile, []byte(cfgContent), 0600); err != nil {
		t.Fatal(err)
	}

	p, err := config.Load(cfgFile, "")
	if err != nil {
		t.Fatal(err)
	}
	if p.IngestAPIKey != "hm_live_fromfile" {
		t.Errorf("got %q, want hm_live_fromfile", p.IngestAPIKey)
	}
}

func TestSaveAndLoadIngestAPIKey(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")

	err := config.Save(cfgFile, "myproj", config.Profile{
		APIKey:       "hm_read_test",
		BaseURL:      "https://example.com",
		IngestAPIKey: "hm_live_saved",
	})
	if err != nil {
		t.Fatal(err)
	}

	p, err := config.Load(cfgFile, "myproj")
	if err != nil {
		t.Fatal(err)
	}
	if p.IngestAPIKey != "hm_live_saved" {
		t.Errorf("got %q, want hm_live_saved", p.IngestAPIKey)
	}
}

func TestListProfilesIncludesIngestKey(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	cfgContent := `
default_profile: default
profiles:
  default:
    api_key: hm_read_test
    base_url: https://example.com
    ingest_api_key: hm_live_listed
`
	if err := os.WriteFile(cfgFile, []byte(cfgContent), 0600); err != nil {
		t.Fatal(err)
	}

	profiles, _, err := config.ListProfiles(cfgFile)
	if err != nil {
		t.Fatal(err)
	}
	if profiles["default"].IngestAPIKey != "hm_live_listed" {
		t.Errorf("got %q, want hm_live_listed", profiles["default"].IngestAPIKey)
	}
}
