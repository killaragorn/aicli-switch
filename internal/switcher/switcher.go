package switcher

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/killaragorn/aicli-switch/internal/config"
	"github.com/killaragorn/aicli-switch/internal/crypto"
	"github.com/killaragorn/aicli-switch/internal/profile"
	"github.com/killaragorn/aicli-switch/internal/token"
)

// Switch switches to the specified profile.
func Switch(name string) error {
	// Verify target profile exists
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

	// Update active profile
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

	dir := config.ProfileDir(name)
	keyPath := filepath.Join(dir, config.AuthKeyFileName)
	filePath := filepath.Join(dir, config.AuthFileName)

	plaintext, err := crypto.DecryptAuthFile(keyPath, filePath)
	if err != nil {
		return fmt.Errorf("decrypt: %w", err)
	}

	var tokens token.AuthTokens
	if err := json.Unmarshal(plaintext, &tokens); err != nil {
		return fmt.Errorf("parse tokens: %w", err)
	}

	fmt.Printf("Refreshing token for %q...\n", name)
	resp, err := token.RefreshToken(tokens.RefreshToken)
	if err != nil {
		return fmt.Errorf("refresh failed: %w\nYou may need to run 'claude login' and re-add this profile", err)
	}

	// Update tokens
	tokens.AccessToken = resp.AccessToken
	if resp.RefreshToken != "" {
		tokens.RefreshToken = resp.RefreshToken
	}

	newPlaintext, _ := json.Marshal(tokens)
	if err := crypto.EncryptAuthFile(keyPath, filePath, newPlaintext); err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}

	exp := token.GetExpiry(tokens.AccessToken)
	fmt.Printf("Token refreshed. New expiry: %s\n", exp.Local().Format("2006-01-02 15:04:05"))

	// Update email in profile if changed
	email := token.GetEmail(tokens.AccessToken)
	if email != "" && email != p.Email {
		p.Email = email
		data, _ := json.MarshalIndent(p, "", "  ")
		os.WriteFile(filepath.Join(dir, config.ProfileFileName), data, 0600)
	}

	return nil
}

func ensureValidToken(name string) error {
	dir := config.ProfileDir(name)
	keyPath := filepath.Join(dir, config.AuthKeyFileName)
	filePath := filepath.Join(dir, config.AuthFileName)

	plaintext, err := crypto.DecryptAuthFile(keyPath, filePath)
	if err != nil {
		return fmt.Errorf("decrypt: %w", err)
	}

	var tokens token.AuthTokens
	if err := json.Unmarshal(plaintext, &tokens); err != nil {
		return fmt.Errorf("parse tokens: %w", err)
	}

	if !token.IsExpired(tokens.AccessToken) {
		return nil // Token still valid
	}

	fmt.Printf("Token expired, refreshing...\n")
	return RefreshProfile(name)
}

func deployOAuth(name string) error {
	dir := config.ProfileDir(name)

	if err := config.EnsureDir(config.FactoryDir()); err != nil {
		return err
	}

	// Copy auth files to ~/.factory/
	if err := copyFile(
		filepath.Join(dir, config.AuthFileName),
		config.FactoryAuthFile(),
	); err != nil {
		return fmt.Errorf("copy auth file: %w", err)
	}

	if err := copyFile(
		filepath.Join(dir, config.AuthKeyFileName),
		config.FactoryAuthKey(),
	); err != nil {
		return fmt.Errorf("copy auth key: %w", err)
	}

	// Clear API key from settings if present (switching to OAuth)
	clearAPIKeyFromSettings()

	return nil
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

	dir := config.ProfileDir(name)

	switch p.Type {
	case "oauth":
		// Backup current ~/.factory/ files
		copyFile(config.FactoryAuthFile(), filepath.Join(dir, config.AuthFileName))
		copyFile(config.FactoryAuthKey(), filepath.Join(dir, config.AuthKeyFileName))
	case "apikey":
		// API key profiles don't need backup from live state
	}
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0600)
}
