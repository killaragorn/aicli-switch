package profile

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/killaragorn/aicli-switch/internal/config"
	"github.com/killaragorn/aicli-switch/internal/crypto"
	"github.com/killaragorn/aicli-switch/internal/token"
)

type Profile struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "oauth" or "apikey"
	Email       string `json:"email,omitempty"`
	CreatedAt   string `json:"created_at"`
	LastSwitched string `json:"last_switched,omitempty"`
}

type EnvSettings struct {
	APIKey  string `json:"ANTHROPIC_API_KEY,omitempty"`
	BaseURL string `json:"ANTHROPIC_BASE_URL,omitempty"`
}

type ProfileInfo struct {
	Profile
	TokenExpiry time.Time
	IsExpired   bool
	IsActive    bool
}

// Add creates a new profile from the current Claude Code state.
func Add(name, profileType string) error {
	dir := config.ProfileDir(name)
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("profile %q already exists", name)
	}

	if err := config.EnsureDir(dir); err != nil {
		return fmt.Errorf("create profile dir: %w", err)
	}

	p := Profile{
		Name:      name,
		Type:      profileType,
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	switch profileType {
	case "oauth":
		// Copy auth files from ~/.factory/
		authFile := config.FactoryAuthFile()
		authKey := config.FactoryAuthKey()

		if _, err := os.Stat(authFile); os.IsNotExist(err) {
			os.RemoveAll(dir)
			return fmt.Errorf("no OAuth credentials found at %s\nPlease run 'claude login' first", authFile)
		}

		if err := copyFile(authFile, filepath.Join(dir, config.AuthFileName)); err != nil {
			os.RemoveAll(dir)
			return fmt.Errorf("copy auth file: %w", err)
		}
		if err := copyFile(authKey, filepath.Join(dir, config.AuthKeyFileName)); err != nil {
			os.RemoveAll(dir)
			return fmt.Errorf("copy auth key: %w", err)
		}

		// Extract email from JWT
		plaintext, err := crypto.DecryptAuthFile(
			filepath.Join(dir, config.AuthKeyFileName),
			filepath.Join(dir, config.AuthFileName),
		)
		if err == nil {
			var tokens token.AuthTokens
			if json.Unmarshal(plaintext, &tokens) == nil {
				p.Email = token.GetEmail(tokens.AccessToken)
			}
		}

	case "apikey":
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("API Key: ")
		apiKey, _ := reader.ReadString('\n')
		apiKey = strings.TrimSpace(apiKey)

		fmt.Print("Base URL (leave empty for default): ")
		baseURL, _ := reader.ReadString('\n')
		baseURL = strings.TrimSpace(baseURL)
		if baseURL == "" {
			baseURL = "https://api.anthropic.com/"
		}

		env := EnvSettings{APIKey: apiKey, BaseURL: baseURL}
		envData, _ := json.MarshalIndent(env, "", "  ")
		if err := os.WriteFile(filepath.Join(dir, config.SettingsEnvName), envData, 0600); err != nil {
			os.RemoveAll(dir)
			return fmt.Errorf("write env settings: %w", err)
		}

	default:
		os.RemoveAll(dir)
		return fmt.Errorf("unknown profile type: %s (use 'oauth' or 'apikey')", profileType)
	}

	profileData, _ := json.MarshalIndent(p, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, config.ProfileFileName), profileData, 0600); err != nil {
		return fmt.Errorf("write profile: %w", err)
	}

	fmt.Printf("Profile %q added (%s)\n", name, profileType)
	if p.Email != "" {
		fmt.Printf("  Email: %s\n", p.Email)
	}
	return nil
}

// Remove deletes a profile.
func Remove(name string) error {
	dir := config.ProfileDir(name)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("profile %q not found", name)
	}

	active := GetActive()
	if active == name {
		os.WriteFile(config.ActiveFile(), []byte(""), 0600)
	}

	return os.RemoveAll(dir)
}

// Get reads a profile by name.
func Get(name string) (*Profile, error) {
	data, err := os.ReadFile(filepath.Join(config.ProfileDir(name), config.ProfileFileName))
	if err != nil {
		return nil, fmt.Errorf("profile %q not found", name)
	}

	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse profile: %w", err)
	}
	return &p, nil
}

// List returns info about all profiles.
func List() ([]ProfileInfo, error) {
	if err := config.EnsureDir(config.ProfilesDir()); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(config.ProfilesDir())
	if err != nil {
		return nil, err
	}

	active := GetActive()
	var profiles []ProfileInfo

	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), "_") {
			continue
		}

		p, err := Get(e.Name())
		if err != nil {
			continue
		}

		info := ProfileInfo{
			Profile:  *p,
			IsActive: p.Name == active,
		}

		if p.Type == "oauth" {
			dir := config.ProfileDir(p.Name)
			plaintext, err := crypto.DecryptAuthFile(
				filepath.Join(dir, config.AuthKeyFileName),
				filepath.Join(dir, config.AuthFileName),
			)
			if err == nil {
				var tokens token.AuthTokens
				if json.Unmarshal(plaintext, &tokens) == nil {
					info.TokenExpiry = token.GetExpiry(tokens.AccessToken)
					info.IsExpired = token.IsExpired(tokens.AccessToken)
				}
			}
		}

		profiles = append(profiles, info)
	}

	return profiles, nil
}

// SaveActive writes the active profile name.
func SaveActive(name string) error {
	if err := config.EnsureDir(config.ProfilesDir()); err != nil {
		return err
	}
	return os.WriteFile(config.ActiveFile(), []byte(name), 0600)
}

// GetActive returns the current active profile name.
func GetActive() string {
	data, err := os.ReadFile(config.ActiveFile())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// UpdateLastSwitched updates the profile's last switched timestamp.
func UpdateLastSwitched(name string) error {
	p, err := Get(name)
	if err != nil {
		return err
	}
	p.LastSwitched = time.Now().Format(time.RFC3339)
	data, _ := json.MarshalIndent(p, "", "  ")
	return os.WriteFile(filepath.Join(config.ProfileDir(name), config.ProfileFileName), data, 0600)
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0600)
}
