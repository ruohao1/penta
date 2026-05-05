package scheduler

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	httprequest "github.com/ruohao1/penta/internal/actions/http_request"
	probehttp "github.com/ruohao1/penta/internal/actions/probe_http"
	resolvedns "github.com/ruohao1/penta/internal/actions/resolve_dns"
	"github.com/ruohao1/penta/internal/model"
	"github.com/ruohao1/penta/internal/storage/sqlite"
	"github.com/ruohao1/penta/internal/targets"
)

func TestDeriveFromTargetEvidenceCreatesResolveDNSAndProbeHTTPForDomain(t *testing.T) {
	candidates := deriveTargetCandidates(t, model.TargetRef{Value: "example.com", Type: targets.TypeDomain})
	assertResolveDNSCandidate(t, candidates, "example.com")
	assertProbeHTTPCandidate(t, candidates, "example.com", targets.TypeDomain)
}

func TestDeriveFromTargetEvidenceCreatesProbeHTTPForIP(t *testing.T) {
	candidates := deriveTargetCandidates(t, model.TargetRef{Value: "1.2.3.4", Type: targets.TypeIP})
	assertProbeHTTPCandidate(t, candidates, "1.2.3.4", targets.TypeIP)
}

func TestDeriveFromTargetEvidenceCreatesProbeHTTPForURL(t *testing.T) {
	candidates := deriveTargetCandidates(t, model.TargetRef{Value: "https://example.com/login", Type: targets.TypeURL})
	assertProbeHTTPCandidate(t, candidates, "https://example.com/login", targets.TypeURL)
}

func TestDeriveFromTargetEvidenceCreatesProbeHTTPForService(t *testing.T) {
	candidates := deriveTargetCandidates(t, model.TargetRef{Value: "127.0.0.1:8000", Type: targets.TypeService})
	assertProbeHTTPCandidate(t, candidates, "127.0.0.1:8000", targets.TypeService)
}

func TestDeriveFromTargetEvidenceIgnoresCIDR(t *testing.T) {
	candidates := deriveTargetCandidates(t, model.TargetRef{Value: "10.0.0.0/24", Type: targets.TypeCIDR})
	if len(candidates) != 0 {
		t.Fatalf("unexpected candidates: %+v", candidates)
	}
}

