package config

import (
	"os"
	"path/filepath"
)

const (
	ProfilesDirName    = ".cc-profiles"
	FactoryDirName     = ".factory"
	ClaudeDirName      = ".claude"
	ActiveFileName     = "_active"
	AuthFileName       = "auth.v2.file"
	AuthKeyFileName    = "auth.v2.key"
	ProfileFileName    = "profile.json"
	SettingsEnvName    = "settings.env.json"
	ClaudeSettingsName = "settings.json"
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

func FactoryDir() string {
	return filepath.Join(HomeDir(), FactoryDirName)
}

func FactoryAuthFile() string {
	return filepath.Join(FactoryDir(), AuthFileName)
}

func FactoryAuthKey() string {
	return filepath.Join(FactoryDir(), AuthKeyFileName)
}

func ClaudeSettingsPath() string {
	return filepath.Join(HomeDir(), ClaudeDirName, ClaudeSettingsName)
}

func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0700)
}
