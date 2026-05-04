package scheduler

import (
	"encoding/json"
	"fmt"

	"github.com/ruohao1/penta/internal/actions"
	fetchroot "github.com/ruohao1/penta/internal/actions/fetch_root"
	probehttp "github.com/ruohao1/penta/internal/actions/probe_http"
	resolvedns "github.com/ruohao1/penta/internal/actions/resolve_dns"
	"github.com/ruohao1/penta/internal/model"
	"github.com/ruohao1/penta/internal/storage/sqlite"
	"github.com/ruohao1/penta/internal/targets"
)

func DeriveFromEvidence(evidence sqlite.Evidence) ([]CandidateTask, error) {
	if evidence.Kind == string(actions.EvidenceService) {
		return deriveServiceCandidates(evidence)
	}
	if evidence.Kind != string(actions.EvidenceTarget) {
		return nil, nil
	}

	var target model.TargetRef
	if err := json.Unmarshal([]byte(evidence.DataJSON), &target); err != nil {
		return nil, fmt.Errorf("decode target evidence %s: %w", evidence.ID, err)
	}

	switch target.Type {
	case targets.TypeDomain:
		probeCandidate, err := newProbeHTTPCandidate(evidence.ID, target)
		if err != nil {
			return nil, err
		}
		dnsInputJSON, err := json.Marshal(resolvedns.Input{Domain: target.Value})
		if err != nil {
			return nil, err
		}
		return []CandidateTask{
			{
				ActionType:        actions.ActionResolveDNS,
				InputJSON:         string(dnsInputJSON),
				Reason:            "domain target can be resolved with DNS",
				ParentEvidenceIDs: []string{evidence.ID},
			},
			probeCandidate,
		}, nil
	case targets.TypeIP, targets.TypeURL:
		candidate, err := newProbeHTTPCandidate(evidence.ID, target)
		if err != nil {
			return nil, err
		}
		return []CandidateTask{candidate}, nil
	default:
		return nil, nil
	}
}

func deriveServiceCandidates(evidence sqlite.Evidence) ([]CandidateTask, error) {
	var service model.Service
	if err := json.Unmarshal([]byte(evidence.DataJSON), &service); err != nil {
		return nil, fmt.Errorf("decode service evidence %s: %w", evidence.ID, err)
	}
	if service.Scheme != "http" && service.Scheme != "https" {
		return nil, nil
	}
	inputJSON, err := json.Marshal(fetchroot.Input(service))
	if err != nil {
		return nil, err
	}
	return []CandidateTask{{ActionType: actions.ActionFetchRoot, InputJSON: string(inputJSON), Reason: "HTTP service root can be fetched", ParentEvidenceIDs: []string{evidence.ID}}}, nil
}

func newProbeHTTPCandidate(evidenceID string, target model.TargetRef) (CandidateTask, error) {
	inputJSON, err := json.Marshal(probehttp.Input{
		Value: target.Value,
		Type:  target.Type,
	})
	if err != nil {
		return CandidateTask{}, err
	}
	return CandidateTask{
		ActionType:        actions.ActionProbeHTTP,
		InputJSON:         string(inputJSON),
		Reason:            fmt.Sprintf("target %s can be probed for HTTP service", target.Type),
		ParentEvidenceIDs: []string{evidenceID},
	}, nil
}
