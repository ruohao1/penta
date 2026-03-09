package contentdiscovery

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/ruohao1/penta/internal/flow"
)

type RequestSpec struct {
	Method    string
	Headers   map[string]string
	Cookies   map[string]string
	UserAgent string
	Body      []byte
}

type DiscoverPayload struct {
	Endpoint string
	Depth    int

	// Filters from options for convenience, can be used in next stages.
}

type DiscoverResult struct {
	URL           string
	Endpoint      string
	Depth         int
	Path          string
	StatusCode    int
	ContentLength int64
	ContentType   string
	DurationMs    int64
	RedirectTo    string
	Error         string
}

type DiscoverStage struct {
	client  *http.Client
	workers int

	request  RequestSpec
	wordlist []string

	statusCodes  []int
	responseSize ResponseSize
	regexps      []*regexp.Regexp
}

var (
	maxDiscoverQueueItems = 2048
	maxDiscoverResults    = 20000
)

func NewDiscoverStage(opts Options) (*DiscoverStage, error) {
	compiledRegexps := make([]*regexp.Regexp, 0, len(opts.Regexps))
	for _, expr := range opts.Regexps {
		re, err := regexp.Compile(expr)
		if err != nil {
			return nil, fmt.Errorf("compile regexp %q: %w", expr, err)
		}
		compiledRegexps = append(compiledRegexps, re)
	}

	wordlist := make([]string, 0)
	// Load wordlist.
	f, err := os.Open(opts.Wordlist)
	if err != nil {
		return nil, fmt.Errorf("open wordlist: %w", err)
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	// Increase max token size in case lines are long.
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		wordlist = append(wordlist, line)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("scan wordlist: %w", err)
	}
	timeout := time.Duration(opts.Timeout) * time.Second

	tr := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: timeout,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
	}
	client := &http.Client{
		Timeout:   timeout,
		Transport: tr,
	}
	return &DiscoverStage{
		client:       client,
		workers:      opts.Workers,
		wordlist:     wordlist,
		request:      buildRequestSpec(opts),
		statusCodes:  opts.StatusCodes,
		responseSize: opts.ResponseSize,
		regexps:      compiledRegexps,
	}, nil
}

func (s *DiscoverStage) Name() string { return "content.discover" }

func (s *DiscoverStage) Workers() int { return s.workers }

func (s *DiscoverStage) Process(ctx context.Context, in flow.Item) ([]flow.Item, error) {
	payload, ok := in.Payload.(DiscoverPayload)
	if !ok {
		ptr, okPtr := in.Payload.(*DiscoverPayload)
		if !okPtr || ptr == nil {
			return nil, fmt.Errorf("content discover: invalid payload type %T", in.Payload)
		}
		payload = *ptr
	}

	results := make([]flow.Item, 0, len(s.wordlist))
	visited := map[string]struct{}{}
	type queueItem struct {
		endpoint string
		depth    int
	}
	queue := []queueItem{{endpoint: payload.Endpoint, depth: payload.Depth}}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		normCur, err := normalizeEndpoint(cur.endpoint)
		if err != nil {
			results = append(results, flow.Item{
				Feature: string(flow.ContentDiscovery),
				Stage:   s.Name(),
				Target:  cur.endpoint,
				Key:     cur.endpoint,
				Payload: DiscoverResult{
					URL:      cur.endpoint,
					Endpoint: cur.endpoint,
					Depth:    cur.depth,
					Error:    "normalize endpoint: " + err.Error(),
				},
			})
			continue
		}
		if _, seen := visited[normCur]; seen {
			continue
		}
		visited[normCur] = struct{}{}

		for _, line := range s.wordlist {
			if len(results) >= maxDiscoverResults {
				results = append(results, flow.Item{
					Feature: string(flow.ContentDiscovery),
					Stage:   s.Name(),
					Target:  cur.endpoint,
					Key:     cur.endpoint,
					Payload: DiscoverResult{
						URL:      cur.endpoint,
						Endpoint: cur.endpoint,
						Depth:    cur.depth,
						Error:    fmt.Sprintf("result limit reached (%d)", maxDiscoverResults),
					},
				})
				return results, nil
			}

			req, targetURL, err := buildRequest(ctx, cur.endpoint, line, s.request)
			if err != nil {
				results = append(results, flow.Item{
					Feature: string(flow.ContentDiscovery),
					Stage:   s.Name(),
					Target:  cur.endpoint,
					Key:     cur.endpoint,
					Payload: DiscoverResult{
						URL:      cur.endpoint,
						Endpoint: cur.endpoint,
						Depth:    cur.depth,
						Path:     line,
						Error:    fmt.Sprintf("build request for %q: %v", line, err),
					},
				})
				continue
			}

			start := time.Now()
			resp, err := s.client.Do(req)
			duration := time.Since(start).Milliseconds()
			if err != nil {
				results = append(results, flow.Item{
					Feature: string(flow.ContentDiscovery),
					Stage:   s.Name(),
					Target:  targetURL,
					Key:     targetURL,
					Payload: DiscoverResult{
						URL:      targetURL,
						Endpoint: cur.endpoint,
						Depth:    cur.depth,
						Path:     line,
						Error:    err.Error(),
					},
				})
				continue
			}

			body, readErr := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if readErr != nil {
				results = append(results, flow.Item{
					Feature: string(flow.ContentDiscovery),
					Stage:   s.Name(),
					Target:  targetURL,
					Key:     targetURL,
					Payload: DiscoverResult{
						URL:      targetURL,
						Endpoint: cur.endpoint,
						Depth:    cur.depth,
						Path:     line,
						Error:    "read response body: " + readErr.Error(),
					},
				})
				continue
			}

			if filter, err := s.filterResponse(resp, body, targetURL); err != nil {
				results = append(results, flow.Item{
					Feature: string(flow.ContentDiscovery),
					Stage:   s.Name(),
					Target:  targetURL,
					Key:     targetURL,
					Payload: DiscoverResult{
						URL:      targetURL,
						Endpoint: cur.endpoint,
						Depth:    cur.depth,
						Path:     line,
						Error:    "filter response: " + err.Error(),
					},
				})
				continue
			} else if filter {
				continue
			}

			contentLength := resp.ContentLength
			if contentLength < 0 {
				contentLength = int64(len(body))
			}

			discoverResult := DiscoverResult{
				URL:           targetURL,
				Endpoint:      cur.endpoint,
				Depth:         cur.depth,
				Path:          line,
				StatusCode:    resp.StatusCode,
				ContentLength: contentLength,
				ContentType:   resp.Header.Get("Content-Type"),
				DurationMs:    duration,
				RedirectTo:    resp.Header.Get("Location"),
			}

			results = append(results, flow.Item{
				Feature: string(flow.ContentDiscovery),
				Stage:   s.Name(),
				Target:  targetURL,
				Key:     targetURL,
				Payload: discoverResult,
			})

			if cur.depth > 0 {
				child, ok := nextEndpointFromResult(discoverResult)
				if !ok {
					continue
				}
				normChild, err := normalizeEndpoint(child)
				if err != nil {
					continue
				}
				if _, seen := visited[normChild]; !seen {
					if len(queue) >= maxDiscoverQueueItems {
						results = append(results, flow.Item{
							Feature: string(flow.ContentDiscovery),
							Stage:   s.Name(),
							Target:  cur.endpoint,
							Key:     cur.endpoint,
							Payload: DiscoverResult{
								URL:      cur.endpoint,
								Endpoint: cur.endpoint,
								Depth:    cur.depth,
								Error:    fmt.Sprintf("queue limit reached (%d)", maxDiscoverQueueItems),
							},
						})
						return results, nil
					}
					queue = append(queue, queueItem{endpoint: child, depth: cur.depth - 1})
				}
			}
		}
	}

	return results, nil
}

