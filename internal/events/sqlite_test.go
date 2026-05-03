package events

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

func openTestSink(t *testing.T) *SQLiteSink {
	t.Helper()

	db, err := sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "penta.db"))
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return &SQLiteSink{DB: db}
}

func createRun(t *testing.T, sink *SQLiteSink, runID string) {
	t.Helper()

	err := sink.DB.CreateRun(context.Background(), sqlite.Run{
		ID:        runID,
		Mode:      "recon",
		Status:    actions.RunStatusRunning,
		CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("create run %s: %v", runID, err)
	}
}

func TestSQLiteSinkAppendAssignsPerRunSequence(t *testing.T) {
	sink := openTestSink(t)
	ctx := context.Background()
	createRun(t, sink, "run_1")
	createRun(t, sink, "run_2")

	first := Event{RunID: "run_1", EventType: EventRunCreated, EntityKind: EntityRun, EntityID: "run_1", PayloadJSON: `{}`, CreatedAt: time.Now()}
	if err := sink.Append(ctx, first); err != nil {
		t.Fatalf("append first event: %v", err)
	}
	second := Event{RunID: "run_1", EventType: EventRunCompleted, EntityKind: EntityRun, EntityID: "run_1", PayloadJSON: `{}`, CreatedAt: time.Now()}
	if err := sink.Append(ctx, second); err != nil {
		t.Fatalf("append second event: %v", err)
	}
	third := Event{RunID: "run_2", EventType: EventRunCreated, EntityKind: EntityRun, EntityID: "run_2", PayloadJSON: `{}`, CreatedAt: time.Now()}
	if err := sink.Append(ctx, third); err != nil {
		t.Fatalf("append third event: %v", err)
	}

	runOneEvents, err := sink.ListByRunSinceSeq(ctx, "run_1", 0, 10)
	if err != nil {
		t.Fatalf("list run_1 events: %v", err)
	}
	if len(runOneEvents) != 2 {
		t.Fatalf("unexpected run_1 event count: got %d want 2", len(runOneEvents))
	}
	if runOneEvents[0].Seq != 1 || runOneEvents[1].Seq != 2 {
		t.Fatalf("unexpected run_1 seq values: %+v", runOneEvents)
	}

	runTwoEvents, err := sink.ListByRunSinceSeq(ctx, "run_2", 0, 10)
	if err != nil {
		t.Fatalf("list run_2 events: %v", err)
	}
	if len(runTwoEvents) != 1 || runTwoEvents[0].Seq != 1 {
		t.Fatalf("unexpected run_2 events: %+v", runTwoEvents)
	}
}

func TestSQLiteSinkListByRunSinceSeq(t *testing.T) {
	sink := openTestSink(t)
	ctx := context.Background()
	createRun(t, sink, "run_1")

	for i, eventType := range []EventType{EventRunCreated, EventTaskEnqueued, EventTaskStarted} {
		if err := sink.Append(ctx, Event{RunID: "run_1", EventType: eventType, EntityKind: EntityRun, EntityID: "entity", PayloadJSON: `{}`, CreatedAt: time.Now().Add(time.Duration(i) * time.Second)}); err != nil {
			t.Fatalf("append event %d: %v", i, err)
		}
	}

	events, err := sink.ListByRunSinceSeq(ctx, "run_1", 1, 10)
	if err != nil {
		t.Fatalf("list events since seq: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("unexpected filtered event count: got %d want 2", len(events))
	}
	if events[0].Seq != 2 || events[1].Seq != 3 {
		t.Fatalf("unexpected filtered seqs: %+v", events)
	}
}
