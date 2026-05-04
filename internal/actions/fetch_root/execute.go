package fetch_root

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/events"
	"github.com/ruohao1/penta/internal/model"
	"github.com/ruohao1/penta/internal/storage/sqlite"
	"github.com/ruohao1/penta/internal/viewmodel"
)

const maxBodyBytes = 1 << 20

var httpClient = newHTTPClient()

func newHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func Execute(ctx context.Context, db *sqlite.DB, sink events.Sink, task *sqlite.Task) error {
	var input Input
	if err := json.Unmarshal([]byte(task.InputJSON), &input); err != nil {
		return err
	}
	rootURL, err := rootURL(input)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rootURL, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch root %s: %w", rootURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes+1))
	if err != nil {
		return fmt.Errorf("read root response %s: %w", rootURL, err)
	}
	bodyBytes := int64(len(body))
	if bodyBytes > maxBodyBytes {
		body = body[:maxBodyBytes]
		bodyBytes = maxBodyBytes
	}
	sum := sha256.Sum256(body)
	evidenceData := Evidence{
		URL:         rootURL,
		StatusCode:  resp.StatusCode,
		Headers:     headers(resp.Header),
		ContentType: resp.Header.Get("Content-Type"),
		BodyBytes:   bodyBytes,
		BodySHA256:  hex.EncodeToString(sum[:]),
	}
	evidenceJSON, err := json.Marshal(evidenceData)
	if err != nil {
		return err
	}

	evidence := sqlite.Evidence{ID: "evidence_" + uuid.NewString(), RunID: task.RunID, TaskID: task.ID, Kind: string(actions.EvidenceHTTPResponse), DataJSON: string(evidenceJSON), CreatedAt: time.Now()}
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
	return sink.Append(ctx, events.Event{RunID: task.RunID, EventType: events.EventEvidenceCreated, EntityKind: events.EntityEvidence, EntityID: evidence.ID, PayloadJSON: mustPayloadJSON(events.EvidenceCreatedPayload{Kind: evidence.Kind, Label: label}), CreatedAt: time.Now()})
}

func rootURL(input Input) (string, error) {
	if input.Host == "" {
		return "", fmt.Errorf("host is required")
	}
	scheme := input.Scheme
	if scheme == "" {
		scheme = "https"
	}
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("unsupported scheme %q", scheme)
	}
	host := input.Host
	if input.Port > 0 {
		if input.Port > 65535 {
			return "", fmt.Errorf("port out of range: %d", input.Port)
		}
		host = net.JoinHostPort(input.Host, strconv.Itoa(input.Port))
	}
	return (&url.URL{Scheme: scheme, Host: host, Path: "/"}).String(), nil
}

func headers(values http.Header) []model.HTTPHeader {
	headers := make([]model.HTTPHeader, 0, len(values))
	for name, vals := range values {
		headers = append(headers, model.HTTPHeader{Name: name, Values: vals})
	}
	return headers
}

func mustPayloadJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}
