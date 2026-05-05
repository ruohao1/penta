package http_request

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

func TestExecuteGETCreatesHTTPResponseEvidence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("hello root"))
	}))
	defer server.Close()

	db := openHTTPRequestTestDB(t)
	task := createScopedHTTPRequestTask(t, db, Input{Method: "GET", URL: server.URL, Depth: 1}, hostFromURL(t, server.URL))

	if err := Execute(context.Background(), db, nil, task); err != nil {
		t.Fatalf("execute http request: %v", err)
	}

	evidence := taskHTTPResponseEvidence(t, db, task.ID)
	wantSHA := sha256.Sum256([]byte("hello root"))
	if evidence.URL != server.URL+"/" || evidence.Depth != 1 || evidence.StatusCode != http.StatusAccepted || evidence.ContentType != "text/html" || evidence.BodyBytes != int64(len("hello root")) || evidence.BodyReadLimitBytes != maxBodyBytes || evidence.BodyTruncated || evidence.BodySHA256 != hex.EncodeToString(wantSHA[:]) || evidence.BodyArtifactID == "" {
		t.Fatalf("unexpected evidence: %+v", evidence)
	}
	artifacts, err := db.ListArtifactsByTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("list artifacts: %v", err)
	}
	if len(artifacts) != 1 || artifacts[0].ID != evidence.BodyArtifactID {
		t.Fatalf("unexpected artifacts: %+v", artifacts)
	}
}

func TestExecuteDoesNotStoreBodyArtifactForNonHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	db := openHTTPRequestTestDB(t)
	task := createScopedHTTPRequestTask(t, db, Input{Method: "GET", URL: server.URL}, hostFromURL(t, server.URL))

	if err := Execute(context.Background(), db, nil, task); err != nil {
		t.Fatalf("execute http request: %v", err)
	}
	evidence := taskHTTPResponseEvidence(t, db, task.ID)
	if evidence.BodyArtifactID != "" {
		t.Fatalf("expected no body artifact for non-html response: %+v", evidence)
	}
	artifacts, err := db.ListArtifactsByTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("list artifacts: %v", err)
	}
	if len(artifacts) != 0 {
		t.Fatalf("unexpected artifacts: %+v", artifacts)
	}
}

func TestExecuteCapsAndRedactsResponseHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-Cookie", "session=secret")
		for i := 0; i < maxHeaderCount+1; i++ {
			w.Header().Set(fmt.Sprintf("X-Fill-%d", i), strings.Repeat("v", maxHeaderValueBytes+10))
		}
		_, _ = w.Write([]byte("headers"))
	}))
	defer server.Close()
	db := openHTTPRequestTestDB(t)
	task := createScopedHTTPRequestTask(t, db, Input{Method: "GET", URL: server.URL}, hostFromURL(t, server.URL))

	if err := Execute(context.Background(), db, nil, task); err != nil {
		t.Fatalf("execute http request: %v", err)
	}
	evidence := taskHTTPResponseEvidence(t, db, task.ID)
	if !evidence.HeadersTruncated || len(evidence.Headers) > maxHeaderCount {
		t.Fatalf("expected capped truncated headers: %+v", evidence.Headers)
	}
	for _, header := range evidence.Headers {
		if strings.EqualFold(header.Name, "Set-Cookie") && (len(header.Values) != 1 || header.Values[0] != redactedHeaderValue) {
			t.Fatalf("expected Set-Cookie redacted: %+v", header)
		}
		for _, value := range header.Values {
			if len(value) > maxHeaderValueBytes {
				t.Fatalf("header value was not capped: %d", len(value))
			}
		}
	}
}

