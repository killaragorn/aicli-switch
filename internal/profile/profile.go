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
	"github.com/killaragorn/aicli-switch/internal/token"
)

type Profile struct {
	Name         string `json:"name"`
	Type         string `json:"type"` // "oauth" or "apikey"
	Email        string `json:"email,omitempty"`
	CreatedAt    string `json:"created_at"`
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

// ReadCredentialsOAuth reads the claudeAiOauth section from ~/.claude/.credentials.json
func ReadCredentialsOAuth() (*token.OAuthData, error) {
	data, err := os.ReadFile(config.CredentialsFile())
	if err != nil {
		return nil, fmt.Errorf("read credentials file: %w", err)
	}

	var creds token.CredentialsFile
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("parse credentials: %w", err)
	}

	if creds.ClaudeAiOauth == nil {
		return nil, fmt.Errorf("no claudeAiOauth found in credentials file")
	}

	var oauth token.OAuthData
	if err := json.Unmarshal(creds.ClaudeAiOauth, &oauth); err != nil {
		return nil, fmt.Errorf("parse claudeAiOauth: %w", err)
	}

	return &oauth, nil
}

// WriteCredentialsOAuth writes the claudeAiOauth section to ~/.claude/.credentials.json
// while preserving other fields (like mcpOAuth).
func WriteCredentialsOAuth(oauth *token.OAuthData) error {
	credPath := config.CredentialsFile()

	// Read existing file to preserve mcpOAuth etc.
	var raw map[string]json.RawMessage
	data, err := os.ReadFile(credPath)
	if err != nil {
		raw = make(map[string]json.RawMessage)
	} else {
		if err := json.Unmarshal(data, &raw); err != nil {
			raw = make(map[string]json.RawMessage)
		}
	}

	oauthBytes, err := json.Marshal(oauth)
	if err != nil {
		return fmt.Errorf("marshal oauth: %w", err)
	}
	raw["claudeAiOauth"] = oauthBytes

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}

	return os.WriteFile(credPath, out, 0600)
}

// ReadProfileOAuth reads the saved oauth.json from a profile directory.
func ReadProfileOAuth(name string) (*token.OAuthData, error) {
	oauthPath := filepath.Join(config.ProfileDir(name), config.OAuthFileName)
	data, err := os.ReadFile(oauthPath)
	if err != nil {
		return nil, fmt.Errorf("read oauth file for profile %q: %w", name, err)
	}

	var oauth token.OAuthData
	if err := json.Unmarshal(data, &oauth); err != nil {
		return nil, fmt.Errorf("parse oauth for profile %q: %w", name, err)
	}
	return &oauth, nil
}

// SaveProfileOAuth saves oauth data to a profile's oauth.json.
func SaveProfileOAuth(name string, oauth *token.OAuthData) error {
	oauthPath := filepath.Join(config.ProfileDir(name), config.OAuthFileName)
	data, err := json.MarshalIndent(oauth, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal oauth: %w", err)
	}
	return os.WriteFile(oauthPath, data, 0600)
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
		oauth, err := ReadCredentialsOAuth()
		if err != nil {
			os.RemoveAll(dir)
			return fmt.Errorf("read current OAuth credentials: %w\nPlease run 'claude login' first", err)
		}

		if err := SaveProfileOAuth(name, oauth); err != nil {
			os.RemoveAll(dir)
			return fmt.Errorf("save oauth data: %w", err)
		}

		p.Email = token.GetEmail(oauth.AccessToken)

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
		if !e.IsDir() || strings.HasPrefix(e.Name(), "_") || strings.HasPrefix(e.Name(), ".") {
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
			oauth, err := ReadProfileOAuth(p.Name)
			if err == nil {
				info.TokenExpiry = token.GetExpiryFromData(oauth)
				info.IsExpired = token.IsExpiredData(oauth)
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
