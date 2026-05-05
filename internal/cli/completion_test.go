package cli

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/storage/sqlite"
	"github.com/spf13/cobra"
)

func TestRunCompletionIncludesLatestAndRuns(t *testing.T) {
	app := openTestApp(t)
	ctx := context.Background()
	now := time.Date(2026, 5, 5, 12, 34, 56, 0, time.UTC)
	session := sqlite.Session{ID: "session_1", Name: "local-dev", Kind: sqlite.SessionKindLab, Status: sqlite.SessionStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := app.DB.CreateSession(ctx, session); err != nil {
		t.Fatalf("create session: %v", err)
	}
	run := sqlite.Run{ID: "run_jkqtqegyjq", SessionID: session.ID, Mode: "recon", Status: actions.RunStatusRunning, CreatedAt: now}
	if err := app.DB.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	suggestions, directive := completeRuns(&cobra.Command{}, app, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("completion directive: got %v want %v", directive, cobra.ShellCompDirectiveNoFileComp)
	}
	assertCompletionContains(t, suggestions, "latest\tlatest run")
	assertCompletionContains(t, suggestions, "run_jkqtqegyjq\trunning recon local-dev (lab)")
}

func TestEvidenceShowCompletionDefaultsToIndexesAndFiltersSelectors(t *testing.T) {
	app := openTestApp(t)
	run := createCLIEvidenceRun(t, app, "run_1", 0)
	createCLIEvidenceItem(t, app, run.ID, "evd_target", "target", `{"type":"service","value":"localhost:8080"}`, 0)
	for i := 0; i < 11; i++ {
		createCLIEvidenceItem(t, app, run.ID, fmt.Sprintf("evd_http_%02d", i), "http_response", fmt.Sprintf(`{"url":"http://localhost:8080/%02d","status_code":200,"content_type":"text/html"}`, i), time.Duration(i+1)*time.Second)
	}

	suggestions, directive := completeEvidenceSelectors(&cobra.Command{}, app, "latest", nil, "")
	if directive != completionNoFileKeepOrder {
		t.Fatalf("completion directive: got %v want %v", directive, completionNoFileKeepOrder)
	}
	assertCompletionOrder(t, suggestions, "1\ttarget service localhost:8080", "2\thttp_response http://localhost:8080/00 200", "10\thttp_response http://localhost:8080/08 200", "12\thttp_response http://localhost:8080/10 200")
	assertCompletionMissing(t, suggestions, "http_response:/00\tsemantic selector")
	assertCompletionMissing(t, suggestions, "evd_target\texact evidence id")

	indexSuggestions, _ := completeEvidenceSelectors(&cobra.Command{}, app, "latest", nil, "1")
	assertCompletionOrder(t, indexSuggestions, "1\ttarget service localhost:8080", "10\thttp_response http://localhost:8080/08 200", "11\thttp_response http://localhost:8080/09 200", "12\thttp_response http://localhost:8080/10 200")
	assertCompletionMissing(t, indexSuggestions, "http_response:/00\tsemantic selector")

	semanticSuggestions, _ := completeEvidenceSelectors(&cobra.Command{}, app, "latest", nil, "http_response:/0")
	assertCompletionOrder(t, semanticSuggestions, "http_response:/00\tsemantic selector", "http_response:/01\tsemantic selector")
	assertCompletionMissing(t, semanticSuggestions, "1\ttarget service localhost:8080")

	idSuggestions, _ := completeEvidenceSelectors(&cobra.Command{}, app, "latest", nil, "evd_")
	assertCompletionContains(t, idSuggestions, "evd_target\texact evidence id")
}

func TestArtifactShowCompletionDefaultsToIndexesAndFiltersSelectors(t *testing.T) {
	app := openTestApp(t)
	run := createCLIEvidenceRun(t, app, "run_1", 0)
	createCLIArtifactItem(t, app, run.ID, "art_secret", "/tmp/secret.html", "http://localhost:8080/secret", time.Second)

	suggestions, directive := completeArtifactSelectors(&cobra.Command{}, app, "latest", nil, "")
	if directive != completionNoFileKeepOrder {
		t.Fatalf("completion directive: got %v want %v", directive, completionNoFileKeepOrder)
	}
	assertCompletionContains(t, suggestions, "1\tbody /secret")
	assertCompletionMissing(t, suggestions, "body:/secret\tsemantic selector")
	assertCompletionMissing(t, suggestions, "art_secret\texact artifact id")

	indexSuggestions, _ := completeArtifactSelectors(&cobra.Command{}, app, "latest", nil, "1")
	assertCompletionContains(t, indexSuggestions, "1\tbody /secret")
	assertCompletionMissing(t, indexSuggestions, "body:/secret\tsemantic selector")

	semanticSuggestions, _ := completeArtifactSelectors(&cobra.Command{}, app, "latest", nil, "body:/s")
	assertCompletionContains(t, semanticSuggestions, "body:/secret\tsemantic selector")
	assertCompletionMissing(t, semanticSuggestions, "1\tbody /secret")

	idSuggestions, _ := completeArtifactSelectors(&cobra.Command{}, app, "latest", nil, "art_")
	assertCompletionContains(t, idSuggestions, "art_secret\texact artifact id")
}

func assertCompletionContains(t *testing.T, suggestions []string, want string) {
	t.Helper()
	for _, suggestion := range suggestions {
		if suggestion == want {
			return
		}
	}
	t.Fatalf("completion missing %q in [%s]", want, strings.Join(suggestions, ", "))
}

func assertCompletionMissing(t *testing.T, suggestions []string, unwanted string) {
	t.Helper()
	for _, suggestion := range suggestions {
		if suggestion == unwanted {
			t.Fatalf("completion unexpectedly contained %q in [%s]", unwanted, strings.Join(suggestions, ", "))
		}
	}
}

func assertCompletionOrder(t *testing.T, suggestions []string, ordered ...string) {
	t.Helper()
	next := 0
	for _, suggestion := range suggestions {
		if next < len(ordered) && suggestion == ordered[next] {
			next++
		}
	}
	if next != len(ordered) {
		t.Fatalf("completion order missing sequence %v in [%s]", ordered, strings.Join(suggestions, ", "))
	}
}
