package http_request

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/events"
	"github.com/ruohao1/penta/internal/model"
	"github.com/ruohao1/penta/internal/storage/sqlite"
	"github.com/ruohao1/penta/internal/viewmodel"
)

const (
	maxBodyBytes        = 1 << 20
	maxHeaderCount      = 64
	maxHeaderValues     = 16
	maxHeaderValueBytes = 1024
	redactedHeaderValue = "[REDACTED]"
)

type ipResolver func(context.Context, string) ([]netip.Addr, error)

var resolver ipResolver = defaultResolver

func Execute(ctx context.Context, db *sqlite.DB, sink events.Sink, task *sqlite.Task) error {
	var input Input
	if err := json.Unmarshal([]byte(task.InputJSON), &input); err != nil {
		return err
	}
	method, err := requestMethod(input.Method)
	if err != nil {
		return err
	}
	requestURL, err := validateURL(input.URL)
	if err != nil {
		return err
	}
	run, err := db.GetRun(ctx, task.RunID)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, nil)
	if err != nil {
		return err
	}
	for _, header := range input.Headers {
		for _, value := range header.Values {
			req.Header.Add(header.Name, value)
		}
	}
	resp, err := newHTTPClient(db, run).Do(req)
	if err != nil {
		return fmt.Errorf("http request %s %s: %w", method, requestURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes+1))
	if err != nil {
		return fmt.Errorf("read http response %s: %w", requestURL, err)
	}
	bodyBytes := int64(len(body))
	bodyTruncated := bodyBytes > maxBodyBytes
	if bodyTruncated {
		body = body[:maxBodyBytes]
		bodyBytes = maxBodyBytes
	}
	responseHeaders, headersTruncated := headers(resp.Header)
	bodyArtifactID, err := createHTMLBodyArtifact(ctx, db, task.ID, resp.Header.Get("Content-Type"), body)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(body)
	evidenceData := Evidence{
		URL:                requestURL,
		StatusCode:         resp.StatusCode,
		Headers:            responseHeaders,
		HeadersTruncated:   headersTruncated,
		ContentType:        resp.Header.Get("Content-Type"),
		ContentLength:      positiveContentLength(resp.ContentLength),
		BodyBytes:          bodyBytes,
		BodyReadLimitBytes: maxBodyBytes,
		BodyTruncated:      bodyTruncated,
		BodySHA256:         hex.EncodeToString(sum[:]),
		BodyArtifactID:     bodyArtifactID,
	}
	evidenceJSON, err := json.Marshal(evidenceData)
	if err != nil {
		return err
	}

	evidence := sqlite.Evidence{ID: "evidence_" + uuid.NewString(), RunID: task.RunID, TaskID: task.ID, Kind: string(actions.EvidenceHTTPResponse), DataJSON: string(evidenceJSON), CreatedAt: time.Now()}
	label, err := viewmodel.EvidenceLabel(evidence)
	if err != nil {
		return err
	}
	if err := db.CreateEvidence(ctx, evidence); err != nil {
		return err
	}
	if sink == nil {
		return nil
	}
	return sink.Append(ctx, events.Event{RunID: task.RunID, EventType: events.EventEvidenceCreated, EntityKind: events.EntityEvidence, EntityID: evidence.ID, PayloadJSON: mustPayloadJSON(events.EvidenceCreatedPayload{Kind: evidence.Kind, Label: label}), CreatedAt: time.Now()})
}

func requestMethod(method string) (string, error) {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = http.MethodGet
	}
	switch method {
	case http.MethodGet, http.MethodHead:
		return method, nil
	default:
		return "", fmt.Errorf("unsupported HTTP method %q", method)
	}
}

func validateURL(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme %q", parsed.Scheme)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("URL host is required")
	}
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	return parsed.String(), nil
}

