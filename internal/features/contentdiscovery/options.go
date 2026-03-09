package contentdiscovery

import (
	"fmt"
	"strings"

	"github.com/ruohao1/penta/internal/flow"
)

type Options struct {
	Targets  []string
	Method   string

	Wordlist  string
	Headers   map[string]string
	Cookies   map[string]string
	UserAgent string
	Data      string

	//Filters
	StatusCodes []int
	ResponseSize ResponseSize
	Regexps      []string

	Timeout  int
	MaxDepth int
	Workers  int
}

type ResponseSize struct {
	Min int
	Max int
}

func (o Options) Validate() error {
	if len(o.Targets) == 0 {
		return fmt.Errorf("content discovery: at least one target is required")
	}
	for _, t := range o.Targets {
		if strings.TrimSpace(t) == "" {
			return fmt.Errorf("content discovery: target cannot be empty")
		}
	}
	if strings.TrimSpace(o.Method) == "" {
		return fmt.Errorf("content discovery: method is required")
	}
	if o.MaxDepth < 0 {
		return fmt.Errorf("content discovery: max depth must be >= 0")
	}
	if o.Workers <= 0 {
		return fmt.Errorf("content discovery: workers must be > 0")
	}
	if o.Timeout <= 0 {
		return fmt.Errorf("content discovery: timeout must be > 0")
	}

	if o.ResponseSize.Min < 0 {
		return fmt.Errorf("content discovery: response size min must be >= 0")
	}
	if o.ResponseSize.Max < 0 {
		return fmt.Errorf("content discovery: response size max must be >= 0")
	}
	if o.ResponseSize.Max > 0 && o.ResponseSize.Min > o.ResponseSize.Max {
		return fmt.Errorf("content discovery: response size min cannot be greater than max")
	}
	
	if o.Wordlist == "" {
		return fmt.Errorf("content discovery: wordlist is required")
	}

	return nil
}

func (o Options) Kind() flow.Type {
	return flow.ContentDiscovery
}
