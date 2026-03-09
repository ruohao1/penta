package contentdiscovery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/ruohao1/penta/internal/flow"
)

func TestDiscoverStageDepthRecursion(t *testing.T) {
	t.Parallel()

	wordlistPath := writeWordlist(t, []string{"dir/"})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	stage, err := NewDiscoverStage(Options{
		Wordlist: wordlistPath,
		Workers:  1,
		Timeout:  5,
		Method:   http.MethodGet,
	})
	if err != nil {
		t.Fatalf("NewDiscoverStage() error = %v", err)
	}

	t.Run("depth zero scans once", func(t *testing.T) {
		out, err := stage.Process(context.Background(), flow.Item{
			Payload: DiscoverPayload{Endpoint: ts.URL + "/", Depth: 0},
		})
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}
		if len(out) != 1 {
			t.Fatalf("len(out) = %d, want 1", len(out))
		}
		res, ok := out[0].Payload.(DiscoverResult)
		if !ok {
			t.Fatalf("payload type = %T, want DiscoverResult", out[0].Payload)
		}
		if !strings.HasSuffix(res.URL, "/dir/") {
			t.Fatalf("result URL = %q, want suffix /dir/", res.URL)
		}
	})

	t.Run("depth one recurses one level", func(t *testing.T) {
		out, err := stage.Process(context.Background(), flow.Item{
			Payload: DiscoverPayload{Endpoint: ts.URL + "/", Depth: 1},
		})
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}
		if len(out) != 2 {
			t.Fatalf("len(out) = %d, want 2", len(out))
		}

		urls := []string{}
		for _, item := range out {
			res, ok := item.Payload.(DiscoverResult)
			if !ok {
				t.Fatalf("payload type = %T, want DiscoverResult", item.Payload)
			}
			urls = append(urls, res.URL)
		}
		if !containsSuffix(urls, "/dir/") {
			t.Fatalf("missing first level URL, got %v", urls)
		}
		if !containsSuffix(urls, "/dir/dir/") {
			t.Fatalf("missing second level URL, got %v", urls)
		}
	})
}

func TestNormalizeEndpoint(t *testing.T) {
	t.Parallel()

	a, err := normalizeEndpoint("https://example.com#frag")
	if err != nil {
		t.Fatalf("normalizeEndpoint() error = %v", err)
	}
	b, err := normalizeEndpoint("https://example.com/")
	if err != nil {
		t.Fatalf("normalizeEndpoint() error = %v", err)
	}
	if a != b {
		t.Fatalf("normalized root mismatch: %q != %q", a, b)
	}

	c, err := normalizeEndpoint(" https://example.com/a#x ")
	if err != nil {
		t.Fatalf("normalizeEndpoint() error = %v", err)
	}
	if c != "https://example.com/a" {
		t.Fatalf("normalized path = %q, want %q", c, "https://example.com/a")
	}

	if _, err := normalizeEndpoint("/only/path"); err == nil {
		t.Fatal("expected error for endpoint without scheme/host")
	}
}

func TestBuildRequestPreservesBasePath(t *testing.T) {
	t.Parallel()

	req, targetURL, err := buildRequest(
		context.Background(),
		"https://example.com/app/",
		"admin",
		RequestSpec{Method: http.MethodGet},
	)
	if err != nil {
		t.Fatalf("buildRequest() error = %v", err)
	}
	if req == nil {
		t.Fatal("buildRequest() returned nil request")
	}
	if targetURL != "https://example.com/app/admin" {
		t.Fatalf("targetURL = %q, want %q", targetURL, "https://example.com/app/admin")
	}
	if req.URL.String() != targetURL {
		t.Fatalf("req.URL = %q, want %q", req.URL.String(), targetURL)
	}
}

