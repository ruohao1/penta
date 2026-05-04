package output

import (
	"fmt"
	"io"
	"log/slog"
)

type Sinks struct {
	Out    io.Writer
	Err    io.Writer
	Logger *slog.Logger
}

func New(out, err io.Writer) Sinks {
	out = nonNilWriter(out)
	err = nonNilWriter(err)
	return Sinks{Out: out, Err: err, Logger: NewLogger(err, slog.LevelWarn)}
}

func NewLogger(w io.Writer, level slog.Leveler) *slog.Logger {
	w = nonNilWriter(w)
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: level}))
}

func (s Sinks) Printf(format string, args ...any) {
	_, _ = fmt.Fprintf(nonNilWriter(s.Out), format, args...)
}

func (s Sinks) Warnf(format string, args ...any) {
	_, _ = fmt.Fprintf(nonNilWriter(s.Err), format, args...)
}

func (s Sinks) Errorf(format string, args ...any) {
	_, _ = fmt.Fprintf(nonNilWriter(s.Err), format, args...)
}

func nonNilWriter(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}
