package sinks

import (
	"context"

	"github.com/Ruohao1/penta/internal/core/events"
	"github.com/rs/zerolog"
)

type LogSink struct {
	logger  zerolog.Logger
	verbose int
}

func NewLogSink(logger zerolog.Logger, verbose int) *LogSink {
	return &LogSink{logger: logger, verbose: verbose}
}

func (s *LogSink) Emit(ctx context.Context, ev events.Event) error {
	switch ev.Type {
	case events.EventError:
		s.logger.Error().Str("stage", ev.Stage).Str("err", ev.Err).Msg(ev.Message)
	case events.EventLog:
		s.logger.Info().Str("stage", ev.Stage).Msg(ev.Message)
	default:
		if s.verbose >= 2 {
			s.logger.Debug().Str("stage", ev.Stage).Str("event_type", string(ev.Type)).Msg(ev.String())
		}
	}
	return nil
}

func (s *LogSink) Close() error { return nil }
