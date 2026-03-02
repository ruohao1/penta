package config

import (
	"context"

	"github.com/Ruohao1/penta/internal/core/sinks"
	"github.com/rs/zerolog"
)

type Config struct{}

func LoadConfig() *Config {
	return &Config{}
}

type (
	ctxLoggerKey struct{}
	ctxConfigKey struct{}
	ctxSinkKey   struct{}
)

func WithLogger(ctx context.Context, l zerolog.Logger) context.Context {
	return context.WithValue(ctx, ctxLoggerKey{}, l)
}

func LoggerFrom(ctx context.Context) zerolog.Logger {
	l, ok := ctx.Value(ctxLoggerKey{}).(zerolog.Logger)
	if !ok {
		return zerolog.Nop()
	}
	return l
}

func WithConfig(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, ctxConfigKey{}, cfg)
}

func ConfigFrom(ctx context.Context) *Config {
	cfg, _ := ctx.Value(ctxConfigKey{}).(*Config)
	return cfg
}

func WithSink(ctx context.Context, s sinks.Sink) context.Context {
	return context.WithValue(ctx, ctxSinkKey{}, s)
}
func SinkFrom(ctx context.Context) sinks.Sink {
	s, _ := ctx.Value(ctxSinkKey{}).(sinks.Sink)
	return s
}