func newHTTPClient(db *sqlite.DB, run *sqlite.Run) *http.Client {
	guard := networkGuard{db: db, run: run, resolver: resolver, dialer: net.Dialer{Timeout: 10 * time.Second}}
	return &http.Client{
		Transport: &http.Transport{DialContext: guard.DialContext},
		Timeout:   10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func headers(values http.Header) ([]model.HTTPHeader, bool) {
	headers := make([]model.HTTPHeader, 0, min(len(values), maxHeaderCount))
	truncated := len(values) > maxHeaderCount
	count := 0
	for name, vals := range values {
		if count >= maxHeaderCount {
			break
		}
		count++
		cappedValues := headerValues(name, vals)
		if len(vals) > len(cappedValues) {
			truncated = true
		}
		headers = append(headers, model.HTTPHeader{Name: name, Values: cappedValues})
	}
	return headers, truncated
}

func headerValues(name string, values []string) []string {
	if sensitiveHeader(name) {
		return []string{redactedHeaderValue}
	}
	limit := min(len(values), maxHeaderValues)
	capped := make([]string, 0, limit)
	for _, value := range values[:limit] {
		if len(value) > maxHeaderValueBytes {
			value = value[:maxHeaderValueBytes]
		}
		capped = append(capped, value)
	}
	return capped
}

func sensitiveHeader(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "authorization", "proxy-authorization", "set-cookie", "www-authenticate", "proxy-authenticate":
		return true
	default:
		return false
	}
}

func positiveContentLength(value int64) int64 {
	if value > 0 {
		return value
	}
	return 0
}

func createHTMLBodyArtifact(ctx context.Context, db *sqlite.DB, taskID, contentType string, body []byte) (string, error) {
	if len(body) == 0 || !isHTMLContentType(contentType) {
		return "", nil
	}
	artifactID := "artifact_" + uuid.NewString()
	dir, err := artifactDir(ctx, db)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, artifactID+".body")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		return "", err
	}
	artifact := sqlite.Artifact{ID: artifactID, TaskID: taskID, Path: path, CreatedAt: time.Now()}
	if err := db.CreateArtifact(ctx, artifact); err != nil {
		return "", err
	}
	return artifactID, nil
}

func isHTMLContentType(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "text/html")
}

func artifactDir(ctx context.Context, db *sqlite.DB) (string, error) {
	var seq int
	var name, path string
	if err := db.QueryRowContext(ctx, `PRAGMA database_list`).Scan(&seq, &name, &path); err != nil {
		return "", err
	}
	if path == "" {
		return filepath.Join(os.TempDir(), "penta-artifacts"), nil
	}
	return filepath.Join(filepath.Dir(path), "artifacts"), nil
}

type networkGuard struct {
	db       *sqlite.DB
	run      *sqlite.Run
	resolver ipResolver
	dialer   net.Dialer
}

func (g networkGuard) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	addrs, err := g.resolver(ctx, host)
	if err != nil {
		return nil, err
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("resolve %s: no addresses", host)
	}
	var lastErr error
	for _, addr := range addrs {
		addr = addr.Unmap()
		if err := g.allowAddress(ctx, addr); err != nil {
			lastErr = err
			continue
		}
		conn, err := g.dialer.DialContext(ctx, network, net.JoinHostPort(addr.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("no allowed addresses for %s", host)
}

func (g networkGuard) allowAddress(ctx context.Context, addr netip.Addr) error {
	if !restrictedAddress(addr) {
		return nil
	}
	if g.run == nil || g.run.SessionID == "" {
		return fmt.Errorf("blocked restricted network address %s", addr)
	}
	rules, err := g.db.ListScopeRulesBySession(ctx, g.run.SessionID)
	if err != nil {
		return err
	}
	if restrictedAddressAllowedByScope(addr, rules) {
		return nil
	}
	return fmt.Errorf("blocked restricted network address %s: not explicitly included in session scope", addr)
}

func restrictedAddress(addr netip.Addr) bool {
	addr = addr.Unmap()
	return addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() || addr.IsUnspecified() || addr.IsMulticast()
}

func restrictedAddressAllowedByScope(addr netip.Addr, rules []sqlite.ScopeRule) bool {
	addr = addr.Unmap()
	for _, rule := range rules {
		if rule.Effect == sqlite.ScopeEffectExclude && scopeRuleMatchesAddress(rule, addr) {
			return false
		}
	}
	for _, rule := range rules {
		if rule.Effect == sqlite.ScopeEffectInclude && scopeRuleMatchesAddress(rule, addr) {
			return true
		}
	}
	return false
}

func scopeRuleMatchesAddress(rule sqlite.ScopeRule, addr netip.Addr) bool {
	switch rule.TargetType {
	case sqlite.ScopeTargetIP:
		ruleAddr, err := netip.ParseAddr(rule.Value)
		return err == nil && ruleAddr.Unmap() == addr
	case sqlite.ScopeTargetCIDR:
		prefix, err := netip.ParsePrefix(rule.Value)
		return err == nil && prefix.Contains(addr)
	default:
		return false
	}
}

func defaultResolver(ctx context.Context, host string) ([]netip.Addr, error) {
	if strings.Contains(host, "%") {
		return nil, fmt.Errorf("zone identifiers are not supported: %s", host)
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		return []netip.Addr{addr}, nil
	}
	ipAddrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	addrs := make([]netip.Addr, 0, len(ipAddrs))
	for _, ipAddr := range ipAddrs {
		addr, ok := netip.AddrFromSlice(ipAddr.IP)
		if ok {
			addrs = append(addrs, addr.Unmap())
		}
	}
	return addrs, nil
}

func mustPayloadJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}
