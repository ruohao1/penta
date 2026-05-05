package scheduler

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/ruohao1/penta/internal/actions"
	crawlaction "github.com/ruohao1/penta/internal/actions/crawl"
	httprequest "github.com/ruohao1/penta/internal/actions/http_request"
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
	if evidence.Kind == string(actions.EvidenceHTTPResponse) {
		return deriveHTTPResponseCandidates(evidence)
	}
	if evidence.Kind == string(actions.EvidenceCrawl) {
		return deriveCrawlCandidates(evidence)
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
				Target:            targetRef(target),
			},
			probeCandidate,
		}, nil
	case targets.TypeIP, targets.TypeService, targets.TypeURL:
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
	requestURL := serviceRootURL(service)
	inputJSON, err := json.Marshal(httprequest.Input{Method: "GET", URL: requestURL, Depth: 0})
	if err != nil {
		return nil, err
	}
	return []CandidateTask{{ActionType: actions.ActionHTTPRequest, InputJSON: string(inputJSON), Reason: "HTTP service root can be requested", ParentEvidenceIDs: []string{evidence.ID}, Target: &model.TargetRef{Value: requestURL, Type: targets.TypeURL}, Depth: 0}}, nil
}

func deriveHTTPResponseCandidates(evidence sqlite.Evidence) ([]CandidateTask, error) {
	var response model.HTTPResponse
	if err := json.Unmarshal([]byte(evidence.DataJSON), &response); err != nil {
		return nil, fmt.Errorf("decode http response evidence %s: %w", evidence.ID, err)
	}
	if response.BodyArtifactID == "" || !strings.Contains(strings.ToLower(response.ContentType), "text/html") {
		return nil, nil
	}
	inputJSON, err := json.Marshal(crawlaction.Input(response))
	if err != nil {
		return nil, err
	}
	return []CandidateTask{{ActionType: actions.ActionCrawl, InputJSON: string(inputJSON), Reason: "HTML response can be crawled for links", ParentEvidenceIDs: []string{evidence.ID}, Target: &model.TargetRef{Value: response.URL, Type: targets.TypeURL}, Depth: response.Depth}}, nil
}

func deriveCrawlCandidates(evidence sqlite.Evidence) ([]CandidateTask, error) {
	var result model.CrawlResult
	if err := json.Unmarshal([]byte(evidence.DataJSON), &result); err != nil {
		return nil, fmt.Errorf("decode crawl evidence %s: %w", evidence.ID, err)
	}
	candidates := make([]CandidateTask, 0, len(result.URLs))
	for _, value := range result.URLs {
		depth := result.Depth + 1
		inputJSON, err := json.Marshal(httprequest.Input{Method: "GET", URL: value, Depth: depth})
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, CandidateTask{ActionType: actions.ActionHTTPRequest, InputJSON: string(inputJSON), Reason: "crawled URL can be requested", ParentEvidenceIDs: []string{evidence.ID}, Target: &model.TargetRef{Value: value, Type: targets.TypeURL}, CrawlDerived: true, Depth: depth})
	}
	return candidates, nil
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
		Target:            targetRef(target),
	}, nil
}

func targetRef(target model.TargetRef) *model.TargetRef {
	return &model.TargetRef{Value: target.Value, Type: target.Type}
}

func serviceTargetRef(service model.Service) *model.TargetRef {
	return &model.TargetRef{Value: serviceRootURL(service), Type: targets.TypeURL}
}

func serviceRootURL(service model.Service) string {
	host := service.Host
	if service.Port > 0 {
		host = net.JoinHostPort(service.Host, strconv.Itoa(service.Port))
	}
	return (&url.URL{Scheme: service.Scheme, Host: host, Path: "/"}).String()
}
