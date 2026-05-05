package model

type HTTPHeader struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

type HTTPResponse struct {
	URL                string       `json:"url"`
	StatusCode         int          `json:"status_code"`
	Headers            []HTTPHeader `json:"headers,omitempty"`
	HeadersTruncated   bool         `json:"headers_truncated,omitempty"`
	ContentType        string       `json:"content_type,omitempty"`
	ContentLength      int64        `json:"content_length,omitempty"`
	BodyBytes          int64        `json:"body_bytes,omitempty"`
	BodyReadLimitBytes int64        `json:"body_read_limit_bytes,omitempty"`
	BodyTruncated      bool         `json:"body_truncated,omitempty"`
	BodySHA256         string       `json:"body_sha256,omitempty"`
	BodyArtifactID     string       `json:"body_artifact_id,omitempty"`
}
