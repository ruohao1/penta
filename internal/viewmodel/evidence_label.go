package viewmodel

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ruohao1/penta/internal/model"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

func EvidenceLabel(evidence sqlite.Evidence) (string, error) {
	switch evidence.Kind {
	case "target":
		var target model.TargetRef
		if err := json.Unmarshal([]byte(evidence.DataJSON), &target); err != nil {
			return "", err
		}
		return fmt.Sprintf("%s %s", target.Type, target.Value), nil
	case "service":
		var service model.Service
		if err := json.Unmarshal([]byte(evidence.DataJSON), &service); err != nil {
			return "", err
		}
		return fmt.Sprintf("%s %s:%d", service.Scheme, service.Host, service.Port), nil
	case "dns_record":
		var payload struct {
			Records []model.DNSRecord `json:"records"`
		}
		if err := json.Unmarshal([]byte(evidence.DataJSON), &payload); err != nil {
			return "", err
		}
		parts := make([]string, 0, len(payload.Records))
		for _, record := range payload.Records {
			parts = append(parts, fmt.Sprintf("%s %s -> %s", record.Type, record.Name, record.Value))
		}
		if len(parts) == 0 {
			return "no records", nil
		}
		return strings.Join(parts, ", "), nil
	default:
		return evidence.Kind, nil
	}
}
