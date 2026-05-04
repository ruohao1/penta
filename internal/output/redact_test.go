package output

import (
	"strings"
	"testing"
)

func TestRedactStringMasksSensitiveKeyValues(t *testing.T) {
	cases := []string{
		"token=abc123",
		"api_key: abc123",
		"access_token=abc123",
		"refresh-token=abc123",
		"x-api-key=abc123",
		"password=secret-value",
		"credential: super-secret",
		"client_secret=topsecret",
	}

	for _, input := range cases {
		got := RedactString(input)
		if strings.Contains(got, "abc123") || strings.Contains(got, "secret-value") || strings.Contains(got, "super-secret") || strings.Contains(got, "topsecret") {
			t.Fatalf("RedactString(%q) leaked secret: %q", input, got)
		}
		if !strings.Contains(got, "[REDACTED]") {
			t.Fatalf("RedactString(%q) did not mark redaction: %q", input, got)
		}
	}
}

func TestRedactStringMasksAuthorizationBearer(t *testing.T) {
	got := RedactString("authorization: Bearer eyJhbGciOiJIUzI1NiJ9")
	if strings.Contains(got, "eyJhbGciOiJIUzI1NiJ9") {
		t.Fatalf("bearer token leaked: %q", got)
	}
	if got != "authorization: Bearer [REDACTED]" {
		t.Fatalf("unexpected redaction: %q", got)
	}
}

func TestRedactStringLeavesNonSensitiveText(t *testing.T) {
	input := "service discovered: https://example.com:443"
	if got := RedactString(input); got != input {
		t.Fatalf("non-sensitive text changed: %q", got)
	}
}
