package fetch_root

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

func TestExecuteFetchesRootAndCreatesHTTPResponseEvidence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("hello root"))
	}))
	defer server.Close()

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}
	db := openFetchRootTestDB(t)
	task := createFetchRootTask(t, db, Input{Scheme: parsed.Scheme, Host: parsed.Hostname(), Port: port})

	if err := Execute(context.Background(), db, nil, task); err != nil {
		t.Fatalf("execute fetch root: %v", err)
	}

	evidenceRows, err := db.ListEvidenceByTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("list evidence: %v", err)
	}
	if len(evidenceRows) != 1 {
		t.Fatalf("unexpected evidence count: %d", len(evidenceRows))
	}
	if evidenceRows[0].Kind != string(actions.EvidenceHTTPResponse) {
		t.Fatalf("unexpected evidence kind: %s", evidenceRows[0].Kind)
	}
	var evidence Evidence
	if err := json.Unmarshal([]byte(evidenceRows[0].DataJSON), &evidence); err != nil {
		t.Fatalf("unmarshal evidence: %v", err)
	}
	wantSHA := sha256.Sum256([]byte("hello root"))
	if evidence.URL != server.URL+"/" || evidence.StatusCode != http.StatusAccepted || evidence.ContentType != "text/html" || evidence.BodyBytes != int64(len("hello root")) || evidence.BodySHA256 != hex.EncodeToString(wantSHA[:]) {
		t.Fatalf("unexpected evidence: %+v", evidence)
	}
}

func TestRootURLRejectsUnsupportedScheme(t *testing.T) {
	_, err := rootURL(Input{Scheme: "ftp", Host: "example.com", Port: 21})
	if err == nil {
		t.Fatal("expected unsupported scheme to fail")
	}
}

func TestRootURLRejectsOutOfRangePort(t *testing.T) {
	_, err := rootURL(Input{Scheme: "https", Host: "example.com", Port: 70000})
	if err == nil {
		t.Fatal("expected out-of-range port to fail")
	}
}

func TestExecuteDoesNotFollowRedirects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://169.254.169.254/", http.StatusFound)
	}))
	defer server.Close()

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}
	db := openFetchRootTestDB(t)
	task := createFetchRootTask(t, db, Input{Scheme: parsed.Scheme, Host: parsed.Hostname(), Port: port})

	if err := Execute(context.Background(), db, nil, task); err != nil {
		t.Fatalf("execute fetch root: %v", err)
	}
	evidenceRows, err := db.ListEvidenceByTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("list evidence: %v", err)
	}
	var evidence Evidence
	if err := json.Unmarshal([]byte(evidenceRows[0].DataJSON), &evidence); err != nil {
		t.Fatalf("unmarshal evidence: %v", err)
	}
	if evidence.StatusCode != http.StatusFound || evidence.URL != server.URL+"/" {
		t.Fatalf("unexpected redirect evidence: %+v", evidence)
	}
}

func openFetchRootTestDB(t *testing.T) *sqlite.DB {
	t.Helper()
	db, err := sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "penta.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func createFetchRootTask(t *testing.T, db *sqlite.DB, input Input) *sqlite.Task {
	t.Helper()
	run := sqlite.Run{ID: "run_fetch", Mode: "recon", Status: actions.RunStatusRunning, CreatedAt: time.Now()}
	if err := db.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	task := sqlite.Task{ID: "task_fetch", RunID: run.ID, ActionType: actions.ActionFetchRoot, InputJSON: string(inputJSON), Status: actions.TaskStatusRunning, CreatedAt: time.Now()}
	if err := db.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	return &task
}
