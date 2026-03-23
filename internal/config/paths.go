package config

import (
	"os"
	"path/filepath"
)

const (
	ProfilesDirName     = ".cc-profiles"
	ClaudeDirName       = ".claude"
	ActiveFileName      = "_active"
	CredentialsFileName = ".credentials.json"
	OAuthFileName       = "oauth.json"
	ProfileFileName     = "profile.json"
	SettingsEnvName     = "settings.env.json"
	ClaudeSettingsName  = "settings.json"
)

func HomeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		panic("cannot determine home directory: " + err.Error())
	}
	return h
}

func ProfilesDir() string {
	return filepath.Join(HomeDir(), ProfilesDirName)
}

func ProfileDir(name string) string {
	return filepath.Join(ProfilesDir(), name)
}

func ActiveFile() string {
	return filepath.Join(ProfilesDir(), ActiveFileName)
}

func ClaudeDir() string {
	return filepath.Join(HomeDir(), ClaudeDirName)
}

func CredentialsFile() string {
	return filepath.Join(ClaudeDir(), CredentialsFileName)
}

func ClaudeSettingsPath() string {
	return filepath.Join(HomeDir(), ClaudeDirName, ClaudeSettingsName)
}

func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0700)
}
