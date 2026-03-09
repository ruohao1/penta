package runtime

import (
	"context"
	"time"

	"github.com/ruohao1/penta/internal/events"
	"github.com/ruohao1/penta/internal/flow"
)

var findingStagePolicy = map[flow.Type][]string{
	flow.ContentDiscovery: {"content.discover"},
}

func findingStagesForFeature(feature flow.Type) []string {
	stages := findingStagePolicy[feature]
	if len(stages) == 0 {
		return nil
	}
	out := make([]string, len(stages))
	copy(out, stages)
	return out
}

type findingPipexSink struct {
	feature flow.Type
	stage   string
	out     Sink
}

func (s findingPipexSink) Name() string {
	return "finding:" + s.stage
}

func (s findingPipexSink) Stage() string {
	return s.stage
}

func (s findingPipexSink) Consume(ctx context.Context, item Item) error {
	if s.out == nil {
		return nil
	}
	data := payloadData(item.Payload)
	errStr := ""
	if v, ok := data["error"].(string); ok {
		errStr = v
	}
	stage := item.Stage
	if stage == "" {
		stage = s.stage
	}
	feature := item.Feature
	if feature == "" {
		feature = string(s.feature)
	}
	return s.out.Emit(ctx, events.Event{
		At:      time.Now(),
		Kind:    events.Finding,
		Feature: feature,
		Stage:   stage,
		Target:  item.Target,
		Err:     errStr,
		Data:    data,
	})
}
