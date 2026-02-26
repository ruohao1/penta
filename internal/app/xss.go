package app

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/Ruohao1/penta/internal/xss"
	"github.com/spf13/cobra"
)

func newXSSCmd() *cobra.Command {
	opts := xss.NewOptions()
	var cookiesExpr string
	var headersExpr string
	var dataExpr string
	var urlExpr string

	xssCmd := &cobra.Command{
		Use:   "xss",
		Short: "Search for cross-site scripting (XSS) vulnerabilities in a web application",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			opts.URL, err = url.Parse(urlExpr)
			if err != nil {
				return err
			}
			opts.EntryPoint = strings.TrimRight(strings.TrimSpace(opts.EntryPoint), "/")
			opts.ExitPoint = strings.TrimRight(strings.TrimSpace(opts.ExitPoint), "/")
			if err := opts.SetHeaders(headersExpr); err != nil {
				return err
			}
			if err := opts.SetCookies(cookiesExpr); err != nil {
				return err
			}
			if dataExpr != "" {
				opts.Method = http.MethodPost
			}
			if err := opts.SetData(dataExpr); err != nil {
				return err
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return xss.Scan(cmd.Context(), opts)
		},
	}

	xssCmd.Flags().StringVarP(&urlExpr, "url", "u", "", "The URL of the web application to test for XSS vulnerabilities")
	_ = xssCmd.MarkFlagRequired("url")
	xssCmd.Flags().StringVar(&opts.EntryPoint, "entrypoint", "", "Optional entry point to test for XSS vulnerabilities (e.g., /search)")
	xssCmd.Flags().StringVar(&opts.ExitPoint, "exitpoint", opts.EntryPoint, "Optional exit point to test for XSS vulnerabilities (e.g., /result)")

	xssCmd.Flags().StringVarP(&headersExpr, "header", "H", "", "Optional headers to include in the request, in the format 'Header-Name: Header-Value'")
	xssCmd.Flags().StringVarP(&opts.UserAgent, "user-agent", "A", "", "Optional User-Agent string to include in the request")
	xssCmd.Flags().StringVarP(&cookiesExpr, "cookies", "c", "", "Optional cookies to include in the request, in the format 'key=value; key2=value2'")
	xssCmd.Flags().StringVarP(&dataExpr, "data", "d", "", "Optional data to include in the request body, for POST requests")
	xssCmd.Flags().StringVarP(&opts.Method, "method", "X", "GET", "HTTP method to use for the request (e.g., GET, POST)")

	xssCmd.Flags().StringVarP(&opts.Wordlist, "wordlist", "w", "", "Path to a wordlist file containing XSS payloads to test against the target URL")

	return xssCmd
}
