package crawl

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/events"
	"github.com/ruohao1/penta/internal/ids"
	"github.com/ruohao1/penta/internal/storage/sqlite"
	"github.com/ruohao1/penta/internal/viewmodel"
)

const maxDiscoveredURLs = 50

var hrefPattern = regexp.MustCompile(`(?i)<a\s+[^>]*href\s*=\s*["']([^"']*)["']`)

func Execute(ctx context.Context, db *sqlite.DB, sink events.Sink, task *sqlite.Task) error {
	var input Input
	if err := json.Unmarshal([]byte(task.InputJSON), &input); err != nil {
		return err
	}
	if input.BodyArtifactID == "" || !strings.Contains(strings.ToLower(input.ContentType), "text/html") {
		return nil
	}
	artifact, err := db.GetArtifact(ctx, input.BodyArtifactID)
	if err != nil {
		return err
	}
	body, err := os.ReadFile(artifact.Path)
	if err != nil {
		return err
	}
	urls := extractLinks(input.URL, string(body), maxDiscoveredURLs)
	if len(urls) == 0 {
		return nil
	}
	evidenceJSON, err := json.Marshal(Evidence{SourceURL: input.URL, Depth: input.Depth, URLs: urls})
	if err != nil {
		return err
	}
	evidence := sqlite.Evidence{ID: ids.New(ids.PrefixEvidence), RunID: task.RunID, TaskID: task.ID, Kind: string(actions.EvidenceCrawl), DataJSON: string(evidenceJSON), CreatedAt: time.Now()}
	label, err := viewmodel.EvidenceLabel(evidence)
	if err != nil {
		return err
	}
	if err := db.CreateEvidence(ctx, evidence); err != nil {
		return err
	}
	if sink == nil {
		return nil
	}
	return sink.Append(ctx, events.Event{RunID: task.RunID, EventType: events.EventEvidenceCreated, EntityKind: events.EntityEvidence, EntityID: evidence.ID, PayloadJSON: mustPayloadJSON(events.EvidenceCreatedPayload{Kind: evidence.Kind, Label: label}), CreatedAt: time.Now()})
}

func extractLinks(baseURL, body string, limit int) []string {
	base, err := url.Parse(baseURL)
	if err != nil || base.Scheme == "" || base.Host == "" || limit <= 0 {
		return nil
	}
	seen := map[string]bool{}
	links := make([]string, 0)
	for _, match := range hrefPattern.FindAllStringSubmatch(body, -1) {
		if len(match) < 2 {
			continue
		}
		link, ok := normalizeLink(base, match[1])
		if !ok || link == canonicalURL(base) || seen[link] {
			continue
		}
		seen[link] = true
		links = append(links, link)
		if len(links) >= limit {
			break
		}
	}
	sort.Strings(links)
	return links
}

func canonicalURL(value *url.URL) string {
	copyValue := *value
	copyValue.Fragment = ""
	return copyValue.String()
}

func normalizeLink(base *url.URL, raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "mailto:") || strings.HasPrefix(lower, "javascript:") || strings.HasPrefix(lower, "tel:") {
		return "", false
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	resolved := base.ResolveReference(parsed)
	resolved.Fragment = ""
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return "", false
	}
	if !sameOrigin(base, resolved) {
		return "", false
	}
	return resolved.String(), true
}

func sameOrigin(a, b *url.URL) bool {
	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Hostname(), b.Hostname()) && effectivePort(a) == effectivePort(b)
}

func effectivePort(value *url.URL) string {
	if port := value.Port(); port != "" {
		return port
	}
	if value.Scheme == "https" {
		return "443"
	}
	return "80"
}

func mustPayloadJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}