func TestExecuteRecordsBodyTruncationMetadata(t *testing.T) {
	body := strings.Repeat("a", maxBodyBytes+1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()
	db := openHTTPRequestTestDB(t)
	task := createScopedHTTPRequestTask(t, db, Input{Method: "GET", URL: server.URL}, hostFromURL(t, server.URL))

	if err := Execute(context.Background(), db, nil, task); err != nil {
		t.Fatalf("execute http request: %v", err)
	}
	evidence := taskHTTPResponseEvidence(t, db, task.ID)
	wantSHA := sha256.Sum256([]byte(body[:maxBodyBytes]))
	if !evidence.BodyTruncated || evidence.BodyBytes != maxBodyBytes || evidence.BodyReadLimitBytes != maxBodyBytes || evidence.BodySHA256 != hex.EncodeToString(wantSHA[:]) {
		t.Fatalf("unexpected truncation evidence: %+v", evidence)
	}
}

func TestExecuteHEADCreatesEmptyBodyHTTPResponseEvidence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	db := openHTTPRequestTestDB(t)
	task := createScopedHTTPRequestTask(t, db, Input{Method: "HEAD", URL: server.URL}, hostFromURL(t, server.URL))

	if err := Execute(context.Background(), db, nil, task); err != nil {
		t.Fatalf("execute HEAD request: %v", err)
	}
	evidence := taskHTTPResponseEvidence(t, db, task.ID)
	emptySHA := sha256.Sum256(nil)
	if evidence.StatusCode != http.StatusNoContent || evidence.BodyBytes != 0 || evidence.BodySHA256 != hex.EncodeToString(emptySHA[:]) {
		t.Fatalf("unexpected HEAD evidence: %+v", evidence)
	}
}

func TestExecuteRejectsUnsupportedMethod(t *testing.T) {
	db := openHTTPRequestTestDB(t)
	task := createHTTPRequestTask(t, db, Input{Method: "POST", URL: "https://example.com/"})

	err := Execute(context.Background(), db, nil, task)
	if err == nil || !strings.Contains(err.Error(), "unsupported HTTP method") {
		t.Fatalf("expected unsupported method error, got %v", err)
	}
}

func TestExecuteRejectsUnsupportedScheme(t *testing.T) {
	db := openHTTPRequestTestDB(t)
	task := createHTTPRequestTask(t, db, Input{Method: "GET", URL: "ftp://example.com/"})

	err := Execute(context.Background(), db, nil, task)
	if err == nil || !strings.Contains(err.Error(), "unsupported URL scheme") {
		t.Fatalf("expected unsupported scheme error, got %v", err)
	}
}

func TestExecuteDoesNotFollowRedirects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://169.254.169.254/", http.StatusFound)
	}))
	defer server.Close()
	db := openHTTPRequestTestDB(t)
	task := createScopedHTTPRequestTask(t, db, Input{Method: "GET", URL: server.URL}, hostFromURL(t, server.URL))

	if err := Execute(context.Background(), db, nil, task); err != nil {
		t.Fatalf("execute http request: %v", err)
	}
	evidence := taskHTTPResponseEvidence(t, db, task.ID)
	if evidence.StatusCode != http.StatusFound || evidence.URL != server.URL+"/" {
		t.Fatalf("unexpected redirect evidence: %+v", evidence)
	}
}

func TestExecuteBlocksRestrictedAddressWithoutScopedSession(t *testing.T) {
	db := openHTTPRequestTestDB(t)
	task := createHTTPRequestTask(t, db, Input{Method: "GET", URL: "http://127.0.0.1:1/"})

	err := Execute(context.Background(), db, nil, task)
	if err == nil || !strings.Contains(err.Error(), "blocked restricted network address 127.0.0.1") {
		t.Fatalf("expected restricted address block, got %v", err)
	}
}

func TestExecuteBlocksHostnameResolvingToRestrictedAddress(t *testing.T) {
	originalResolver := resolver
	resolver = func(context.Context, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
	}
	t.Cleanup(func() { resolver = originalResolver })
	db := openHTTPRequestTestDB(t)
	task := createHTTPRequestTask(t, db, Input{Method: "GET", URL: "http://blocked.test/"})

	err := Execute(context.Background(), db, nil, task)
	if err == nil || !strings.Contains(err.Error(), "blocked restricted network address 127.0.0.1") {
		t.Fatalf("expected restricted resolved address block, got %v", err)
	}
}

func TestRestrictedAddressClassification(t *testing.T) {
	tests := []struct {
		addr       string
		restricted bool
	}{
		{addr: "1.2.3.4", restricted: false},
		{addr: "127.0.0.1", restricted: true},
		{addr: "10.0.0.1", restricted: true},
		{addr: "169.254.169.254", restricted: true},
		{addr: "0.0.0.0", restricted: true},
		{addr: "224.0.0.1", restricted: true},
		{addr: "::1", restricted: true},
		{addr: "fc00::1", restricted: true},
		{addr: "fe80::1", restricted: true},
		{addr: "ff02::1", restricted: true},
		{addr: "::ffff:127.0.0.1", restricted: true},
	}
	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			addr := netip.MustParseAddr(tt.addr)
			if got := restrictedAddress(addr); got != tt.restricted {
				t.Fatalf("restrictedAddress(%s) = %v, want %v", tt.addr, got, tt.restricted)
			}
		})
	}
}

