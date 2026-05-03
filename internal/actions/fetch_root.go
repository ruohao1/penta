package actions

type FetchRootInput struct {
	Scheme string `json:"scheme,omitempty"`
	Host   string `json:"host"`
	Port   int    `json:"port,omitempty"`
}

type FetchRootEvidence struct {
	RootURL string `json:"root_url"`
}
