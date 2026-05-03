package actions

type CrawlInput = FetchRootEvidence

type CrawlEvidence struct {
	URLs []string `json:"urls"`
}
