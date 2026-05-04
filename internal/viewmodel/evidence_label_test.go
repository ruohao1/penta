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
	if label != "https example.com:443" {
		t.Fatalf("unexpected label: %q", label)
	}
}

func TestEvidenceLabelFormatsDNSRecords(t *testing.T) {
	label := evidenceLabelForTest(t, sqlite.Evidence{Kind: "dns_record", DataJSON: `{"records":[{"name":"example.com","type":"A","value":"93.184.216.34"}]}`})
	if label != "A example.com -> 93.184.216.34" {
		t.Fatalf("unexpected label: %q", label)
	}
}

func evidenceLabelForTest(t *testing.T, evidence sqlite.Evidence) string {
	t.Helper()
	evidence.ID = "evidence_1"
	evidence.RunID = "run_1"
	evidence.TaskID = "task_1"
	evidence.CreatedAt = time.Now()
	label, err := EvidenceLabel(evidence)
	if err != nil {
		t.Fatalf("evidence label: %v", err)
	}
	return label
}
