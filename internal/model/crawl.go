package model

type CrawlResult struct {
	SourceURL string   `json:"source_url"`
	URLs      []string `json:"urls"`
}
