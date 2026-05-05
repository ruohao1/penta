package reporting

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/output"
	"github.com/ruohao1/penta/internal/storage/sqlite"
	"github.com/ruohao1/penta/internal/viewmodel"
)

type RenderOptions struct {
	Redact bool
}

func RenderTerminalReport(summary *viewmodel.RunSummary) string {
	return RenderTerminalReportWithOptions(summary, RenderOptions{})
}

func RenderTerminalReportWithOptions(summary *viewmodel.RunSummary, options RenderOptions) string {
	summary = renderSummary(summary, options)
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
	return RenderMarkdownReportWithOptions(summary, RenderOptions{})
}

func RenderMarkdownReportWithOptions(summary *viewmodel.RunSummary, options RenderOptions) string {
	summary = renderSummary(summary, options)
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

func renderSummary(summary *viewmodel.RunSummary, options RenderOptions) *viewmodel.RunSummary {
	if !options.Redact || summary == nil {
		return summary
	}
	redacted := *summary
	redacted.Evidence = make([]viewmodel.EvidenceSummary, len(summary.Evidence))
	for i, evidence := range summary.Evidence {
		redacted.Evidence[i] = redactEvidenceSummary(evidence)
	}
	return &redacted
}

func redactEvidenceSummary(evidence viewmodel.EvidenceSummary) viewmodel.EvidenceSummary {
	evidence.Label = output.RedactString(evidence.Label)
	evidence.URL = output.RedactString(evidence.URL)
	evidence.Details = append([]string(nil), evidence.Details...)
	for i, detail := range evidence.Details {
		evidence.Details[i] = output.RedactString(detail)
	}
	return evidence
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
		if section.kind == "http_response" {
			renderTerminalHTTPResponses(b, evidenceRows)
			continue
		}
		if section.kind == "crawl" {
			renderTerminalCrawl(b, evidenceRows)
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

func renderTerminalHTTPResponses(b *strings.Builder, evidenceRows []viewmodel.EvidenceSummary) {
	b.WriteString("\nHTTP Responses\n")
	for _, evidence := range evidenceRows {
		fprintf(b, "- %s\n", compactHTTPResponseLine(evidence))
	}
}

func compactHTTPResponseLine(evidence viewmodel.EvidenceSummary) string {
	status := ""
	fields := strings.Fields(evidence.Label)
	if len(fields) > 0 {
		last := fields[len(fields)-1]
		if _, err := strconv.Atoi(last); err == nil {
			status = last
		}
	}
	parts := make([]string, 0, 4)
	if status != "" {
		parts = append(parts, status)
	}
	if contentType := compactDetailValue(evidence.Details, "content-type: "); contentType != "" {
		parts = append(parts, compactContentType(contentType))
	}
	if body := compactBodySize(evidence.Details); body != "" {
		parts = append(parts, body)
	}
	parts = append(parts, displayURLPath(evidence.URL))
	if hasDetailPrefix(evidence.Details, "headers: truncated") {
		parts = append(parts, "(headers truncated)")
	}
	if strings.Contains(strings.Join(evidence.Details, "\n"), "(truncated") {
		parts = append(parts, "(body truncated)")
	}
	return strings.Join(parts, " ")
}

func renderTerminalCrawl(b *strings.Builder, evidenceRows []viewmodel.EvidenceSummary) {
	unique := map[string]bool{}
	for _, evidence := range evidenceRows {
		for _, detail := range evidence.Details {
			if detail != "" {
				unique[detail] = true
			}
		}
	}
	urls := sortedKeys(unique)
	fprintf(b, "\nCrawl\n")
	fprintf(b, "%d unique URLs discovered from %d pages\n", len(urls), len(evidenceRows))
	for _, value := range urls {
		fprintf(b, "- %s\n", displayURLPath(value))
	}
}

func compactDetailValue(details []string, prefix string) string {
	for _, detail := range details {
		if strings.HasPrefix(detail, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(detail, prefix))
		}
	}
	return ""
}

func compactContentType(value string) string {
	value = strings.TrimSpace(strings.Split(value, ";")[0])
	if value == "" {
		return "unknown"
	}
	return value
}

func compactBodySize(details []string) string {
	body := compactDetailValue(details, "body: ")
	if body == "" {
		return ""
	}
	fields := strings.Fields(body)
	if len(fields) >= 2 {
		return fields[0] + fields[1]
	}
	return body
}

func hasDetailPrefix(details []string, prefix string) bool {
	for _, detail := range details {
		if strings.HasPrefix(detail, prefix) {
			return true
		}
	}
	return false
}

func displayURLPath(value string) string {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return value
	}
	path := parsed.EscapedPath()
	if path == "" {
		path = "/"
	}
	if parsed.RawQuery != "" {
		path += "?" + parsed.RawQuery
	}
	return path
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
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
		{kind: "crawl", title: "Crawl"},
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
	ordered := []string{"target", "dns_record", "service", "http_response", "crawl"}
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
