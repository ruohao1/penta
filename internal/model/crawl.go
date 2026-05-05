package model

type CrawlResult struct {
	SourceURL string   `json:"source_url"`
	Depth     int      `json:"depth,omitempty"`
	URLs      []string `json:"urls"`
}
