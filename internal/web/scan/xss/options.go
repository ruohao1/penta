package xss

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

type Options struct {
	URL        *url.URL
	EntryPoint string
	ExitPoint  string

	Headers   http.Header
	UserAgent string
	Cookies   *cookiejar.Jar
	Data      url.Values
	Method    string

	Wordlist string
}

func NewOptions() (*Options,error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	return &Options{
		Headers: make(http.Header),
		Cookies: jar,
		Data:    make(url.Values),
		Method:  "GET",
	}, nil
}

func (opts *Options) SetHeaders(headersExpr string) error {
	headersExpr = strings.TrimSpace(headersExpr)
	if headersExpr == "" {
		return nil
	}

	for _, pair := range strings.Split(headersExpr, ";") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid header format %q (expected Key: Value)", pair)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			return fmt.Errorf("invalid header %q: empty key", pair)
		}

		if strings.EqualFold(key, "User-Agent") && opts.UserAgent == "" {
			opts.UserAgent = value
			continue
		}
		if strings.EqualFold(key, "Cookie") {
			if err := opts.AddCookie(value); err != nil {
				return fmt.Errorf("invalid Cookie header: %w", err)
			}
			continue
		}

		opts.Headers.Set(key, value)
	}

	return nil
}

func (opts *Options) AddCookie(cookieExpr string) error {
	if opts.URL == nil || opts.Cookies == nil {
		return fmt.Errorf("cannot set cookies: URL or cookie jar is nil")
	}
	parsed, err := parseCookies(cookieExpr)
	if err != nil {
		return err
	}
	opts.Cookies.SetCookies(opts.URL, parsed)
	return nil
}

func (opts *Options) SetCookies(cookiesExpr string) error {
	cookiesExpr = strings.TrimSpace(cookiesExpr)
	if cookiesExpr == "" {
		return nil
	}
	if opts.URL == nil || opts.Cookies == nil {
		return fmt.Errorf("cannot set cookies: URL or cookie jar is nil")
	}
	parsed, err := parseCookies(cookiesExpr)
	if err != nil {
		return err
	}

	// replace all cookies for this URL
	existing := opts.Cookies.Cookies(opts.URL)
	for _, c := range existing {
		opts.Cookies.SetCookies(opts.URL, []*http.Cookie{{
			Name:   c.Name,
			Value:  "",
			Path:   c.Path,
			Domain: c.Domain,
			MaxAge: -1, // delete
		}})
	}

	opts.Cookies.SetCookies(opts.URL, parsed)
	return nil
}

func parseCookies(expr string) ([]*http.Cookie, error) {
	var out []*http.Cookie
	for _, pair := range strings.Split(expr, ";") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid cookie format %q (expected key=value)", pair)
		}
		name := strings.TrimSpace(parts[0])
		if name == "" {
			return nil, fmt.Errorf("invalid cookie %q: empty name", pair)
		}
		out = append(out, &http.Cookie{
			Name:  name,
			Value: strings.TrimSpace(parts[1]),
			Path:  "/",
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no valid cookies found")
	}
	return out, nil
}

// SetData parses a data expression and adds the key-value pairs to the Data field
func (opts *Options) SetData(dataExpr string) error {
	// dataExpr can be in JSON format: {"key": "value", "key2": "value2"}
	// or in key=value&key2=value2 format
	dataExpr = strings.TrimSpace(dataExpr)
	if dataExpr == "" {
		return nil
	}

	// Try JSON first
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(dataExpr), &data); err == nil {
		if len(data) == 0 {
			return fmt.Errorf("invalid data expression: JSON object is empty")
		}
		for k, v := range data {
			key := strings.TrimSpace(k)
			if key == "" {
				return fmt.Errorf("invalid data expression: empty JSON key")
			}
			opts.Data.Set(key, fmt.Sprint(v))
		}
		return nil
	}

	// Fallback: key=value&key2=value2
	parsedAny := false
	for _, pair := range strings.Split(dataExpr, "&") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid data format %q (expected key=value)", pair)
		}
		key := strings.TrimSpace(parts[0])
		if key == "" {
			return fmt.Errorf("invalid data %q: empty key", pair)
		}
		opts.Data.Set(key, strings.TrimSpace(parts[1]))
		parsedAny = true
	}
	if !parsedAny {
		return fmt.Errorf("invalid data expression: no key-value pairs found")
	}
	return nil
}
