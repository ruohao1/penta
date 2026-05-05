package http_request

import "github.com/ruohao1/penta/internal/model"

type Input struct {
	Method  string             `json:"method"`
	URL     string             `json:"url"`
	Headers []model.HTTPHeader `json:"headers,omitempty"`
	Depth   int                `json:"depth,omitempty"`
}

type Evidence = model.HTTPResponse
