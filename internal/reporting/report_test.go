package reporting

import (
	"strings"
	"testing"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/viewmodel"
)

func TestRenderTerminalReportIncludesSummaryAndEvidence(t *testing.T) {
	summary := sampleSummary()

	got := RenderTerminalReport(summary)
	for _, want := range []string{
		"Run        run_1",
		"Status     completed",
		"Tasks      2 completed / 1 failed / 0 pending",
		"Evidence   1 target / 1 dns_record / 1 service / 1 http_response",
		"Database   /tmp/penta.db",
		"Targets",
		"- domain example.com",
		"DNS Records",
		"- example.com",
		"  A example.com -> 93.184.216.34",
		"Services",
		"- https://example.com",
		"HTTP Responses",
		"- https://example.com 200",
		"  content-type: text/html",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("terminal report missing %q in %q", want, got)
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
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown report missing %q in %q", want, got)
		}
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
			"http_response": 1,
		},
		Evidence: []viewmodel.EvidenceSummary{
			{ID: "evidence_1", Kind: "target", Label: "domain example.com"},
			{ID: "evidence_2", Kind: "dns_record", Label: "example.com", Details: []string{"A example.com -> 93.184.216.34"}},
			{ID: "evidence_3", Kind: "service", Label: "https://example.com", URL: "https://example.com"},
			{ID: "evidence_4", Kind: "http_response", Label: "https://example.com 200", URL: "https://example.com", Details: []string{"content-type: text/html"}},
		},
	}
}
