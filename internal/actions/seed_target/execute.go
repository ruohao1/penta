package seed_target

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/ruohao1/penta/internal/events"
	"github.com/ruohao1/penta/internal/storage/sqlite"
	"github.com/ruohao1/penta/internal/targets"
	"github.com/ruohao1/penta/internal/viewmodel"
)

func Execute(ctx context.Context, db *sqlite.DB, sink events.Sink, task *sqlite.Task) error {
	var input Input
	if err := json.Unmarshal([]byte(task.InputJSON), &input); err != nil {
		return err
	}

	target, err := targets.Parse(input.Raw)
	if err != nil {
		return err
	}

	evidenceData := Evidence{
		Value: target.String(),
		Type:  target.Type(),
	}
	evidenceJSON, err := json.Marshal(evidenceData)
	if err != nil {
		return err
	}

	evidence := sqlite.Evidence{
		ID:        "evidence_" + uuid.NewString(),
		RunID:     task.RunID,
		TaskID:    task.ID,
		Kind:      "target",
		DataJSON:  string(evidenceJSON),
		CreatedAt: time.Now(),
	}
	label, err := viewmodel.EvidenceLabel(evidence)
	if err != nil {
		return err
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
		PayloadJSON: mustPayloadJSON(events.EvidenceCreatedPayload{Kind: evidence.Kind, Label: label}),
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
