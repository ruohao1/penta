package viewmodel

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/ruohao1/penta/internal/apperr"
	"github.com/ruohao1/penta/internal/ids"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

type EvidenceList struct {
	Run      sqlite.Run
	Latest   bool
	Evidence []IndexedEvidence
}

type IndexedEvidence struct {
	Index int
	Row   sqlite.Evidence
	EvidenceSummary
}

func BuildEvidenceList(ctx context.Context, db *sqlite.DB, runSelector string) (*EvidenceList, error) {
	run, latest, err := resolveRun(ctx, db, runSelector)
	if err != nil {
		return nil, err
	}
	rows, err := db.ListEvidenceByRun(ctx, run.ID)
	if err != nil {
		return nil, err
	}
	list := &EvidenceList{Run: *run, Latest: latest, Evidence: make([]IndexedEvidence, 0, len(rows))}
	for i, row := range rows {
		summary, err := EvidenceSummaryFor(row)
		if err != nil {
			return nil, err
		}
		list.Evidence = append(list.Evidence, IndexedEvidence{Index: i + 1, Row: row, EvidenceSummary: summary})
	}
	return list, nil
}

func ResolveEvidence(ctx context.Context, db *sqlite.DB, runSelector, selector string) (*EvidenceList, IndexedEvidence, error) {
	if ids.IsEvidenceID(selector) && (runSelector == "" || runSelector == "latest") {
		row, err := db.GetEvidence(ctx, selector)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, IndexedEvidence{}, apperr.NotFound("evidence not found for %q; run `penta evidence list`", selector)
			}
			return nil, IndexedEvidence{}, err
		}
		list, err := BuildEvidenceList(ctx, db, row.RunID)
		if err != nil {
			return nil, IndexedEvidence{}, err
		}
		match, err := resolveEvidenceSelector(list.Evidence, selector)
		if err != nil {
			return nil, IndexedEvidence{}, err
		}
		return list, match, nil
	}
	list, err := BuildEvidenceList(ctx, db, runSelector)
	if err != nil {
		return nil, IndexedEvidence{}, err
	}
	match, err := resolveEvidenceSelector(list.Evidence, selector)
	if err != nil {
		return nil, IndexedEvidence{}, err
	}
	return list, match, nil
}

func resolveRun(ctx context.Context, db *sqlite.DB, selector string) (*sqlite.Run, bool, error) {
	if selector == "" || selector == "latest" {
		run, err := db.LatestRun(ctx)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, true, apperr.NotFound("no runs found")
			}
			return nil, true, err
		}
		return run, true, nil
	}
	run, err := db.GetRun(ctx, selector)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, apperr.NotFound("run not found: %s", selector)
		}
		return nil, false, err
	}
	return run, false, nil
}

func resolveEvidenceSelector(evidence []IndexedEvidence, selector string) (IndexedEvidence, error) {
	if selector == "" {
		return IndexedEvidence{}, apperr.InvalidInput("evidence selector is required")
	}
	if index, err := strconv.Atoi(selector); err == nil {
		if index < 1 || index > len(evidence) {
			return IndexedEvidence{}, apperr.NotFound("evidence index %d not found; run `penta evidence list`", index)
		}
		return evidence[index-1], nil
	}
	if matches := matchesEvidenceID(evidence, selector); len(matches) > 0 || ids.IsEvidenceID(selector) {
		return singleEvidenceMatch(selector, matches)
	}
	parts := strings.SplitN(selector, ":", 2)
	if len(parts) != 2 || parts[0] == "" {
		return IndexedEvidence{}, apperr.InvalidInput("unsupported evidence selector %q; run `penta evidence list`", selector)
	}
	switch parts[0] {
	case "http_response":
		return singleEvidenceMatch(selector, matchesKindValue(evidence, "http_response", parts[1], true))
	case "service":
		return singleEvidenceMatch(selector, matchesKindValue(evidence, "service", parts[1], false))
	case "crawl":
		return singleEvidenceMatch(selector, matchesKindValue(evidence, "crawl", parts[1], true))
	default:
		return IndexedEvidence{}, apperr.InvalidInput("unsupported evidence selector kind %q; run `penta evidence list`", parts[0])
	}
}

func matchesEvidenceID(evidence []IndexedEvidence, id string) []IndexedEvidence {
	var matches []IndexedEvidence
	for _, item := range evidence {
		if item.ID == id {
			matches = append(matches, item)
		}
	}
	return matches
}

func matchesKindValue(evidence []IndexedEvidence, kind, value string, pathMatch bool) []IndexedEvidence {
	var matches []IndexedEvidence
	for _, item := range evidence {
		if item.Kind != kind {
			continue
		}
		if item.Label == value || item.URL == value || (pathMatch && displayEvidenceURLPath(item.URL) == value) {
			matches = append(matches, item)
		}
	}
	return matches
}

func singleEvidenceMatch(selector string, matches []IndexedEvidence) (IndexedEvidence, error) {
	if len(matches) == 0 {
		return IndexedEvidence{}, apperr.NotFound("evidence not found for %q; run `penta evidence list`", selector)
	}
	if len(matches) > 1 {
		return IndexedEvidence{}, apperr.Conflict("ambiguous evidence selector %q matched %d items; run `penta evidence list` and choose an index", selector, len(matches))
	}
	return matches[0], nil
}

func FormatRunContext(runID string, latest bool) string {
	if latest {
		return fmt.Sprintf("%s (latest)", runID)
	}
	return runID
}

func displayEvidenceURLPath(value string) string {
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
