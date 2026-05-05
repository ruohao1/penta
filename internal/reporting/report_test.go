package reporting

import (
	"strings"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/storage/sqlite"
	"github.com/ruohao1/penta/internal/viewmodel"
)

func TestRenderTerminalReportIncludesSummaryAndEvidence(t *testing.T) {
	summary := sampleSummary()

	got := RenderTerminalReport(summary)
	for _, want := range []string{
		"Run        run_1",
		"Status     completed",
		"Tasks      2 completed / 1 failed / 0 pending",
		"Evidence   1 target / 1 dns_record / 1 service / 2 http_response / 2 crawl",
		"Database   /tmp/penta.db",
		"Targets",
		"- domain example.com",
		"DNS Records",
		"- example.com",
		"  A example.com -> 93.184.216.34",
		"Services",
		"- https://example.com",
		"HTTP Responses",
		"- 200 text/html 512bytes /",
		"- 302 /redirect",
		"Crawl",
		"3 unique URLs discovered from 2 pages",
		"- /docs",
		"- /login",
		"- /secret",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("terminal report missing %q in %q", want, got)
		}
	}
	for _, unwanted := range []string{"sha256:", "body artifact:"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("terminal report should hide %q in %q", unwanted, got)
		}
	}
}

func TestRenderTerminalReportCanColorHTTPStatusCodes(t *testing.T) {
	summary := sampleSummary()

	got := RenderTerminalReportWithOptions(summary, RenderOptions{Color: true})
	for _, want := range []string{"\x1b[32m200\x1b[0m", "\x1b[33m302\x1b[0m"} {
		if !strings.Contains(got, want) {
			t.Fatalf("colored terminal report missing %q in %q", want, got)
		}
	}
	for _, want := range []string{"- ", " text/html 512bytes /", " /redirect"} {
		if !strings.Contains(got, want) {
			t.Fatalf("colored terminal report missing compact row content %q in %q", want, got)
		}
	}
}

func TestRenderMarkdownReportIncludesSummaryAndEvidence(t *testing.T) {
	summary := sampleSummary()

	got := RenderMarkdownReport(summary)
	for _, want := range []string{
		"# Penta Recon Report",
		"## Summary",
		"- Run: `run_1`",
		"- Status: `completed`",
		"- Tasks: 2 completed / 1 failed / 0 pending",
		"## Targets",
		"- domain example.com",
		"## DNS Records",
		"- example.com",
		"  - A example.com -> 93.184.216.34",
		"## Services",
		"- [https://example.com](https://example.com)",
		"## HTTP Responses",
		"- [https://example.com 200](https://example.com)",
		"  - content-type: text/html",
		"  - sha256: abc123",
		"  - body artifact: artifact_1",
		"## Crawl",
		"- [2 urls from https://example.com/](https://example.com/)",
		"  - https://example.com/login",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown report missing %q in %q", want, got)
		}
	}
}

func TestRenderReportsIncludeSessionMetadata(t *testing.T) {
	summary := sampleSummary()
	summary.Session = &sqlite.Session{ID: "session_1", Name: "Acme", Kind: sqlite.SessionKindBugBounty, Status: sqlite.SessionStatusActive, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	summary.ScopeRules = []sqlite.ScopeRule{
		{ID: "scope_1", Effect: sqlite.ScopeEffectInclude, TargetType: sqlite.ScopeTargetDomain, Value: "*.example.com"},
		{ID: "scope_2", Effect: sqlite.ScopeEffectExclude, TargetType: sqlite.ScopeTargetDomain, Value: "admin.example.com"},
	}

	terminal := RenderTerminalReport(summary)
	for _, want := range []string{"Session    session_1 (Acme, bugbounty)", "Scope      1 include / 1 exclude"} {
		if !strings.Contains(terminal, want) {
			t.Fatalf("terminal report missing %q in %q", want, terminal)
		}
	}

	markdown := RenderMarkdownReport(summary)
	for _, want := range []string{"## Session", "- ID: `session_1`", "- Name: Acme", "## Scope", "- include domain `*.example.com` (scope_1)", "## Run Summary"} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("markdown report missing %q in %q", want, markdown)
		}
	}
}

func TestRenderReportsCanRedactEvidenceStrings(t *testing.T) {
	summary := sampleSummary()
	summary.Evidence = []viewmodel.EvidenceSummary{{ID: "evidence_secret", Kind: "http_response", Label: "https://example.com?token=label-secret 200", URL: "https://example.com?token=url-secret", Details: []string{"authorization: Bearer detail-secret"}}}

	terminal := RenderTerminalReportWithOptions(summary, RenderOptions{Redact: true})
	markdown := RenderMarkdownReportWithOptions(summary, RenderOptions{Redact: true})
	for _, got := range []string{terminal, markdown} {
		if strings.Contains(got, "label-secret") || strings.Contains(got, "url-secret") || strings.Contains(got, "detail-secret") {
			t.Fatalf("redacted report leaked secret: %q", got)
		}
		if !strings.Contains(got, "[REDACTED]") {
			t.Fatalf("redacted report missing marker: %q", got)
		}
	}
	if strings.Contains(summary.Evidence[0].Label, "[REDACTED]") || strings.Contains(summary.Evidence[0].Details[0], "[REDACTED]") {
		t.Fatalf("redaction mutated summary: %+v", summary.Evidence[0])
	}
}

func sampleSummary() *viewmodel.RunSummary {
	return &viewmodel.RunSummary{
		RunID:  "run_1",
		Status: actions.RunStatusCompleted,
		DBPath: "/tmp/penta.db",
		TaskCounts: map[actions.TaskStatus]int{
			actions.TaskStatusCompleted: 2,
			actions.TaskStatusFailed:    1,
		},
		EvidenceCounts: map[string]int{
			"target":        1,
			"dns_record":    1,
			"service":       1,
			"http_response": 2,
			"crawl":         2,
		},
		Evidence: []viewmodel.EvidenceSummary{
			{ID: "evidence_1", Kind: "target", Label: "domain example.com"},
			{ID: "evidence_2", Kind: "dns_record", Label: "example.com", Details: []string{"A example.com -> 93.184.216.34"}},
			{ID: "evidence_3", Kind: "service", Label: "https://example.com", URL: "https://example.com"},
			{ID: "evidence_4", Kind: "http_response", Label: "https://example.com 200", URL: "https://example.com", Details: []string{"content-type: text/html; charset=utf-8", "body: 512 bytes", "sha256: abc123", "body artifact: artifact_1"}},
			{ID: "evidence_5", Kind: "http_response", Label: "https://example.com/redirect 302", URL: "https://example.com/redirect"},
			{ID: "evidence_6", Kind: "crawl", Label: "2 urls from https://example.com/", URL: "https://example.com/", Details: []string{"https://example.com/login", "https://example.com/docs"}},
			{ID: "evidence_7", Kind: "crawl", Label: "2 urls from https://example.com/docs", URL: "https://example.com/docs", Details: []string{"https://example.com/login", "https://example.com/secret"}},
		},
	}
}
