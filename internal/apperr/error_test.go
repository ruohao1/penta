package apperr

import (
	"errors"
	"testing"
)

func TestErrorKindAndMessage(t *testing.T) {
	err := Conflict("report file already exists: %s", "report.md")
	var appErr *Error
	if !errors.As(err, &appErr) {
		t.Fatal("expected app error")
	}
	if appErr.Kind != KindConflict {
		t.Fatalf("unexpected kind: %s", appErr.Kind)
	}
	if err.Error() != "report file already exists: report.md" {
		t.Fatalf("unexpected message: %q", err.Error())
	}
}

func TestErrorWrapsCause(t *testing.T) {
	cause := errors.New("storage failed")
	err := Wrap(KindNotFound, "session not found", cause)
	if !errors.Is(err, cause) {
		t.Fatalf("expected wrapped cause")
	}
}

func TestReportedErrorMarksAlreadyPrintedErrors(t *testing.T) {
	cause := errors.New("already printed")
	err := Reported(cause)
	if !IsReported(err) {
		t.Fatal("expected reported error")
	}
	if !errors.Is(err, cause) {
		t.Fatal("expected reported error to wrap cause")
	}
	if IsReported(cause) {
		t.Fatal("plain cause should not be reported")
	}
}