func TestFilterResponseRegexInclude(t *testing.T) {
	t.Parallel()

	stage := DiscoverStage{
		regexps: []*regexp.Regexp{regexp.MustCompile("admin")},
	}
	resp := &http.Response{StatusCode: http.StatusOK, ContentLength: -1}

	t.Run("match keeps result", func(t *testing.T) {
		filtered, err := stage.filterResponse(resp, []byte("found /admin panel"), "https://example.com/")
		if err != nil {
			t.Fatalf("filterResponse() error = %v", err)
		}
		if filtered {
			t.Fatal("expected response to be kept when regex matches")
		}
	})

	t.Run("no match filters result", func(t *testing.T) {
		filtered, err := stage.filterResponse(resp, []byte("public homepage"), "https://example.com/")
		if err != nil {
			t.Fatalf("filterResponse() error = %v", err)
		}
		if !filtered {
			t.Fatal("expected response to be filtered when regex does not match")
		}
	})

	t.Run("url match keeps result", func(t *testing.T) {
		filtered, err := stage.filterResponse(resp, nil, "https://example.com/admin")
		if err != nil {
			t.Fatalf("filterResponse() error = %v", err)
		}
		if filtered {
			t.Fatal("expected response to be kept when regex matches URL")
		}
	})
}

func TestNewDiscoverStageRejectsInvalidRegex(t *testing.T) {
	t.Parallel()

	wordlistPath := writeWordlist(t, []string{"admin"})
	_, err := NewDiscoverStage(Options{
		Wordlist: wordlistPath,
		Workers:  1,
		Timeout:  5,
		Method:   http.MethodGet,
		Regexps:  []string{"("},
	})
	if err == nil {
		t.Fatal("expected error for invalid regexp")
	}
}

func TestFilterResponseStatusCodes(t *testing.T) {
	t.Parallel()

	stage := DiscoverStage{statusCodes: []int{http.StatusOK}}
	respOK := &http.Response{StatusCode: http.StatusOK, ContentLength: 0}
	respNotFound := &http.Response{StatusCode: http.StatusNotFound, ContentLength: 0}

	filtered, err := stage.filterResponse(respOK, nil, "https://example.com/")
	if err != nil {
		t.Fatalf("filterResponse() error = %v", err)
	}
	if filtered {
		t.Fatal("expected status 200 to pass filter")
	}

	filtered, err = stage.filterResponse(respNotFound, nil, "https://example.com/")
	if err != nil {
		t.Fatalf("filterResponse() error = %v", err)
	}
	if !filtered {
		t.Fatal("expected status 404 to be filtered")
	}
}

func TestFilterResponseSizeBounds(t *testing.T) {
	t.Parallel()

	stage := DiscoverStage{responseSize: ResponseSize{Min: 5, Max: 10}}
	resp := &http.Response{StatusCode: http.StatusOK, ContentLength: -1}

	filtered, err := stage.filterResponse(resp, []byte("1234"), "https://example.com/")
	if err != nil {
		t.Fatalf("filterResponse() error = %v", err)
	}
	if !filtered {
		t.Fatal("expected short body to be filtered")
	}

	filtered, err = stage.filterResponse(resp, []byte("12345"), "https://example.com/")
	if err != nil {
		t.Fatalf("filterResponse() error = %v", err)
	}
	if filtered {
		t.Fatal("expected in-range body to pass")
	}

	filtered, err = stage.filterResponse(resp, []byte("12345678901"), "https://example.com/")
	if err != nil {
		t.Fatalf("filterResponse() error = %v", err)
	}
	if !filtered {
		t.Fatal("expected long body to be filtered")
	}
}

func TestFilterResponseCombined(t *testing.T) {
	t.Parallel()

	stage := DiscoverStage{
		statusCodes:  []int{http.StatusOK},
		responseSize: ResponseSize{Min: 3, Max: 20},
		regexps:      []*regexp.Regexp{regexp.MustCompile("admin")},
	}
	resp := &http.Response{StatusCode: http.StatusOK, ContentLength: -1}

	filtered, err := stage.filterResponse(resp, []byte("hello world"), "https://example.com/foo")
	if err != nil {
		t.Fatalf("filterResponse() error = %v", err)
	}
	if !filtered {
		t.Fatal("expected non-matching regex content to be filtered")
	}

	filtered, err = stage.filterResponse(resp, []byte("admin panel"), "https://example.com/foo")
	if err != nil {
		t.Fatalf("filterResponse() error = %v", err)
	}
	if filtered {
		t.Fatal("expected response to pass when all filters match")
	}
}

