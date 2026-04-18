package config

import (
	"os"
	"path/filepath"
)

type Paths struct {
	ConfigDir  string
	StateDir   string
	CacheDir   string
	DataDir    string 

	DBPath     string 
} 

func getEnvOrDefault(env, fallback string) string {
	if v := os.Getenv(env); v != "" && filepath.IsAbs(v) {
		return v
	}
	return fallback
}

func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return h
}

func defaultPaths() *Paths {
	home := homeDir()

	configHome := getEnvOrDefault("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	stateHome  := getEnvOrDefault("XDG_STATE_HOME", filepath.Join(home, ".local", "state"))
	cacheHome  := getEnvOrDefault("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	dataHome   := getEnvOrDefault("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))

	p := &Paths{
		ConfigDir: filepath.Join(configHome, "penta"),
		StateDir:  filepath.Join(stateHome, "penta"),
		CacheDir:  filepath.Join(cacheHome, "penta"),
		DataDir:   filepath.Join(dataHome, "penta"),
	}

	p.DBPath     = filepath.Join(p.StateDir, "penta.db")

	return p
}

func (p *Paths) Ensure() error {
	dirs := []string{
		p.ConfigDir,
		p.StateDir,
		p.CacheDir,
		p.DataDir,
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	return nil
}
