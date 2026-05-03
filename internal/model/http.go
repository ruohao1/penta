package model

type HTTPHeader struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

type HTTPResponse struct {
	URL            string       `json:"url"`
	StatusCode     int          `json:"status_code"`
	Headers        []HTTPHeader `json:"headers,omitempty"`
	ContentType    string       `json:"content_type,omitempty"`
	BodyBytes      int64        `json:"body_bytes,omitempty"`
	BodySHA256     string       `json:"body_sha256,omitempty"`
	BodyArtifactID string       `json:"body_artifact_id,omitempty"`
}