func TestDeriveFromEvidenceIgnoresNonTargetEvidence(t *testing.T) {
	candidates, err := DeriveFromEvidence(sqlite.Evidence{
		ID:        "evidence_dns",
		RunID:     "run_1",
		TaskID:    "task_1",
		Kind:      string(actions.EvidenceDNSRecord),
		DataJSON:  `{"domain":"example.com","records":["1.2.3.4"]}`,
		CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("derive non-target evidence: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("unexpected candidates: %+v", candidates)
	}
}

func TestDeriveFromServiceEvidenceCreatesHTTPRequest(t *testing.T) {
	service := model.Service{Scheme: "https", Host: "example.com", Port: 443}
	data, err := json.Marshal(service)
	if err != nil {
		t.Fatalf("marshal service: %v", err)
	}
	candidates, err := DeriveFromEvidence(sqlite.Evidence{ID: "evidence_service", RunID: "run_1", TaskID: "task_1", Kind: string(actions.EvidenceService), DataJSON: string(data), CreatedAt: time.Now()})
	if err != nil {
		t.Fatalf("derive service evidence: %v", err)
	}
	candidate := findCandidate(t, candidates, actions.ActionHTTPRequest)
	if len(candidate.ParentEvidenceIDs) != 1 || candidate.ParentEvidenceIDs[0] != "evidence_service" {
		t.Fatalf("unexpected parent evidence IDs: %+v", candidate.ParentEvidenceIDs)
	}
	var input httprequest.Input
	if err := json.Unmarshal([]byte(candidate.InputJSON), &input); err != nil {
		t.Fatalf("unmarshal http request input: %v", err)
	}
	if input.Method != "GET" || input.URL != "https://example.com:443/" || input.Depth != 0 {
		t.Fatalf("unexpected http request input: %+v", input)
	}
	if candidate.Depth != 0 || candidate.CrawlDerived {
		t.Fatalf("unexpected root request metadata: %+v", candidate)
	}
	assertCandidateTarget(t, candidate, "https://example.com:443/", targets.TypeURL)
}

func TestDeriveFromHTTPResponseEvidenceCreatesCrawl(t *testing.T) {
	response := model.HTTPResponse{URL: "https://example.com/", Depth: 1, ContentType: "text/html", BodyArtifactID: "artifact_1"}
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	candidates, err := DeriveFromEvidence(sqlite.Evidence{ID: "evidence_http", RunID: "run_1", TaskID: "task_1", Kind: string(actions.EvidenceHTTPResponse), DataJSON: string(data), CreatedAt: time.Now()})
	if err != nil {
		t.Fatalf("derive http response evidence: %v", err)
	}
	candidate := findCandidate(t, candidates, actions.ActionCrawl)
	if len(candidate.ParentEvidenceIDs) != 1 || candidate.ParentEvidenceIDs[0] != "evidence_http" {
		t.Fatalf("unexpected parent evidence IDs: %+v", candidate.ParentEvidenceIDs)
	}
	var input model.HTTPResponse
	if err := json.Unmarshal([]byte(candidate.InputJSON), &input); err != nil {
		t.Fatalf("unmarshal crawl input: %v", err)
	}
	if input.URL != response.URL || input.BodyArtifactID != response.BodyArtifactID {
		t.Fatalf("unexpected crawl input: %+v", input)
	}
	if input.Depth != 1 || candidate.Depth != 1 {
		t.Fatalf("unexpected crawl depth: input=%+v candidate=%+v", input, candidate)
	}
	assertCandidateTarget(t, candidate, "https://example.com/", targets.TypeURL)
}

func TestDeriveFromCrawlEvidenceCreatesHTTPRequests(t *testing.T) {
	result := model.CrawlResult{SourceURL: "https://example.com/", Depth: 0, URLs: []string{"https://example.com/login", "https://example.com/help"}}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal crawl result: %v", err)
	}
	candidates, err := DeriveFromEvidence(sqlite.Evidence{ID: "evidence_crawl", RunID: "run_1", TaskID: "task_1", Kind: string(actions.EvidenceCrawl), DataJSON: string(data), CreatedAt: time.Now()})
	if err != nil {
		t.Fatalf("derive crawl evidence: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("unexpected candidates: %+v", candidates)
	}
	for _, candidate := range candidates {
		if candidate.ActionType != actions.ActionHTTPRequest {
			t.Fatalf("unexpected action type: %s", candidate.ActionType)
		}
		var input httprequest.Input
		if err := json.Unmarshal([]byte(candidate.InputJSON), &input); err != nil {
			t.Fatalf("unmarshal http request input: %v", err)
		}
		if input.Method != "GET" || input.URL == "" || input.Depth != 1 {
			t.Fatalf("unexpected http request input: %+v", input)
		}
		if !candidate.CrawlDerived || candidate.Depth != 1 {
			t.Fatalf("unexpected crawl candidate metadata: %+v", candidate)
		}
		assertCandidateTarget(t, candidate, input.URL, targets.TypeURL)
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

func deriveTargetCandidates(t *testing.T, target model.TargetRef) []CandidateTask {
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

	candidate := findCandidate(t, candidates, actions.ActionProbeHTTP)
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
	assertCandidateTarget(t, candidate, value, targetType)
}

func assertResolveDNSCandidate(t *testing.T, candidates []CandidateTask, domain string) {
	t.Helper()

	candidate := findCandidate(t, candidates, actions.ActionResolveDNS)
	if len(candidate.ParentEvidenceIDs) != 1 || candidate.ParentEvidenceIDs[0] != "evidence_target" {
		t.Fatalf("unexpected parent evidence IDs: %+v", candidate.ParentEvidenceIDs)
	}
	if candidate.Reason == "" {
		t.Fatal("expected candidate reason")
	}

	var input resolvedns.Input
	if err := json.Unmarshal([]byte(candidate.InputJSON), &input); err != nil {
		t.Fatalf("unmarshal resolve dns input: %v", err)
	}
	if input.Domain != domain {
		t.Fatalf("unexpected resolve dns input: %+v", input)
	}
	assertCandidateTarget(t, candidate, domain, targets.TypeDomain)
}

func assertCandidateTarget(t *testing.T, candidate CandidateTask, value string, targetType targets.Type) {
	t.Helper()
	if candidate.Target == nil {
		t.Fatalf("candidate missing target metadata: %+v", candidate)
	}
	if candidate.Target.Value != value || candidate.Target.Type != targetType {
		t.Fatalf("unexpected candidate target: %+v", candidate.Target)
	}
}

func findCandidate(t *testing.T, candidates []CandidateTask, actionType actions.ActionType) CandidateTask {
	t.Helper()

	for _, candidate := range candidates {
		if candidate.ActionType == actionType {
			return candidate
		}
	}
	t.Fatalf("candidate action %q not found in %+v", actionType, candidates)
	return CandidateTask{}
}