func TestRestrictedAddressAllowedOnlyByExplicitIPOrCIDRScope(t *testing.T) {
	addr := netip.MustParseAddr("127.0.0.1")
	if restrictedAddressAllowedByScope(addr, nil) {
		t.Fatal("empty scope should not explicitly allow restricted address")
	}
	if restrictedAddressAllowedByScope(addr, []sqlite.ScopeRule{{ID: "domain", Effect: sqlite.ScopeEffectInclude, TargetType: sqlite.ScopeTargetDomain, Value: "localhost"}}) {
		t.Fatal("domain scope should not explicitly allow restricted address")
	}
	if !restrictedAddressAllowedByScope(addr, []sqlite.ScopeRule{{ID: "ip", Effect: sqlite.ScopeEffectInclude, TargetType: sqlite.ScopeTargetIP, Value: "127.0.0.1"}}) {
		t.Fatal("matching IP include should allow restricted address")
	}
	if !restrictedAddressAllowedByScope(addr, []sqlite.ScopeRule{{ID: "cidr", Effect: sqlite.ScopeEffectInclude, TargetType: sqlite.ScopeTargetCIDR, Value: "127.0.0.0/8"}}) {
		t.Fatal("matching CIDR include should allow restricted address")
	}
	if restrictedAddressAllowedByScope(addr, []sqlite.ScopeRule{
		{ID: "include", Effect: sqlite.ScopeEffectInclude, TargetType: sqlite.ScopeTargetCIDR, Value: "127.0.0.0/8"},
		{ID: "exclude", Effect: sqlite.ScopeEffectExclude, TargetType: sqlite.ScopeTargetIP, Value: "127.0.0.1"},
	}) {
		t.Fatal("exclude should override include for restricted address")
	}
}

func taskHTTPResponseEvidence(t *testing.T, db *sqlite.DB, taskID string) Evidence {
	t.Helper()
	evidenceRows, err := db.ListEvidenceByTask(context.Background(), taskID)
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
	return evidence
}

func openHTTPRequestTestDB(t *testing.T) *sqlite.DB {
	t.Helper()
	db, err := sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "penta.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func createHTTPRequestTask(t *testing.T, db *sqlite.DB, input Input) *sqlite.Task {
	t.Helper()
	run := sqlite.Run{ID: "run_http", Mode: "recon", Status: actions.RunStatusRunning, CreatedAt: time.Now()}
	return createHTTPRequestTaskWithRun(t, db, input, run)
}

func createScopedHTTPRequestTask(t *testing.T, db *sqlite.DB, input Input, scopeIP string) *sqlite.Task {
	t.Helper()
	session := sqlite.Session{ID: "session_http", Name: "HTTP", Kind: sqlite.SessionKindLab, Status: sqlite.SessionStatusActive, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := db.CreateSession(context.Background(), session); err != nil {
		t.Fatalf("create session: %v", err)
	}
	rule := sqlite.ScopeRule{ID: "scope_http", SessionID: session.ID, Effect: sqlite.ScopeEffectInclude, TargetType: sqlite.ScopeTargetIP, Value: scopeIP, CreatedAt: time.Now()}
	if err := db.CreateScopeRule(context.Background(), rule); err != nil {
		t.Fatalf("create scope rule: %v", err)
	}
	run := sqlite.Run{ID: "run_http", SessionID: session.ID, Mode: "recon", Status: actions.RunStatusRunning, CreatedAt: time.Now()}
	return createHTTPRequestTaskWithRun(t, db, input, run)
}

func createHTTPRequestTaskWithRun(t *testing.T, db *sqlite.DB, input Input, run sqlite.Run) *sqlite.Task {
	t.Helper()
	if err := db.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	task := sqlite.Task{ID: "task_http", RunID: run.ID, ActionType: actions.ActionHTTPRequest, InputJSON: string(inputJSON), Status: actions.TaskStatusRunning, CreatedAt: time.Now()}
	if err := db.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	return &task
}

func hostFromURL(t *testing.T, rawURL string) string {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	return parsed.Hostname()
}