func nextEndpointFromResult(r DiscoverResult) (string, bool) {
	if r.Error != "" {
		return "", false
	}
	if r.StatusCode < 200 || r.StatusCode >= 400 {
		return "", false
	}
	u, err := url.Parse(r.URL)
	if err != nil {
		return "", false
	}
	if !strings.HasSuffix(u.Path, "/") {
		return "", false
	}
	return u.String(), true
}

func normalizeEndpoint(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("missing scheme or host")
	}
	if u.Path == "" {
		u.Path = "/"
	}
	u.Fragment = ""
	return u.String(), nil
}

func buildRequestSpec(opts Options) RequestSpec {
	return RequestSpec{
		Method:    strings.ToUpper(strings.TrimSpace(opts.Method)),
		Headers:   cloneMap(opts.Headers),
		Cookies:   cloneMap(opts.Cookies),
		UserAgent: strings.TrimSpace(opts.UserAgent),
		Body:      []byte(opts.Data),
	}
}

func buildRequest(
	ctx context.Context,
	baseEndpoint string,
	pathPayload string,
	spec RequestSpec,
) (*http.Request, string, error) {
	base, err := url.Parse(baseEndpoint)
	if err != nil {
		return nil, "", err
	}
	// normalize path from wordlist
	p := strings.TrimSpace(pathPayload)
	p = strings.TrimPrefix(p, "/")
	if base.Path == "" {
		base.Path = "/"
	}
	if !strings.HasSuffix(base.Path, "/") {
		base.Path += "/"
	}
	target := base.ResolveReference(&url.URL{Path: p})
	var body io.Reader
	if len(spec.Body) > 0 {
		body = bytes.NewReader(spec.Body) // new reader each request
	}
	method := strings.ToUpper(strings.TrimSpace(spec.Method))
	if method == "" {
		method = http.MethodGet
	}
	req, err := http.NewRequestWithContext(ctx, method, target.String(), body)
	if err != nil {
		return nil, "", err
	}
	for k, v := range spec.Headers {
		req.Header.Set(k, v)
	}
	if spec.UserAgent != "" {
		req.Header.Set("User-Agent", spec.UserAgent)
	}
	for k, v := range spec.Cookies {
		req.AddCookie(&http.Cookie{Name: k, Value: v})
	}
	return req, target.String(), nil
}

func cloneMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	c := make(map[string]string, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}

func (s DiscoverStage) filterResponse(resp *http.Response, body []byte, targetURL string) (bool, error) {
	filter := false
	if len(s.statusCodes) > 0 {
		match := false
		for _, code := range s.statusCodes {
			if resp.StatusCode == code {
				match = true
				break
			}
		}
		if !match {
			filter = true
		}
	}

	if !filter && (s.responseSize.Min > 0 || s.responseSize.Max > 0) {
		size := resp.ContentLength
		if size < 0 {
			size = int64(len(body))
		}
		if (s.responseSize.Min > 0 && size < int64(s.responseSize.Min)) ||
			(s.responseSize.Max > 0 && size > int64(s.responseSize.Max)) {
			filter = true
		}
	}

	if !filter && len(s.regexps) > 0 {
		matched := false
		search := targetURL
		if len(body) > 0 {
			search += "\n" + string(body)
		}
		for _, re := range s.regexps {
			if re.MatchString(search) {
				matched = true
				break
			}
		}
		if !matched {
			filter = true
		}
	}

	return filter, nil
}
