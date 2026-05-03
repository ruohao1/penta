package scheduler

import (
	"encoding/json"
	"fmt"

	"github.com/ruohao1/penta/internal/actions"
	probehttp "github.com/ruohao1/penta/internal/actions/probe_http"
	"github.com/ruohao1/penta/internal/model"
	"github.com/ruohao1/penta/internal/storage/sqlite"
	"github.com/ruohao1/penta/internal/targets"
)

func DeriveFromEvidence(evidence sqlite.Evidence) ([]CandidateTask, error) {
	if evidence.Kind != string(actions.EvidenceTarget) {
		return nil, nil
	}

	var target model.TargetRef
	if err := json.Unmarshal([]byte(evidence.DataJSON), &target); err != nil {
		return nil, fmt.Errorf("decode target evidence %s: %w", evidence.ID, err)
	}

	switch target.Type {
	case targets.TypeDomain, targets.TypeIP, targets.TypeURL:
		inputJSON, err := json.Marshal(probehttp.Input{
			Value: target.Value,
			Type:  target.Type,
		})
		if err != nil {
			return nil, err
		}

		return []CandidateTask{
			{
				ActionType:        actions.ActionProbeHTTP,
				InputJSON:         string(inputJSON),
				Reason:            fmt.Sprintf("target %s can be probed for HTTP service", target.Type),
				ParentEvidenceIDs: []string{evidence.ID},
			},
		}, nil
	default:
		return nil, nil
	}
}
