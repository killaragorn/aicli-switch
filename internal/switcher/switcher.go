package switcher

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/killaragorn/aicli-switch/internal/config"
	"github.com/killaragorn/aicli-switch/internal/profile"
	"github.com/killaragorn/aicli-switch/internal/token"
)

// Switch switches to the specified profile.
func Switch(name string) error {
	target, err := profile.Get(name)
	if err != nil {
		return err
	}

	// Backup current credentials to active profile (if any)
	active := profile.GetActive()
	if active != "" && active != name {
		backupCurrent(active)
	}

	// For OAuth profiles, check and refresh token if needed
	if target.Type == "oauth" {
		if err := ensureValidToken(name); err != nil {
			return fmt.Errorf("token validation: %w", err)
		}
	}

	// Deploy credentials
	switch target.Type {
	case "oauth":
		if err := deployOAuth(name); err != nil {
			return fmt.Errorf("deploy oauth: %w", err)
		}
	case "apikey":
		if err := deployAPIKey(name); err != nil {
			return fmt.Errorf("deploy apikey: %w", err)
		}
	}

	profile.SaveActive(name)
	profile.UpdateLastSwitched(name)

	fmt.Printf("Switched to profile %q (%s)\n", name, target.Type)
	if target.Email != "" {
		fmt.Printf("  Email: %s\n", target.Email)
	}
	return nil
}

// RefreshProfile refreshes the OAuth token for a profile.
func RefreshProfile(name string) error {
	p, err := profile.Get(name)
	if err != nil {
		return err
	}
	if p.Type != "oauth" {
		return fmt.Errorf("profile %q is not OAuth type", name)
	}

	oauth, err := profile.ReadProfileOAuth(name)
	if err != nil {
		return err
	}

	fmt.Printf("Refreshing token for %q...\n", name)
	resp, err := token.RefreshOAuthToken(oauth.RefreshToken)
	if err != nil {
		return fmt.Errorf("refresh failed: %w\nYou may need to run 'claude login' and re-add this profile", err)
	}

	// Update tokens
	oauth.AccessToken = resp.AccessToken
	if resp.RefreshToken != "" {
		oauth.RefreshToken = resp.RefreshToken
	}
	if resp.ExpiresIn > 0 {
		oauth.ExpiresAt = time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second).UnixMilli()
	}

	if err := profile.SaveProfileOAuth(name, oauth); err != nil {
		return fmt.Errorf("save refreshed token: %w", err)
	}

	exp := token.GetExpiryFromData(oauth)
	fmt.Printf("Token refreshed. New expiry: %s\n", exp.Local().Format("2006-01-02 15:04:05"))

	// Update email in profile if changed
	email := token.GetEmail(oauth.AccessToken)
	if email != "" && email != p.Email {
		p.Email = email
		data, _ := json.MarshalIndent(p, "", "  ")
		os.WriteFile(filepath.Join(config.ProfileDir(name), config.ProfileFileName), data, 0600)
	}

	return nil
}

func ensureValidToken(name string) error {
	oauth, err := profile.ReadProfileOAuth(name)
	if err != nil {
		return err
	}

	if !token.IsExpiredData(oauth) {
		return nil
	}

	fmt.Printf("Token expired, refreshing...\n")
	return RefreshProfile(name)
}

func deployOAuth(name string) error {
	oauth, err := profile.ReadProfileOAuth(name)
	if err != nil {
		return err
	}

	// Write claudeAiOauth to ~/.claude/.credentials.json, preserving mcpOAuth
	return profile.WriteCredentialsOAuth(oauth)
}

func deployAPIKey(name string) error {
	dir := config.ProfileDir(name)

	envData, err := os.ReadFile(filepath.Join(dir, config.SettingsEnvName))
	if err != nil {
		return fmt.Errorf("read env settings: %w", err)
	}

	var env profile.EnvSettings
	if err := json.Unmarshal(envData, &env); err != nil {
		return fmt.Errorf("parse env settings: %w", err)
	}

	return mergeEnvToSettings(env)
}

func mergeEnvToSettings(env profile.EnvSettings) error {
	settingsPath := config.ClaudeSettingsPath()

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return fmt.Errorf("read settings: %w", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("parse settings: %w", err)
	}

	envMap, ok := settings["env"].(map[string]any)
	if !ok {
		envMap = make(map[string]any)
	}

	if env.APIKey != "" {
		envMap["ANTHROPIC_API_KEY"] = env.APIKey
	}
	if env.BaseURL != "" {
		envMap["ANTHROPIC_BASE_URL"] = env.BaseURL
	}

	settings["env"] = envMap

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	return os.WriteFile(settingsPath, out, 0600)
}

func clearAPIKeyFromSettings() {
	settingsPath := config.ClaudeSettingsPath()
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return
	}

	var settings map[string]any
	if json.Unmarshal(data, &settings) != nil {
		return
	}

	envMap, ok := settings["env"].(map[string]any)
	if !ok {
		return
	}

	delete(envMap, "ANTHROPIC_API_KEY")
	delete(envMap, "ANTHROPIC_BASE_URL")
	settings["env"] = envMap

	out, _ := json.MarshalIndent(settings, "", "  ")
	os.WriteFile(settingsPath, out, 0600)
}

func backupCurrent(name string) {
	p, err := profile.Get(name)
	if err != nil {
		return
	}

	if p.Type == "oauth" {
		// Backup current ~/.claude/.credentials.json oauth data to profile
		oauth, err := profile.ReadCredentialsOAuth()
		if err != nil {
			return
		}
		profile.SaveProfileOAuth(name, oauth)
	}
}
