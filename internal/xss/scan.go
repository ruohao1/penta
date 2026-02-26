package xss

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func Scan(ctx context.Context, opts *Options) error {
	if opts == nil {
		return fmt.Errorf("options is nil")
	}
	if opts.URL == nil {
		return fmt.Errorf("url is required")
	}

	method := strings.ToUpper(strings.TrimSpace(opts.Method))
	if method == "" {
		method = http.MethodGet
	}

	client := &http.Client{
		Jar: opts.Cookies,
	}
	// 1) load payloads (opts.Wordlist or defaults)
	file, err := os.Open(opts.Wordlist)
	if err != nil {
		return fmt.Errorf("failed to open %s file: %w", opts.Wordlist, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		payload := scanner.Text()

		// 3) inject payloads into query/body/headers
		// URL
		targetURL := strings.ReplaceAll(opts.URL.String(), "FUZZ", url.QueryEscape(payload))

		// Body (form)
		var body io.Reader
		if method != http.MethodGet && len(opts.Data) > 0 {
			data := make(url.Values, len(opts.Data))
			for k, vv := range opts.Data {
				nk := strings.ReplaceAll(k, "FUZZ", payload)
				for _, v := range vv {
					data.Add(nk, strings.ReplaceAll(v, "FUZZ", payload))
				}
			}
			encoded := data.Encode()
			body = strings.NewReader(encoded)
		}

		req, err := http.NewRequestWithContext(ctx, method, targetURL, body)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		// Headers
		req.Header = make(http.Header, len(opts.Headers))
		for k, vv := range opts.Headers {
			nk := strings.ReplaceAll(k, "FUZZ", payload)
			for _, v := range vv {
				req.Header.Add(nk, strings.ReplaceAll(v, "FUZZ", payload))
			}
		}
		if opts.UserAgent != "" {
			req.Header.Set("User-Agent", strings.ReplaceAll(opts.UserAgent, "FUZZ", payload))
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}

		// 4) send requests, detect reflections/execution signals
		res, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send request: %w", err)
		}

		if opts.ExitPoint != "" && res.Request.URL.Path != opts.ExitPoint {
			res, err = client.Get(opts.URL.String() + opts.ExitPoint)
			if err != nil {
				return fmt.Errorf("failed to request exit point: %w", err)
			}
		}

		if isVulnerable(res, payload) {
			fmt.Printf("[VULNERABLE] Payload %q is reflected/executed at %s\n", payload, res.Request.URL)
		}

	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read %s file: %w", opts.Wordlist, err)
	}
	// 5) report findings

	_ = client
	_ = method
	return nil
}

func isVulnerable(resp *http.Response, payload string) bool {
  	defer resp.Body.Close()

  	// 1) Reflection in headers
  	for _, vals := range resp.Header {
  		for _, v := range vals {
  			if strings.Contains(v, payload) {
  				return true
  			}
  		}
  	}

  	// 2) Reflection in body
  	b, err := io.ReadAll(resp.Body)
  	if err != nil {
  		return false
  	}
  	body := string(b)
  	if strings.Contains(body, payload) {
  		return true
  	}

  	return false
  }


func showContext(resp *http.Response, payload string) error {
	defer resp.Body.Close()

	found := false

	// Headers
	for k, vals := range resp.Header {
		for _, v := range vals {
			if strings.Contains(v, payload) {
				fmt.Printf("[header] %s: %s\n", k, v)
				found = true
			}
		}
	}

	// Body lines
	sc := bufio.NewScanner(resp.Body)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := sc.Text()
		if strings.Contains(line, payload) {
			fmt.Printf("[body line %d] %s\n", lineNo, line)
			found = true
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}

	if !found {
		fmt.Println("payload not reflected in headers or body")
	}
	return nil
}
