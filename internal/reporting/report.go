package reporting

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/storage/sqlite"
	"github.com/ruohao1/penta/internal/viewmodel"
)

func RenderTerminalReport(summary *viewmodel.RunSummary) string {
	var b strings.Builder
	fprintf(&b, "Run        %s\n", summary.RunID)
	fprintf(&b, "Status     %s\n", summary.Status)
	if summary.Session != nil {
		fprintf(&b, "Session    %s (%s, %s)\n", summary.Session.ID, summary.Session.Name, summary.Session.Kind)
		fprintf(&b, "Scope      %s\n", FormatScopeCounts(summary.ScopeRules))
	}
	fprintf(&b, "Tasks      %s\n", FormatTaskCounts(summary.TaskCounts))
	fprintf(&b, "Evidence   %s\n", FormatEvidenceCounts(summary.EvidenceCounts))
	if summary.DBPath != "" {
		fprintf(&b, "Database   %s\n", summary.DBPath)
	}
	renderTerminalEvidenceSections(&b, summary)
	return b.String()
}

func RenderMarkdownReport(summary *viewmodel.RunSummary) string {
	var b strings.Builder
	b.WriteString("# Penta Recon Report\n\n")
	b.WriteString("## Summary\n\n")
	fprintf(&b, "- Run: `%s`\n", summary.RunID)
	fprintf(&b, "- Status: `%s`\n", summary.Status)
	if summary.Session != nil {
		b.WriteString("\n## Session\n\n")
		fprintf(&b, "- ID: `%s`\n", summary.Session.ID)
		fprintf(&b, "- Name: %s\n", summary.Session.Name)
		fprintf(&b, "- Kind: `%s`\n", summary.Session.Kind)
		fprintf(&b, "- Status: `%s`\n", summary.Session.Status)
		if len(summary.ScopeRules) > 0 {
			b.WriteString("\n## Scope\n\n")
			for _, rule := range summary.ScopeRules {
				fprintf(&b, "- %s %s `%s` (%s)\n", rule.Effect, rule.TargetType, rule.Value, rule.ID)
			}
		}
		b.WriteString("\n## Run Summary\n\n")
	}
	fprintf(&b, "- Tasks: %s\n", FormatTaskCounts(summary.TaskCounts))
	fprintf(&b, "- Evidence: %s\n", FormatEvidenceCounts(summary.EvidenceCounts))
	if summary.DBPath != "" {
		fprintf(&b, "- Database: `%s`\n", summary.DBPath)
	}
	if len(summary.Evidence) == 0 {
		b.WriteString("\n## Evidence\n\n")
		b.WriteString("No evidence recorded.\n")
		return b.String()
	}
	renderMarkdownEvidenceSections(&b, summary)
	return b.String()
}

func FormatScopeCounts(rules []sqlite.ScopeRule) string {
	includeCount := 0
	excludeCount := 0
	for _, rule := range rules {
		switch rule.Effect {
		case sqlite.ScopeEffectInclude:
			includeCount++
		case sqlite.ScopeEffectExclude:
			excludeCount++
		}
	}
	return fmt.Sprintf("%d include / %d exclude", includeCount, excludeCount)
}

func renderTerminalEvidenceSections(b *strings.Builder, summary *viewmodel.RunSummary) {
	groups := evidenceByKind(summary)
	for _, section := range evidenceSections() {
		evidenceRows := groups[section.kind]
		if len(evidenceRows) == 0 {
			continue
		}
		fprintf(b, "\n%s\n", section.title)
		for _, evidence := range evidenceRows {
			fprintf(b, "- %s\n", evidence.Label)
			for _, detail := range evidence.Details {
				fprintf(b, "  %s\n", detail)
			}
		}
	}
	if other := otherEvidence(summary); len(other) > 0 {
		b.WriteString("\nOther Evidence\n")
		for _, evidence := range other {
			fprintf(b, "- %s: %s\n", evidence.Kind, evidence.Label)
			for _, detail := range evidence.Details {
				fprintf(b, "  %s\n", detail)
			}
		}
	}
}

func renderMarkdownEvidenceSections(b *strings.Builder, summary *viewmodel.RunSummary) {
	groups := evidenceByKind(summary)
	for _, section := range evidenceSections() {
		evidenceRows := groups[section.kind]
		if len(evidenceRows) == 0 {
			continue
		}
		fprintf(b, "\n## %s\n\n", section.title)
		for _, evidence := range evidenceRows {
			renderMarkdownEvidenceBullet(b, evidence)
		}
	}
	if other := otherEvidence(summary); len(other) > 0 {
		b.WriteString("\n## Other Evidence\n\n")
		for _, evidence := range other {
			renderMarkdownEvidenceBullet(b, evidence)
		}
	}
}

func renderMarkdownEvidenceBullet(b *strings.Builder, evidence viewmodel.EvidenceSummary) {
	label := evidence.Label
	if evidence.URL != "" {
		label = fmt.Sprintf("[%s](%s)", evidence.Label, evidence.URL)
	}
	fprintf(b, "- %s\n", label)
	for _, detail := range evidence.Details {
		fprintf(b, "  - %s\n", detail)
	}
}

type evidenceSection struct {
	kind  string
	title string
}

func evidenceSections() []evidenceSection {
	return []evidenceSection{
		{kind: "target", title: "Targets"},
		{kind: "dns_record", title: "DNS Records"},
		{kind: "service", title: "Services"},
		{kind: "http_response", title: "HTTP Responses"},
	}
}

func evidenceByKind(summary *viewmodel.RunSummary) map[string][]viewmodel.EvidenceSummary {
	groups := map[string][]viewmodel.EvidenceSummary{}
	for _, evidence := range summary.Evidence {
		groups[evidence.Kind] = append(groups[evidence.Kind], evidence)
	}
	return groups
}

func otherEvidence(summary *viewmodel.RunSummary) []viewmodel.EvidenceSummary {
	known := map[string]bool{}
	for _, section := range evidenceSections() {
		known[section.kind] = true
	}
	other := make([]viewmodel.EvidenceSummary, 0)
	for _, evidence := range summary.Evidence {
		if !known[evidence.Kind] {
			other = append(other, evidence)
		}
	}
	return other
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
