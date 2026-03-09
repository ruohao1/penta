package runtime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/ruohao1/penta/internal/events"
	"github.com/ruohao1/penta/internal/features/contentdiscovery"
	"github.com/ruohao1/penta/internal/flow"
	"github.com/ruohao1/penta/internal/sinks"
	"github.com/ruohao1/pipex"
)

type testStage struct {
	name    string
	workers int
	run     func(context.Context, Item) ([]Item, error)
}

func (s testStage) Name() string { return s.name }
func (s testStage) Workers() int { return s.workers }
func (s testStage) Process(ctx context.Context, in Item) ([]Item, error) {
	return s.run(ctx, in)
}

type captureSink struct {
	mu     sync.Mutex
	events []events.Event
}

func (s *captureSink) Emit(ctx context.Context, ev events.Event) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, ev)
	return nil
}

func (s *captureSink) Close() error { return nil }

func (s *captureSink) snapshot() []events.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]events.Event, len(s.events))
	copy(out, s.events)
	return out
}

func TestRuntimeRunEmitsFindingsFromPolicyStage(t *testing.T) {
	t.Parallel()

	seed := testStage{name: "seed", workers: 1, run: func(ctx context.Context, in Item) ([]Item, error) {
		_ = ctx
		return []Item{{Feature: in.Feature, Stage: "seed", Target: in.Target, Payload: map[string]any{"seed": true}}}, nil
	}}
	discover := testStage{name: "content.discover", workers: 1, run: func(ctx context.Context, in Item) ([]Item, error) {
		_ = ctx
		_ = in
		return []Item{{Feature: string(flow.ContentDiscovery), Stage: "content.discover", Target: "https://example.com/a", Payload: struct {
			URL    string
			Error  string
			Status int
		}{URL: "https://example.com/a", Error: "boom", Status: 500}}}, nil
	}}
	other := testStage{name: "other.stage", workers: 1, run: func(ctx context.Context, in Item) ([]Item, error) {
		_ = ctx
		_ = in
		return []Item{{Feature: string(flow.ContentDiscovery), Stage: "other.stage", Target: "https://example.com/other", Payload: map[string]any{"url": "https://example.com/other"}}}, nil
	}}

	plan := flow.Plan{
		Feature: flow.ContentDiscovery,
		Stages:  []pipex.Stage[flow.Item]{seed, discover, other},
		Edges: []flow.Edge{
			{From: seed.Name(), To: discover.Name()},
			{From: seed.Name(), To: other.Name()},
		},
		Seeds: map[string][]flow.Item{seed.Name(): []flow.Item{{Feature: string(flow.ContentDiscovery), Stage: seed.Name(), Target: "seed"}}},
	}

	cap := &captureSink{}
	rt := New(DefaultConfig(), cap)
	if err := rt.Run(context.Background(), plan, cap); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	evs := cap.snapshot()
	if len(evs) == 0 {
		t.Fatal("expected events")
	}

	firstFinding := -1
	runFinished := -1
	for i, ev := range evs {
		if ev.Kind == events.Finding {
			if firstFinding == -1 {
				firstFinding = i
			}
			if ev.Stage != "content.discover" {
				t.Fatalf("finding stage = %q, want %q", ev.Stage, "content.discover")
			}
			if ev.Err != "boom" {
				t.Fatalf("finding err = %q, want %q", ev.Err, "boom")
			}
		}
		if ev.Kind == events.RunFinished {
			runFinished = i
		}
	}
	if firstFinding == -1 {
		t.Fatal("expected at least one finding event")
	}
	if runFinished == -1 {
		t.Fatal("expected run finished event")
	}
	if firstFinding > runFinished {
		t.Fatalf("first finding index %d should be before run finished index %d", firstFinding, runFinished)
	}
}

func TestRuntimeContentDiscoveryFindingsCarryFields(t *testing.T) {
	t.Parallel()

	wordlist := writeTempWordlist(t, []string{"admin"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("admin endpoint"))
	}))
	defer server.Close()

	feat := contentdiscovery.New()
	plan, err := feat.Build(context.Background(), flow.BuildInput{Task: contentdiscovery.Options{
		Targets:  []string{server.URL},
		Method:   http.MethodGet,
		Wordlist: wordlist,
		Workers:  1,
		Timeout:  5,
		Regexps:  []string{"admin"},
	}})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	cap := &captureSink{}
	rt := New(DefaultConfig(), sinks.NewMultiSink(cap))
	if err := rt.Run(context.Background(), plan, cap); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	evs := cap.snapshot()
	var finding *events.Event
	for i := range evs {
		if evs[i].Kind == events.Finding {
			finding = &evs[i]
			break
		}
	}
	if finding == nil {
		t.Fatal("expected a finding event")
	}
	if finding.Data == nil {
		t.Fatal("expected finding data")
	}
	for _, key := range []string{"status_code", "url", "depth", "content_length", "duration_ms"} {
		if _, ok := finding.Data[key]; !ok {
			t.Fatalf("missing finding data key %q", key)
		}
	}
}

func writeTempWordlist(t *testing.T, lines []string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "wordlist.txt")
	content := ""
	for _, line := range lines {
		content += line + "\n"
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	return p
}
