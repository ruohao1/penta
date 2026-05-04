package output

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
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
	handler := slog.NewTextHandler(w, &slog.HandlerOptions{Level: level})
	return slog.New(redactingHandler{next: handler})
}

type redactingHandler struct {
	next slog.Handler
}

func (h redactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h redactingHandler) Handle(ctx context.Context, record slog.Record) error {
	redacted := slog.NewRecord(record.Time, record.Level, RedactString(record.Message), record.PC)
	record.Attrs(func(attr slog.Attr) bool {
		redacted.AddAttrs(redactAttr(attr))
		return true
	})
	return h.next.Handle(ctx, redacted)
}

func (h redactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	redacted := make([]slog.Attr, 0, len(attrs))
	for _, attr := range attrs {
		redacted = append(redacted, redactAttr(attr))
	}
	return redactingHandler{next: h.next.WithAttrs(redacted)}
}

func (h redactingHandler) WithGroup(name string) slog.Handler {
	return redactingHandler{next: h.next.WithGroup(name)}
}

func redactAttr(attr slog.Attr) slog.Attr {
	attr.Value = attr.Value.Resolve()
	if attr.Value.Kind() == slog.KindGroup {
		attrs := attr.Value.Group()
		redacted := make([]slog.Attr, 0, len(attrs))
		for _, groupAttr := range attrs {
			redacted = append(redacted, redactAttr(groupAttr))
		}
		attr.Value = slog.GroupValue(redacted...)
	} else if attr.Value.Kind() == slog.KindString {
		attr.Value = slog.StringValue(RedactString(attr.Value.String()))
	} else if attr.Value.Kind() == slog.KindAny {
		attr.Value = slog.StringValue(RedactString(attr.Value.String()))
	}
	if isSensitiveKey(attr.Key) {
		attr.Value = slog.StringValue("[REDACTED]")
	}
	return attr
}

func (s Sinks) Printf(format string, args ...any) {
	_, _ = fmt.Fprintf(nonNilWriter(s.Out), format, args...)
}

func (s Sinks) Warnf(format string, args ...any) {
	_, _ = fmt.Fprint(nonNilWriter(s.Err), RedactString(fmt.Sprintf(format, args...)))
}

func (s Sinks) Errorf(format string, args ...any) {
	_, _ = fmt.Fprint(nonNilWriter(s.Err), RedactString(fmt.Sprintf(format, args...)))
}

func nonNilWriter(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}

func isSensitiveKey(key string) bool {
	switch strings.ToLower(key) {
	case "token", "access_token", "access-token", "refresh_token", "refresh-token", "api_key", "api-key", "x-api-key", "x_api_key", "authorization", "auth", "password", "passwd", "secret", "client_secret", "client-secret", "credential":
		return true
	default:
		return false
	}
}
