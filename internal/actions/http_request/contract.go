package http_request

import "github.com/ruohao1/penta/internal/model"

type Input struct {
	Method  string             `json:"method"`
	URL     string             `json:"url"`
	Headers []model.HTTPHeader `json:"headers,omitempty"`
}

type Evidence = model.HTTPResponse