func TestProcessErrorResultCarriesContext(t *testing.T) {
	t.Parallel()

	stage := &DiscoverStage{
		workers:  1,
		wordlist: []string{"admin"},
		request:  RequestSpec{Method: "BAD METHOD"},
	}

	out, err := stage.Process(context.Background(), flow.Item{
		Payload: DiscoverPayload{Endpoint: "https://example.com/", Depth: 2},
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if len(out) == 0 {
		t.Fatal("expected at least one error result")
	}

	res, ok := out[0].Payload.(DiscoverResult)
	if !ok {
		t.Fatalf("payload type = %T, want DiscoverResult", out[0].Payload)
	}
	if res.Error == "" {
		t.Fatal("expected non-empty error")
	}
	if res.Endpoint != "https://example.com/" {
		t.Fatalf("Endpoint = %q, want %q", res.Endpoint, "https://example.com/")
	}
	if res.Depth != 2 {
		t.Fatalf("Depth = %d, want %d", res.Depth, 2)
	}
	if res.Path != "admin" {
		t.Fatalf("Path = %q, want %q", res.Path, "admin")
	}
}

func TestProcessResultLimitGuardrail(t *testing.T) {
	t.Parallel()

	old := maxDiscoverResults
	maxDiscoverResults = 2
	defer func() { maxDiscoverResults = old }()

	wordlistPath := writeWordlist(t, []string{"a", "b", "c"})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	stage, err := NewDiscoverStage(Options{Wordlist: wordlistPath, Workers: 1, Timeout: 5, Method: http.MethodGet})
	if err != nil {
		t.Fatalf("NewDiscoverStage() error = %v", err)
	}

	out, err := stage.Process(context.Background(), flow.Item{Payload: DiscoverPayload{Endpoint: ts.URL + "/", Depth: 0}})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("len(out) = %d, want 3", len(out))
	}
	last, ok := out[len(out)-1].Payload.(DiscoverResult)
	if !ok {
		t.Fatalf("payload type = %T, want DiscoverResult", out[len(out)-1].Payload)
	}
	if !strings.Contains(last.Error, "result limit reached") {
		t.Fatalf("last error = %q, want result limit reached", last.Error)
	}
}

func TestProcessQueueLimitGuardrail(t *testing.T) {
	t.Parallel()

	old := maxDiscoverQueueItems
	maxDiscoverQueueItems = 1
	defer func() { maxDiscoverQueueItems = old }()

	wordlistPath := writeWordlist(t, []string{"a/", "b/"})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	stage, err := NewDiscoverStage(Options{Wordlist: wordlistPath, Workers: 1, Timeout: 5, Method: http.MethodGet})
	if err != nil {
		t.Fatalf("NewDiscoverStage() error = %v", err)
	}

	out, err := stage.Process(context.Background(), flow.Item{Payload: DiscoverPayload{Endpoint: ts.URL + "/", Depth: 1}})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if len(out) == 0 {
		t.Fatal("expected results")
	}
	found := false
	for _, item := range out {
		res, ok := item.Payload.(DiscoverResult)
		if !ok {
			t.Fatalf("payload type = %T, want DiscoverResult", item.Payload)
		}
		if strings.Contains(res.Error, "queue limit reached") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected queue limit reached error result")
	}
}

func writeWordlist(t *testing.T, lines []string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "wordlist.txt")
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	return path
}

func containsSuffix(values []string, suffix string) bool {
	for _, v := range values {
		if strings.HasSuffix(v, suffix) {
			return true
		}
	}
	return false
}
