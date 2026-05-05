package actions

import "github.com/ruohao1/penta/internal/model"

type CrawlInput = model.HTTPResponse

type CrawlEvidence struct {
	URLs []string `json:"urls"`
}
