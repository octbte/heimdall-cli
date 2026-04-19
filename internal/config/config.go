package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Profile holds credentials for one Heimdall project.
type Profile struct {
	APIKey  string
	BaseURL string
}

// Load resolves credentials using this priority order:
//  1. HEIMDALL_API_KEY / HEIMDALL_BASE_URL env vars
//  2. profileName from cfgFile (or default_profile if profileName is "")
//
// cfgFile may be "" to use the default path (~/.heimdall/config.yaml).
func Load(cfgFile, profileName string) (*Profile, error) {
	if key := os.Getenv("HEIMDALL_API_KEY"); key != "" {
		baseURL := os.Getenv("HEIMDALL_BASE_URL")
		if baseURL == "" {
			baseURL = "https://api.heimdall.io"
		}
		return &Profile{APIKey: key, BaseURL: baseURL}, nil
	}

	v := viper.New()
	v.SetConfigType("yaml")

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot determine home directory: %w", err)
		}
		v.SetConfigFile(home + "/.heimdall/config.yaml")
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("no config found — run 'heimdall configure' to get started")
	}

	if profileName == "" {
		profileName = v.GetString("default_profile")
	}
	if profileName == "" {
		profileName = "default"
	}

	apiKey := v.GetString(fmt.Sprintf("profiles.%s.api_key", profileName))
	if apiKey == "" {
		return nil, fmt.Errorf("profile %q not found — run 'heimdall configure' to add it", profileName)
	}

	return &Profile{
		APIKey:  apiKey,
		BaseURL: v.GetString(fmt.Sprintf("profiles.%s.base_url", profileName)),
	}, nil
}

// Save writes a profile to the config file.
func Save(cfgFile, profileName string, profile Profile) error {
	v := viper.New()
	v.SetConfigType("yaml")

	if cfgFile == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		dir := home + "/.heimdall"
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("cannot create config directory: %w", err)
		}
		cfgFile = dir + "/config.yaml"
	}

	v.SetConfigFile(cfgFile)
	v.ReadInConfig() // ignore error — file may not exist yet

	v.Set(fmt.Sprintf("profiles.%s.api_key", profileName), profile.APIKey)
	v.Set(fmt.Sprintf("profiles.%s.base_url", profileName), profile.BaseURL)

	if v.GetString("default_profile") == "" {
		v.Set("default_profile", profileName)
	}

	return v.WriteConfigAs(cfgFile)
}

// ListProfiles returns all profile names from the config file.
func ListProfiles(cfgFile string) (map[string]Profile, string, error) {
	v := viper.New()
	v.SetConfigType("yaml")

	if cfgFile == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		v.SetConfigFile(home + "/.heimdall/config.yaml")
	} else {
		v.SetConfigFile(cfgFile)
	}

	if err := v.ReadInConfig(); err != nil {
		return map[string]Profile{}, "", nil
	}

	profiles := map[string]Profile{}
	raw := v.GetStringMap("profiles")
	for name := range raw {
		profiles[name] = Profile{
			APIKey:  v.GetString(fmt.Sprintf("profiles.%s.api_key", name)),
			BaseURL: v.GetString(fmt.Sprintf("profiles.%s.base_url", name)),
		}
	}

	return profiles, v.GetString("default_profile"), nil
}
