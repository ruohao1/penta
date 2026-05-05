package crawl

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/model"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

func TestExtractLinksKeepsSameOriginHTTPLinksAndSkipsSelf(t *testing.T) {
	body := `<html><a href="/login#top">Login</a><a href="settings">Settings</a><a href="/app/">Self</a><a href="https://evil.test/">Evil</a><a href="mailto:a@example.com">Mail</a><a href="javascript:alert(1)">JS</a></html>`
	links := extractLinks("http://example.com/app/", body, maxDiscoveredURLs)
	want := []string{"http://example.com/app/settings", "http://example.com/login"}
	if strings.Join(links, "\n") != strings.Join(want, "\n") {
		t.Fatalf("unexpected links: %+v", links)
	}
}

func TestExtractLinksCapsResults(t *testing.T) {
	var body strings.Builder
	for i := 0; i < maxDiscoveredURLs+10; i++ {
		body.WriteString(`<a href="/`)
		body.WriteString(string(rune('a' + i%26)))
		body.WriteString(`">x</a>`)
	}
	links := extractLinks("http://example.com/", body.String(), 5)
	if len(links) != 5 {
		t.Fatalf("unexpected link count: %d", len(links))
	}
}

func TestExecuteCreatesCrawlEvidenceFromHTMLArtifact(t *testing.T) {
	db := openCrawlTestDB(t)
	bodyPath := filepath.Join(t.TempDir(), "body.html")
	if err := os.WriteFile(bodyPath, []byte(`<a href="/login">Login</a>`), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}
	task := createCrawlTask(t, db, Input{URL: "http://example.com/", Depth: 1, ContentType: "text/html", BodyArtifactID: "artifact_body"})
	if err := db.CreateArtifact(context.Background(), sqlite.Artifact{ID: "artifact_body", TaskID: task.ID, Path: bodyPath, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("create artifact: %v", err)
	}

	if err := Execute(context.Background(), db, nil, task); err != nil {
		t.Fatalf("execute crawl: %v", err)
	}
	evidenceRows, err := db.ListEvidenceByTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("list evidence: %v", err)
	}
	if len(evidenceRows) != 1 || evidenceRows[0].Kind != string(actions.EvidenceCrawl) {
		t.Fatalf("unexpected evidence rows: %+v", evidenceRows)
	}
	var evidence Evidence
	if err := json.Unmarshal([]byte(evidenceRows[0].DataJSON), &evidence); err != nil {
		t.Fatalf("unmarshal crawl evidence: %v", err)
	}
	if evidence.SourceURL != "http://example.com/" || evidence.Depth != 1 || len(evidence.URLs) != 1 || evidence.URLs[0] != "http://example.com/login" {
		t.Fatalf("unexpected crawl evidence: %+v", evidence)
	}
}

func TestExecuteIgnoresNonHTMLOrMissingArtifact(t *testing.T) {
	db := openCrawlTestDB(t)
	task := createCrawlTask(t, db, Input{URL: "http://example.com/", ContentType: "application/json"})
	if err := Execute(context.Background(), db, nil, task); err != nil {
		t.Fatalf("execute crawl: %v", err)
	}
	evidenceRows, err := db.ListEvidenceByTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("list evidence: %v", err)
	}
	if len(evidenceRows) != 0 {
		t.Fatalf("unexpected evidence rows: %+v", evidenceRows)
	}
}

func openCrawlTestDB(t *testing.T) *sqlite.DB {
	t.Helper()
	db, err := sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "penta.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func createCrawlTask(t *testing.T, db *sqlite.DB, input Input) *sqlite.Task {
	t.Helper()
	run := sqlite.Run{ID: "run_crawl", Mode: "recon", Status: actions.RunStatusRunning, CreatedAt: time.Now()}
	if err := db.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	task := sqlite.Task{ID: "task_crawl", RunID: run.ID, ActionType: actions.ActionCrawl, InputJSON: string(inputJSON), Status: actions.TaskStatusRunning, CreatedAt: time.Now()}
	if err := db.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	return &task
}

var _ = model.CrawlResult{}
