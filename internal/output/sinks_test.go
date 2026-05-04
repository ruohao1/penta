package output

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestSinksRouteOutputAndErrors(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	sinks := New(&out, &errOut)

	sinks.Printf("hello %s", "out")
	sinks.Warnf("warn %d", 1)
	sinks.Errorf("error %d", 2)

	if out.String() != "hello out" {
		t.Fatalf("unexpected stdout: %q", out.String())
	}
	if errOut.String() != "warn 1error 2" {
		t.Fatalf("unexpected stderr: %q", errOut.String())
	}
}

func TestSinksRedactWarningsAndErrors(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	sinks := New(&out, &errOut)

	sinks.Printf("token=stdout-secret")
	sinks.Warnf("token=%s", "warn-secret")
	sinks.Errorf("authorization: Bearer %s", "error-secret")

	if out.String() != "token=stdout-secret" {
		t.Fatalf("stdout should not be redacted: %q", out.String())
	}
	gotErr := errOut.String()
	if strings.Contains(gotErr, "warn-secret") || strings.Contains(gotErr, "error-secret") {
		t.Fatalf("stderr leaked secret: %q", gotErr)
	}
	if !strings.Contains(gotErr, "token=[REDACTED]") || !strings.Contains(gotErr, "authorization: Bearer [REDACTED]") {
		t.Fatalf("stderr missing redaction markers: %q", gotErr)
	}
}

func TestSinksAreNilSafe(t *testing.T) {
	sinks := New(nil, nil)
	sinks.Printf("ignored")
	sinks.Warnf("ignored")
	sinks.Errorf("ignored")
	if sinks.Logger == nil {
		t.Fatal("expected nil-safe logger")
	}
}

func TestLoggerWritesToErrorStream(t *testing.T) {
	var errOut bytes.Buffer
	sinks := New(nil, &errOut)

	sinks.Logger.Warn("diagnostic", "component", "test")

	got := errOut.String()
	if !strings.Contains(got, "level=WARN") || !strings.Contains(got, "diagnostic") || !strings.Contains(got, "component=test") {
		t.Fatalf("unexpected log output: %q", got)
	}
}

func TestLoggerRedactsMessagesAndStringAttrs(t *testing.T) {
	var errOut bytes.Buffer
	sinks := New(nil, &errOut)

	sinks.Logger.Warn("request failed token=message-secret", "api_key", "attr-secret", "authorization", "Bearer bearer-secret", "err", fmt.Errorf("authorization: Bearer error-secret"), "count", 2)

	got := errOut.String()
	if strings.Contains(got, "message-secret") || strings.Contains(got, "attr-secret") || strings.Contains(got, "bearer-secret") || strings.Contains(got, "error-secret") {
		t.Fatalf("logger leaked secret: %q", got)
	}
	if !strings.Contains(got, "token=[REDACTED]") || !strings.Contains(got, "api_key=[REDACTED]") || !strings.Contains(got, "authorization=[REDACTED]") || !strings.Contains(got, "err=\"authorization: Bearer [REDACTED]\"") || !strings.Contains(got, "count=2") {
		t.Fatalf("logger missing expected fields: %q", got)
	}
}
