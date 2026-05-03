package model

type Service struct {
	Scheme string `json:"scheme,omitempty"`
	Host   string `json:"host"`
	Port   int    `json:"port,omitempty"`
}
