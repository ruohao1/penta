package output

import (
	"bytes"
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
