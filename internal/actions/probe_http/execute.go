package probe_http

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
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

	service := Evidence{}
	switch input.Type {
	case targets.TypeURL:
		parsed, err := targets.Parse(input.Value)
		if err != nil {
			return err
		}
		urlTarget, ok := parsed.(*targets.URL)
		if !ok {
			return fmt.Errorf("expected url target")
		}
		service.Host = urlTarget.Host
		service.Scheme = urlTarget.Scheme
		service.Port, err = defaultPort(urlTarget.Scheme, urlTarget.Port)
		if err != nil {
			return err
		}
	case targets.TypeDomain, targets.TypeIP:
		service.Host = input.Value
		service.Scheme = "https"
		service.Port = 443
	case targets.TypeService:
		parsed, err := targets.Parse(input.Value)
		if err != nil {
			return err
		}
		serviceTarget, ok := parsed.(*targets.Service)
		if !ok {
			return fmt.Errorf("expected service target")
		}
		service.Host = serviceTarget.Host
		service.Port, err = defaultPort("http", serviceTarget.Port)
		if err != nil {
			return err
		}
		service.Scheme = schemeForServicePort(service.Port)
	default:
		return fmt.Errorf("unsupported target type: %s", input.Type)
	}

	evidenceJSON, err := json.Marshal(service)
	if err != nil {
		return err
	}

	evidence := sqlite.Evidence{
		ID:        "evidence_" + uuid.NewString(),
		RunID:     task.RunID,
		TaskID:    task.ID,
		Kind:      "service",
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

func schemeForServicePort(port int) string {
	if port == 443 {
		return "https"
	}
	return "http"
}

func defaultPort(scheme, port string) (int, error) {
	if port != "" {
		parsed, err := strconv.Atoi(port)
		if err != nil {
			return 0, fmt.Errorf("invalid port %q", port)
		}
		if parsed < 1 || parsed > 65535 {
			return 0, fmt.Errorf("port out of range: %d", parsed)
		}
		return parsed, nil
	}

	switch scheme {
	case "http":
		return 80, nil
	case "https":
		return 443, nil
	default:
		return 0, fmt.Errorf("unsupported scheme %q", scheme)
	}
}

func mustPayloadJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}
