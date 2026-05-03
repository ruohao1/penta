package resolve_dns

import "github.com/ruohao1/penta/internal/model"

type Input struct {
	Domain string `json:"domain"`
}

type Evidence struct {
	Records []model.DNSRecord `json:"records"`
}
