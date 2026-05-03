package resolve_dns

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ruohao1/penta/internal/events"
	"github.com/ruohao1/penta/internal/model"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

type Resolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

var defaultResolver Resolver = net.DefaultResolver

func Execute(ctx context.Context, db *sqlite.DB, sink events.Sink, task *sqlite.Task) error {
	return executeWithResolver(ctx, db, sink, task, defaultResolver)
}

func executeWithResolver(ctx context.Context, db *sqlite.DB, sink events.Sink, task *sqlite.Task, resolver Resolver) error {
	var input Input
	if err := json.Unmarshal([]byte(task.InputJSON), &input); err != nil {
		return err
	}
	input.Domain = strings.TrimSpace(input.Domain)
	if input.Domain == "" {
		return fmt.Errorf("domain is required")
	}

	addrs, err := resolver.LookupIPAddr(ctx, input.Domain)
	if err != nil {
		return err
	}

	records := make([]model.DNSRecord, 0, len(addrs))
	for _, addr := range addrs {
		recordType := "A"
		if addr.IP.To4() == nil {
			recordType = "AAAA"
		}
		records = append(records, model.DNSRecord{
			Name:  input.Domain,
			Type:  recordType,
			Value: addr.IP.String(),
		})
	}

	evidenceJSON, err := json.Marshal(Evidence{Records: records})
	if err != nil {
		return err
	}
	evidence := sqlite.Evidence{
		ID:        "evidence_" + uuid.NewString(),
		RunID:     task.RunID,
		TaskID:    task.ID,
		Kind:      "dns_record",
		DataJSON:  string(evidenceJSON),
		CreatedAt: time.Now(),
	}
	if err := db.CreateEvidence(ctx, evidence); err != nil {
		return err
	}
	if sink == nil {
		return nil
	}

	return sink.Append(ctx, events.Event{
		RunID:       task.RunID,
		EventType:   events.EventEvidenceCreated,
		EntityKind:  events.EntityEvidence,
		EntityID:    evidence.ID,
		PayloadJSON: mustPayloadJSON(events.EvidenceCreatedPayload{Kind: evidence.Kind}),
		CreatedAt:   time.Now(),
	})
}

func mustPayloadJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}
