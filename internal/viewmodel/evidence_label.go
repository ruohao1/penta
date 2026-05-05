package viewmodel

import (
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/ruohao1/penta/internal/model"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

func EvidenceLabel(evidence sqlite.Evidence) (string, error) {
	summary, err := EvidenceSummaryFor(evidence)
	if err != nil {
		return "", err
	}
	return summary.Label, nil
}

func EvidenceSummaryFor(evidence sqlite.Evidence) (EvidenceSummary, error) {
	summary := EvidenceSummary{ID: evidence.ID, Kind: evidence.Kind}
	switch evidence.Kind {
	case "target":
		var target model.TargetRef
		if err := json.Unmarshal([]byte(evidence.DataJSON), &target); err != nil {
			return EvidenceSummary{}, err
		}
		summary.Label = fmt.Sprintf("%s %s", target.Type, target.Value)
		return summary, nil
	case "service":
		var service model.Service
		if err := json.Unmarshal([]byte(evidence.DataJSON), &service); err != nil {
			return EvidenceSummary{}, err
		}
		summary.URL = serviceURL(service)
		if summary.URL != "" {
			summary.Label = summary.URL
		} else {
			summary.Label = fmt.Sprintf("%s %s:%d", service.Scheme, service.Host, service.Port)
		}
		return summary, nil
	case "dns_record":
		var payload struct {
			Records []model.DNSRecord `json:"records"`
		}
		if err := json.Unmarshal([]byte(evidence.DataJSON), &payload); err != nil {
			return EvidenceSummary{}, err
		}
		details := make([]string, 0, len(payload.Records))
		names := map[string]bool{}
		for _, record := range payload.Records {
			details = append(details, fmt.Sprintf("%s %s -> %s", record.Type, record.Name, record.Value))
			if record.Name != "" {
				names[record.Name] = true
			}
		}
		if len(details) == 0 {
			summary.Label = "no records"
			return summary, nil
		}
		summary.Label = strings.Join(sortedKeys(names), ", ")
		summary.Details = details
		return summary, nil
	case "http_response":
		var response model.HTTPResponse
		if err := json.Unmarshal([]byte(evidence.DataJSON), &response); err != nil {
			return EvidenceSummary{}, err
		}
		summary.URL = response.URL
		if response.StatusCode > 0 {
			summary.Label = fmt.Sprintf("%s %d", response.URL, response.StatusCode)
		} else {
			summary.Label = response.URL
		}
		if response.ContentType != "" {
			summary.Details = append(summary.Details, "content-type: "+response.ContentType)
		}
		if response.ContentLength > 0 {
			summary.Details = append(summary.Details, fmt.Sprintf("content-length: %d bytes", response.ContentLength))
		}
		if response.BodyBytes > 0 {
			bodyDetail := fmt.Sprintf("body: %d bytes", response.BodyBytes)
			if response.BodyTruncated {
				bodyDetail += fmt.Sprintf(" (truncated at %d bytes)", response.BodyReadLimitBytes)
			}
			summary.Details = append(summary.Details, bodyDetail)
		} else if response.BodyReadLimitBytes > 0 && response.BodyTruncated {
			summary.Details = append(summary.Details, fmt.Sprintf("body: truncated at %d bytes", response.BodyReadLimitBytes))
		}
		if response.HeadersTruncated {
			summary.Details = append(summary.Details, "headers: truncated")
		}
		if response.BodySHA256 != "" {
			summary.Details = append(summary.Details, "sha256: "+response.BodySHA256)
		}
		if response.BodyArtifactID != "" {
			summary.Details = append(summary.Details, "body artifact: "+response.BodyArtifactID)
		}
		return summary, nil
	case "crawl":
		var result model.CrawlResult
		if err := json.Unmarshal([]byte(evidence.DataJSON), &result); err != nil {
			return EvidenceSummary{}, err
		}
		summary.URL = result.SourceURL
		count := len(result.URLs)
		if count == 1 {
			summary.Label = "1 url from " + result.SourceURL
		} else {
			summary.Label = fmt.Sprintf("%d urls from %s", count, result.SourceURL)
		}
		for _, value := range result.URLs {
			summary.Details = append(summary.Details, value)
		}
		return summary, nil
	default:
		summary.Label = evidence.Kind
		return summary, nil
	}
}

func serviceURL(service model.Service) string {
	if service.Scheme == "" || service.Host == "" {
		return ""
	}
	host := service.Host
	if strings.Contains(host, ":") {
		host = "[" + strings.Trim(host, "[]") + "]"
	}
	if service.Port == 0 || (service.Scheme == "https" && service.Port == 443) || (service.Scheme == "http" && service.Port == 80) {
		return service.Scheme + "://" + host
	}
	return service.Scheme + "://" + net.JoinHostPort(strings.Trim(host, "[]"), fmt.Sprintf("%d", service.Port))
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
