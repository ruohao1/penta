package scheduler

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	probehttp "github.com/ruohao1/penta/internal/actions/probe_http"
	"github.com/ruohao1/penta/internal/storage/sqlite"
	"github.com/ruohao1/penta/internal/targets"
)

func TestDeriveFromTargetEvidenceCreatesProbeHTTPForDomain(t *testing.T) {
	candidates := deriveTargetCandidates(t, targets.TargetRef{Value: "example.com", Type: targets.TypeDomain})
	assertProbeHTTPCandidate(t, candidates, "example.com", targets.TypeDomain)
}

func TestDeriveFromTargetEvidenceCreatesProbeHTTPForIP(t *testing.T) {
	candidates := deriveTargetCandidates(t, targets.TargetRef{Value: "1.2.3.4", Type: targets.TypeIP})
	assertProbeHTTPCandidate(t, candidates, "1.2.3.4", targets.TypeIP)
}

func TestDeriveFromTargetEvidenceCreatesProbeHTTPForURL(t *testing.T) {
	candidates := deriveTargetCandidates(t, targets.TargetRef{Value: "https://example.com/login", Type: targets.TypeURL})
	assertProbeHTTPCandidate(t, candidates, "https://example.com/login", targets.TypeURL)
}

func TestDeriveFromTargetEvidenceIgnoresCIDR(t *testing.T) {
	candidates := deriveTargetCandidates(t, targets.TargetRef{Value: "10.0.0.0/24", Type: targets.TypeCIDR})
	if len(candidates) != 0 {
		t.Fatalf("unexpected candidates: %+v", candidates)
	}
}

func TestDeriveFromEvidenceIgnoresNonTargetEvidence(t *testing.T) {
	candidates, err := DeriveFromEvidence(sqlite.Evidence{
		ID:        "evidence_service",
		RunID:     "run_1",
		TaskID:    "task_1",
		Kind:      string(actions.EvidenceService),
		DataJSON:  `{"host":"example.com","scheme":"https","port":443}`,
		CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("derive non-target evidence: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("unexpected candidates: %+v", candidates)
	}
}

func TestDeriveFromEvidenceRejectsInvalidTargetJSON(t *testing.T) {
	_, err := DeriveFromEvidence(sqlite.Evidence{
		ID:        "evidence_bad",
		RunID:     "run_1",
		TaskID:    "task_1",
		Kind:      string(actions.EvidenceTarget),
		DataJSON:  `{"value":`,
		CreatedAt: time.Now(),
	})
	if err == nil {
		t.Fatal("expected invalid target evidence to fail")
	}
}

func deriveTargetCandidates(t *testing.T, target targets.TargetRef) []CandidateTask {
	t.Helper()

	data, err := json.Marshal(target)
	if err != nil {
		t.Fatalf("marshal target: %v", err)
	}
	candidates, err := DeriveFromEvidence(sqlite.Evidence{
		ID:        "evidence_target",
		RunID:     "run_1",
		TaskID:    "task_1",
		Kind:      string(actions.EvidenceTarget),
		DataJSON:  string(data),
		CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("derive target evidence: %v", err)
	}
	return candidates
}

func assertProbeHTTPCandidate(t *testing.T, candidates []CandidateTask, value string, targetType targets.Type) {
	t.Helper()

	if len(candidates) != 1 {
		t.Fatalf("unexpected candidate count: got %d want 1", len(candidates))
	}
	candidate := candidates[0]
	if candidate.ActionType != actions.ActionProbeHTTP {
		t.Fatalf("unexpected candidate action: got %q want %q", candidate.ActionType, actions.ActionProbeHTTP)
	}
	if len(candidate.ParentEvidenceIDs) != 1 || candidate.ParentEvidenceIDs[0] != "evidence_target" {
		t.Fatalf("unexpected parent evidence IDs: %+v", candidate.ParentEvidenceIDs)
	}
	if candidate.Reason == "" {
		t.Fatal("expected candidate reason")
	}

	var input probehttp.Input
	if err := json.Unmarshal([]byte(candidate.InputJSON), &input); err != nil {
		t.Fatalf("unmarshal candidate input: %v", err)
	}
	if input.Value != value || input.Type != targetType {
		t.Fatalf("unexpected candidate input: %+v", input)
	}
}
