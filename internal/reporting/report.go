package reporting

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/viewmodel"
)

func RenderTerminalReport(summary *viewmodel.RunSummary) string {
	var b strings.Builder
	fprintf(&b, "Run        %s\n", summary.RunID)
	fprintf(&b, "Status     %s\n", summary.Status)
	fprintf(&b, "Tasks      %s\n", FormatTaskCounts(summary.TaskCounts))
	fprintf(&b, "Evidence   %s\n", FormatEvidenceCounts(summary.EvidenceCounts))
	if summary.DBPath != "" {
		fprintf(&b, "Database   %s\n", summary.DBPath)
	}
	if len(summary.Evidence) > 0 {
		b.WriteString("\nFindings\n")
		for _, evidence := range summary.Evidence {
			fprintf(&b, "- %s: %s\n", evidence.Kind, evidence.Label)
		}
	}
	return b.String()
}

func RenderMarkdownReport(summary *viewmodel.RunSummary) string {
	var b strings.Builder
	b.WriteString("# Penta Recon Report\n\n")
	b.WriteString("## Summary\n\n")
	fprintf(&b, "- Run: `%s`\n", summary.RunID)
	fprintf(&b, "- Status: `%s`\n", summary.Status)
	fprintf(&b, "- Tasks: %s\n", FormatTaskCounts(summary.TaskCounts))
	fprintf(&b, "- Evidence: %s\n", FormatEvidenceCounts(summary.EvidenceCounts))
	if summary.DBPath != "" {
		fprintf(&b, "- Database: `%s`\n", summary.DBPath)
	}
	b.WriteString("\n## Evidence\n\n")
	if len(summary.Evidence) == 0 {
		b.WriteString("No evidence recorded.\n")
		return b.String()
	}
	for _, evidence := range summary.Evidence {
		fprintf(&b, "- **%s**: %s\n", evidence.Kind, evidence.Label)
	}
	return b.String()
}

func FormatTaskCounts(counts map[actions.TaskStatus]int) string {
	return fmt.Sprintf("%d completed / %d failed / %d pending", counts[actions.TaskStatusCompleted], counts[actions.TaskStatusFailed], counts[actions.TaskStatusPending])
}

func FormatEvidenceCounts(counts map[string]int) string {
	ordered := []string{"target", "dns_record", "service", "http_response"}
	parts := make([]string, 0, len(counts))
	seen := map[string]bool{}
	for _, kind := range ordered {
		if count := counts[kind]; count > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", count, kind))
			seen[kind] = true
		}
	}
	remaining := make([]string, 0, len(counts))
	for kind, count := range counts {
		if seen[kind] || count == 0 {
			continue
		}
		remaining = append(remaining, fmt.Sprintf("%d %s", count, kind))
	}
	sort.Strings(remaining)
	parts = append(parts, remaining...)
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, " / ")
}

type stringWriter interface {
	WriteString(string) (int, error)
}

func fprintf(w stringWriter, format string, args ...any) {
	_, _ = w.WriteString(fmt.Sprintf(format, args...))
}
