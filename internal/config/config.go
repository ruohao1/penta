package config

import (
	"errors"
	"log"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Storage StorageConfig `mapstructure:"storage"`
}

type StorageConfig struct {
	DBPath string `mapstructure:"db_path"`
}

func Load() (*Config, error) {
	v := viper.New()

	// --- Defaults (baseline)
	paths := defaultPaths()
	err := paths.Ensure() // ensure directories exist
	if err != nil {
		return nil, err
	}
	v.SetDefault("storage.db_path", paths.DBPath)
	v.SetEnvPrefix("PENTA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	_ = v.BindEnv("storage.db_path")

	// --- Config file (optional)
	v.SetConfigName("penta")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath(".penta")
	v.AddConfigPath(paths.ConfigDir)

	// --- Read config file
	if err := v.ReadInConfig(); err == nil {
		log.Println("using config file:", v.ConfigFileUsed())
	} else {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, err
		}
	}

	// --- Unmarshal
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
