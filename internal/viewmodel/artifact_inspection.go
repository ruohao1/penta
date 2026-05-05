package viewmodel

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/ruohao1/penta/internal/apperr"
	"github.com/ruohao1/penta/internal/model"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

type ArtifactList struct {
	Run       sqlite.Run
	Latest    bool
	Artifacts []IndexedArtifact
}

type IndexedArtifact struct {
	Index  int
	Row    sqlite.Artifact
	Task   sqlite.Task
	Kind   string
	Source string
}

func BuildArtifactList(ctx context.Context, db *sqlite.DB, runSelector string) (*ArtifactList, error) {
	run, latest, err := resolveRun(ctx, db, runSelector)
	if err != nil {
		return nil, err
	}
	artifacts, err := db.ListArtifactsByRun(ctx, run.ID)
	if err != nil {
		return nil, err
	}
	summaries, err := artifactSummaries(ctx, db, run.ID)
	if err != nil {
		return nil, err
	}
	list := &ArtifactList{Run: *run, Latest: latest, Artifacts: make([]IndexedArtifact, 0, len(artifacts))}
	for i, artifact := range artifacts {
		task, err := db.GetTask(ctx, artifact.TaskID)
		if err != nil {
			return nil, err
		}
		item := IndexedArtifact{Index: i + 1, Row: artifact, Task: *task, Kind: "artifact"}
		if summary, ok := summaries[artifact.ID]; ok {
			item.Kind = summary.kind
			item.Source = summary.source
		}
		list.Artifacts = append(list.Artifacts, item)
	}
	return list, nil
}

func ResolveArtifact(ctx context.Context, db *sqlite.DB, runSelector, selector string) (*ArtifactList, IndexedArtifact, error) {
	if strings.HasPrefix(selector, "artifact_") && (runSelector == "" || runSelector == "latest") {
		latest, err := BuildArtifactList(ctx, db, runSelector)
		if err != nil {
			return nil, IndexedArtifact{}, err
		}
		if match, err := resolveArtifactSelector(latest.Artifacts, selector); err == nil {
			return latest, match, nil
		}

		artifact, err := db.GetArtifact(ctx, selector)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, IndexedArtifact{}, apperr.NotFound("artifact not found for %q; run `penta artifacts list`", selector)
			}
			return nil, IndexedArtifact{}, err
		}
		task, err := db.GetTask(ctx, artifact.TaskID)
		if err != nil {
			return nil, IndexedArtifact{}, err
		}
		list, err := BuildArtifactList(ctx, db, task.RunID)
		if err != nil {
			return nil, IndexedArtifact{}, err
		}
		match, err := resolveArtifactSelector(list.Artifacts, selector)
		if err != nil {
			return nil, IndexedArtifact{}, err
		}
		return list, match, nil
	}

	list, err := BuildArtifactList(ctx, db, runSelector)
	if err != nil {
		return nil, IndexedArtifact{}, err
	}
	match, err := resolveArtifactSelector(list.Artifacts, selector)
	if err != nil {
		return nil, IndexedArtifact{}, err
	}
	return list, match, nil
}

type artifactSummary struct {
	kind   string
	source string
}

func artifactSummaries(ctx context.Context, db *sqlite.DB, runID string) (map[string]artifactSummary, error) {
	evidenceRows, err := db.ListEvidenceByRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	summaries := map[string]artifactSummary{}
	for _, row := range evidenceRows {
		if row.Kind != "http_response" {
			continue
		}
		var response model.HTTPResponse
		if err := json.Unmarshal([]byte(row.DataJSON), &response); err != nil {
			return nil, err
		}
		if response.BodyArtifactID == "" {
			continue
		}
		summaries[response.BodyArtifactID] = artifactSummary{kind: "body", source: displayEvidenceURLPath(response.URL)}
	}
	return summaries, nil
}

func resolveArtifactSelector(artifacts []IndexedArtifact, selector string) (IndexedArtifact, error) {
	if selector == "" {
		return IndexedArtifact{}, apperr.InvalidInput("artifact selector is required")
	}
	if index, err := strconv.Atoi(selector); err == nil {
		if index < 1 || index > len(artifacts) {
			return IndexedArtifact{}, apperr.NotFound("artifact index %d not found; run `penta artifacts list`", index)
		}
		return artifacts[index-1], nil
	}
	if matches := matchesArtifactID(artifacts, selector); len(matches) > 0 || strings.HasPrefix(selector, "artifact_") {
		return singleArtifactMatch(selector, matches)
	}
	parts := strings.SplitN(selector, ":", 2)
	if len(parts) != 2 || parts[0] == "" {
		return IndexedArtifact{}, apperr.InvalidInput("unsupported artifact selector %q; run `penta artifacts list`", selector)
	}
	if parts[0] != "body" {
		return IndexedArtifact{}, apperr.InvalidInput("unsupported artifact selector kind %q; run `penta artifacts list`", parts[0])
	}
	return singleArtifactMatch(selector, matchesArtifactKindSource(artifacts, "body", parts[1]))
}

func matchesArtifactID(artifacts []IndexedArtifact, id string) []IndexedArtifact {
	var matches []IndexedArtifact
	for _, item := range artifacts {
		if item.Row.ID == id {
			matches = append(matches, item)
		}
	}
	return matches
}

func matchesArtifactKindSource(artifacts []IndexedArtifact, kind, source string) []IndexedArtifact {
	var matches []IndexedArtifact
	for _, item := range artifacts {
		if item.Kind == kind && item.Source == source {
			matches = append(matches, item)
		}
	}
	return matches
}

func singleArtifactMatch(selector string, matches []IndexedArtifact) (IndexedArtifact, error) {
	if len(matches) == 0 {
		return IndexedArtifact{}, apperr.NotFound("artifact not found for %q; run `penta artifacts list`", selector)
	}
	if len(matches) > 1 {
		return IndexedArtifact{}, apperr.Conflict("ambiguous artifact selector %q matched %d items; run `penta artifacts list` and choose an index", selector, len(matches))
	}
	return matches[0], nil
}
