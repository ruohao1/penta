package runtime

import "testing"

type payloadSample struct {
	StatusCode    int
	DurationMs    int64
	ContentType   string
	Error         string
	privateIgnore string
}

func TestPayloadDataFromStruct(t *testing.T) {
	t.Parallel()

	data := payloadData(payloadSample{
		StatusCode:    200,
		DurationMs:    12,
		ContentType:   "text/plain",
		Error:         "",
		privateIgnore: "x",
	})
	if data == nil {
		t.Fatal("expected non-nil data")
	}
	if got := data["status_code"]; got != int64(200) {
		t.Fatalf("status_code = %v, want 200", got)
	}
	if got := data["duration_ms"]; got != int64(12) {
		t.Fatalf("duration_ms = %v, want 12", got)
	}
	if got := data["content_type"]; got != "text/plain" {
		t.Fatalf("content_type = %v, want text/plain", got)
	}
	if _, ok := data["private_ignore"]; ok {
		t.Fatal("did not expect private field in payload data")
	}
}

func TestPayloadDataFromPointerAndMap(t *testing.T) {
	t.Parallel()

	ptrData := payloadData(&payloadSample{StatusCode: 201, DurationMs: 3})
	if ptrData["status_code"] != int64(201) {
		t.Fatalf("status_code = %v, want 201", ptrData["status_code"])
	}

	mapData := payloadData(map[string]any{"url": "https://example.com", "depth": 1})
	if mapData["url"] != "https://example.com" {
		t.Fatalf("url = %v, want https://example.com", mapData["url"])
	}
	if mapData["depth"] != 1 {
		t.Fatalf("depth = %v, want 1", mapData["depth"])
	}
}

func TestPayloadDataNilAndCamelToSnake(t *testing.T) {
	t.Parallel()

	if got := payloadData(nil); got != nil {
		t.Fatalf("payloadData(nil) = %v, want nil", got)
	}
	if got := camelToSnake("StatusCode"); got != "status_code" {
		t.Fatalf("camelToSnake(StatusCode) = %q, want %q", got, "status_code")
	}
	if got := camelToSnake(""); got != "" {
		t.Fatalf("camelToSnake(empty) = %q, want empty", got)
	}
}
