package viewmodel

import (
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/storage/sqlite"
)

func TestEvidenceLabelFormatsTarget(t *testing.T) {
	label := evidenceLabelForTest(t, sqlite.Evidence{Kind: "target", DataJSON: `{"value":"example.com","type":"domain"}`})
	if label != "domain example.com" {
		t.Fatalf("unexpected label: %q", label)
	}
}

func TestEvidenceLabelFormatsService(t *testing.T) {
	label := evidenceLabelForTest(t, sqlite.Evidence{Kind: "service", DataJSON: `{"scheme":"https","host":"example.com","port":443}`})
	if label != "https://example.com" {
		t.Fatalf("unexpected label: %q", label)
	}
}

func TestEvidenceSummaryFormatsServiceURLWithNonDefaultPort(t *testing.T) {
	summary := evidenceSummaryForTest(t, sqlite.Evidence{Kind: "service", DataJSON: `{"scheme":"https","host":"example.com","port":8443}`})
	if summary.Label != "https://example.com:8443" || summary.URL != "https://example.com:8443" {
		t.Fatalf("unexpected service summary: %+v", summary)
	}
}

func TestEvidenceLabelFormatsDNSRecords(t *testing.T) {
	label := evidenceLabelForTest(t, sqlite.Evidence{Kind: "dns_record", DataJSON: `{"records":[{"name":"example.com","type":"A","value":"93.184.216.34"}]}`})
	if label != "example.com" {
		t.Fatalf("unexpected label: %q", label)
	}
}

func TestEvidenceSummaryFormatsDNSRecordsAsDetails(t *testing.T) {
	summary := evidenceSummaryForTest(t, sqlite.Evidence{Kind: "dns_record", DataJSON: `{"records":[{"name":"example.com","type":"A","value":"93.184.216.34"},{"name":"www.example.com","type":"CNAME","value":"example.com"}]}`})
	if summary.Label != "example.com, www.example.com" {
		t.Fatalf("unexpected dns summary label: %+v", summary)
	}
	wantDetails := []string{"A example.com -> 93.184.216.34", "CNAME www.example.com -> example.com"}
	if len(summary.Details) != len(wantDetails) {
		t.Fatalf("unexpected dns details: %+v", summary.Details)
	}
	for i, want := range wantDetails {
		if summary.Details[i] != want {
			t.Fatalf("unexpected dns detail %d: got %q want %q", i, summary.Details[i], want)
		}
	}
}

func TestEvidenceSummaryFormatsHTTPResponse(t *testing.T) {
	summary := evidenceSummaryForTest(t, sqlite.Evidence{Kind: "http_response", DataJSON: `{"url":"https://example.com","status_code":200,"content_type":"text/html","body_bytes":512,"body_sha256":"abc123","body_artifact_id":"artifact_1"}`})
	if summary.Label != "https://example.com 200" || summary.URL != "https://example.com" {
		t.Fatalf("unexpected http response summary: %+v", summary)
	}
	wantDetails := []string{"content-type: text/html", "body: 512 bytes", "sha256: abc123", "body artifact: artifact_1"}
	for i, want := range wantDetails {
		if summary.Details[i] != want {
			t.Fatalf("unexpected http detail %d: got %q want %q", i, summary.Details[i], want)
		}
	}
}

func evidenceLabelForTest(t *testing.T, evidence sqlite.Evidence) string {
	t.Helper()
	summary := evidenceSummaryForTest(t, evidence)
	return summary.Label
}

func evidenceSummaryForTest(t *testing.T, evidence sqlite.Evidence) EvidenceSummary {
	t.Helper()
	evidence.ID = "evidence_1"
	evidence.RunID = "run_1"
	evidence.TaskID = "task_1"
	evidence.CreatedAt = time.Now()
	summary, err := EvidenceSummaryFor(evidence)
	if err != nil {
		t.Fatalf("evidence summary: %v", err)
	}
	return summary
}
