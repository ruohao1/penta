package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/ruohao1/penta/internal/config"
	"github.com/ruohao1/penta/internal/controller"
	"github.com/ruohao1/penta/internal/features/contentdiscovery"
	"github.com/ruohao1/penta/internal/flow"
	"github.com/ruohao1/penta/internal/runtime"
	"github.com/ruohao1/penta/internal/sinks"
	"github.com/spf13/cobra"
)

func newWebCmd() *cobra.Command {
	webCmd := &cobra.Command{
		Use:   "web",
		Short: "Web application security testing",
	}

	webCmd.AddCommand(newContentDiscoveryCmd())
	return webCmd
}

func newFuzzCmd() *cobra.Command {
	fuzzCmd := &cobra.Command{
		Use:   "fuzz",
		Short: "Web application fuzzing",
	}

	return fuzzCmd
}

func newContentDiscoveryCmd() *cobra.Command {
	var (
		targets []string
		method  string

		wordlist  string
		headers   []string
		headersKV map[string]string
		cookies   map[string]string
		userAgent string
		data      string

		maxDepth int
		workers  int
		timeout  int
	)

	cmd := &cobra.Command{
		Use:   "content-discovery [target]",
		Short: "Discover web content",
		Args:  cobra.MaximumNArgs(1),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				targets = append(targets, strings.TrimSpace(args[0]))
			}
			targets = compactNonEmpty(targets)
			if len(targets) == 0 {
				_ = cmd.Usage()
				return fmt.Errorf("content discovery: at least one target is required")
			}
			if strings.TrimSpace(wordlist) == "" {
				_ = cmd.Usage()
				return fmt.Errorf("content discovery: --wordlist is required")
			}
			if _, err := os.Stat(wordlist); err != nil {
				return fmt.Errorf("content discovery: wordlist: %w", err)
			}
			parsedHeaders, err := parseHeaders(headers)
			if err != nil {
				return err
			}
			headersKV = parsedHeaders
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			runCfg, ok := config.RuntimeConfigFrom(cmd.Context())
			if !ok {
				runCfg = runtime.DefaultConfig()
			}

			task := contentdiscovery.Options{
				Targets:   targets,
				Method:    strings.ToUpper(strings.TrimSpace(method)),
				Wordlist:  wordlist,
				Headers:   headersKV,
				Cookies:   cookies,
				UserAgent: userAgent,
				Data:      data,
				MaxDepth:  maxDepth,
				Workers:   workers,
				Timeout:   timeout,
			}

			if err := task.Validate(); err != nil {
				return err
			}

			consoleSink := sinks.NewConsoleSink(cmd.OutOrStdout())
			rt := runtime.New(runCfg, consoleSink)
			ctrl := controller.New(rt, consoleSink)
			session, err := ctrl.Start(cmd.Context(), controller.StartInput{
				Feature: flow.ContentDiscovery,
				Task:    task,
				Run:     runCfg,
			})
			if err != nil {
				return err
			}

			if session == nil || session.Done == nil {
				return fmt.Errorf("content discovery: invalid session")
			}
			return <-session.Done
		},
	}

	cmd.Flags().StringSliceVarP(&targets, "targets", "t", nil, "Comma-separated list of targets (IP, CIDR, or http(s) URL)")
	cmd.Flags().StringVarP(&method, "method", "m", "GET", "HTTP method")
	cmd.Flags().StringVarP(&wordlist, "wordlist", "w", "", "Path to wordlist for content discovery")
	cmd.Flags().StringArrayVarP(&headers, "header", "H", nil, "Custom HTTP header (repeatable, e.g. --header \"X-Api-Key: 12345\")")
	cmd.Flags().StringToStringVarP(&cookies, "cookie", "c", nil, "Custom cookies (e.g. --cookie \"sessionid=abc123\")")
	cmd.Flags().StringVarP(&userAgent, "user-agent", "A", "", "Custom User-Agent header")
	cmd.Flags().StringVarP(&data, "data", "d", "", "HTTP request body for POST/PUT requests")

	cmd.Flags().IntVar(&maxDepth, "max-depth", 3, "Maximum recursion depth")
	cmd.Flags().IntVar(&workers, "workers", 8, "Override workers for content discovery stage")
	cmd.Flags().IntVar(&timeout, "timeout", 10, "Override timeout for content discovery stage (in seconds)")

	return cmd
}

func compactNonEmpty(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		v := strings.TrimSpace(item)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func parseHeaders(values []string) (map[string]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(values))
	for _, raw := range values {
		line := strings.TrimSpace(raw)
		if line == "" {
			return nil, fmt.Errorf("content discovery: header cannot be empty")
		}

		key, value, ok := strings.Cut(line, ":")
		if !ok {
			key, value, ok = strings.Cut(line, "=")
		}
		if !ok {
			return nil, fmt.Errorf("content discovery: invalid header %q (use Key: Value)", raw)
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return nil, fmt.Errorf("content discovery: invalid header %q (empty header key)", raw)
		}
		out[key] = value
	}
	return out, nil
}
