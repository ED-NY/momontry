//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Config struct {
	Modifiers uint32 `json:"modifiers"`
	Key       uint32 `json:"key"`
	SaveDir   string `json:"saveDir"`
	Autostart bool   `json:"autostart"`
	Lang      string `json:"lang"`
}

var (
	cfgMu sync.Mutex
	cfg   Config
)

func appDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "momontry")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func picturesDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	for _, name := range []string{"Pictures", "图片"} {
		p := filepath.Join(home, name)
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	return home
}

func defaultConfig() Config {
	return Config{
		Modifiers: modControl | modAlt,
		Key:       0x41,
		SaveDir:   filepath.Join(picturesDir(), "Momontry"),
		Lang:      "zh",
	}
}

func normalizeConfig(c *Config) {
	if c.Key == 0 {
		c.Modifiers = modControl | modAlt
		c.Key = 0x41
	}
	c.Modifiers &^= modNoRepeat
	if c.SaveDir == "" {
		c.SaveDir = filepath.Join(picturesDir(), "Momontry")
	}
	if c.Lang != "en" && c.Lang != "zh" {
		c.Lang = "zh"
	}
}

func configPath() (string, error) {
	dir, err := appDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func loadConfig() {
	cfg = defaultConfig()
	path, err := configPath()
	if err != nil {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		_ = saveConfigFile(cfg)
		return
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		cfg = defaultConfig()
		return
	}
	normalizeConfig(&cfg)
}

func getConfig() Config {
	cfgMu.Lock()
	defer cfgMu.Unlock()
	return cfg
}

func updateConfig(fn func(*Config)) error {
	cfgMu.Lock()
	fn(&cfg)
	normalizeConfig(&cfg)
	snapshot := cfg
	cfgMu.Unlock()
	return saveConfigFile(snapshot)
}

func saveConfigFile(c Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func logf(format string, args ...any) {
	dir, err := appDir()
	if err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, "momontry.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, time.Now().Format("2006-01-02 15:04:05 ")+format+"\n", args...)
}
