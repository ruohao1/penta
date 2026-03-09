package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ruohao1/penta/internal/runtime"
	"github.com/spf13/viper"
)

type Config struct {
	Run RunConfig `mapstructure:"run"`
}

type RunConfig struct {
	FailFast     *bool          `mapstructure:"fail_fast"`
	BufferSize   *int           `mapstructure:"buffer_size"`
	Workers      *int           `mapstructure:"workers"`
	MaxRate      *float64       `mapstructure:"max_rate"`
	RateBurst    *int           `mapstructure:"rate_burst"`
	MaxRetries   *int           `mapstructure:"max_retries"`
	RetryBackoff *time.Duration `mapstructure:"retry_backoff"`
	Timeout      *time.Duration `mapstructure:"timeout"`
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "penta", "config.yaml"), nil
}

func Load(path string) (Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetEnvPrefix("PENTA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if path == "" {
		defaultPath, err := DefaultPath()
		if err != nil {
			return Config{}, err
		}
		path = defaultPath
	}

	v.SetConfigFile(path)
	err := v.ReadInConfig()
	if err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && !errors.Is(err, os.ErrNotExist) {
			return Config{}, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) ToRuntimeConfig() runtime.Config {
	out := runtime.DefaultConfig()

	if c.Run.FailFast != nil {
		out.FailFast = *c.Run.FailFast
	}
	if c.Run.BufferSize != nil {
		out.BufferSize = *c.Run.BufferSize
	}
	if c.Run.Workers != nil {
		out.Workers = *c.Run.Workers
	}
	if c.Run.MaxRate != nil {
		out.MaxRate = *c.Run.MaxRate
	}
	if c.Run.RateBurst != nil {
		out.RateBurst = *c.Run.RateBurst
	}
	if c.Run.MaxRetries != nil {
		out.MaxRetries = *c.Run.MaxRetries
	}
	if c.Run.RetryBackoff != nil {
		out.RetryBackoff = *c.Run.RetryBackoff
	}
	if c.Run.Timeout != nil {
		out.Timeout = *c.Run.Timeout
	}

	return out
}

type runtimeConfigKey struct{}

func WithRuntimeConfig(ctx context.Context, cfg runtime.Config) context.Context {
	return context.WithValue(ctx, runtimeConfigKey{}, cfg)
}

func RuntimeConfigFrom(ctx context.Context) (runtime.Config, bool) {
	cfg, ok := ctx.Value(runtimeConfigKey{}).(runtime.Config)
	return cfg, ok
}
