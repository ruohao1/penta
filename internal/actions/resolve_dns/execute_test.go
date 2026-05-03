package resolve_dns

import (
	"context"
	"encoding/json"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/model"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

type fakeResolver struct {
	addrs []net.IPAddr
	err   error
}

func (r fakeResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) {
	return r.addrs, r.err
}

func TestExecuteCreatesDNSRecordEvidence(t *testing.T) {
	db := openResolveDNSTestDB(t)
	ctx := context.Background()
	run := sqlite.Run{ID: "run_1", Mode: "recon", Status: actions.RunStatusRunning, CreatedAt: time.Now()}
	if err := db.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	inputJSON := mustMarshalTestJSON(t, Input{Domain: "example.com"})
	task := sqlite.Task{ID: "task_dns", RunID: run.ID, ActionType: actions.ActionResolveDNS, InputJSON: inputJSON, Status: actions.TaskStatusPending, CreatedAt: time.Now()}
	if err := db.CreateTask(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	resolver := fakeResolver{addrs: []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}, {IP: net.ParseIP("2606:2800:220:1:248:1893:25c8:1946")}}}
	if err := executeWithResolver(ctx, db, nil, &task, resolver); err != nil {
		t.Fatalf("execute resolve dns: %v", err)
	}

	evidenceRows, err := db.ListEvidenceByTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("list evidence: %v", err)
	}
	if len(evidenceRows) != 1 {
		t.Fatalf("unexpected evidence count: got %d want 1", len(evidenceRows))
	}
	if evidenceRows[0].Kind != string(actions.EvidenceDNSRecord) {
		t.Fatalf("unexpected evidence kind: got %q want %q", evidenceRows[0].Kind, actions.EvidenceDNSRecord)
	}

	var payload Evidence
	if err := json.Unmarshal([]byte(evidenceRows[0].DataJSON), &payload); err != nil {
		t.Fatalf("unmarshal evidence: %v", err)
	}
	want := []model.DNSRecord{
		{Name: "example.com", Type: "A", Value: "93.184.216.34"},
		{Name: "example.com", Type: "AAAA", Value: "2606:2800:220:1:248:1893:25c8:1946"},
	}
	if len(payload.Records) != len(want) {
		t.Fatalf("unexpected record count: got %d want %d", len(payload.Records), len(want))
	}
	for i := range want {
		if payload.Records[i] != want[i] {
			t.Fatalf("unexpected record at %d: got %+v want %+v", i, payload.Records[i], want[i])
		}
	}
}

func openResolveDNSTestDB(t *testing.T) *sqlite.DB {
	t.Helper()

	db, err := sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "penta.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func mustMarshalTestJSON(t *testing.T, v any) string {
	t.Helper()

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return string(data)
}
