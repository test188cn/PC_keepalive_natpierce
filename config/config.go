package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

const (
	DefaultServerURL = "ws://127.0.0.1:33272/ws"
	ConfigFileName   = "config.json"
)

// Account represents a saved user account.
type Account struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	SavePassword bool   `json:"save_password"`
	AutoLogin    bool   `json:"auto_login"`
	LastCode     string `json:"last_code"`
	LastConpwd   string `json:"last_conpwd"`
}

// Config holds all application settings.
type Config struct {
	ServerURL      string    `json:"server_url"`
	ExePath        string    `json:"exe_path"`
	Accounts       []Account `json:"accounts"`
	LastAccountIdx int       `json:"last_account_index"`
	KeepAlive      int       `json:"keep_alive"`
	LastCode       string    `json:"last_code"`
	LastConpwd     string    `json:"last_conpwd"`
}

// appDir returns the application directory.
// In production (built exe), it's the exe's directory.
// In dev mode (go run), it falls back to the go.mod directory.
func appDir() string {
	exe, err := os.Executable()
	if err != nil {
		wd, _ := os.Getwd()
		return wd
	}
	dir := filepath.Dir(exe)

	// Detect dev mode: if go.mod exists in this directory, we're in dev mode
	// (go run puts the binary in a temp directory)
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return dir
	}

	// Production mode: check if the exe dir looks like a temp dir
	if runtime.GOOS == "windows" {
		if len(dir) > 4 && dir[:4] == "C:\\T" {
			// Likely temp dir from `go run`, try to find go.mod
			wd, _ := os.Getwd()
			if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
				return wd
			}
		}
	}

	return dir
}

// ConfigPath returns the full path to config.json.
func ConfigPath() string {
	return filepath.Join(appDir(), ConfigFileName)
}

// Default returns a Config with default values.
func Default() *Config {
	return &Config{
		ServerURL: DefaultServerURL,
		KeepAlive: 60,
	}
}

// Load reads config from disk, or returns defaults if not found.
func Load() *Config {
	cfg := Default()
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return cfg
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return cfg
	}
	return cfg
}

// Save writes config to disk atomically.
func (c *Config) Save() error {
	data, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return err
	}
	path := ConfigPath()
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
