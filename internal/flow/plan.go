package flow

import "github.com/ruohao1/pipex"

type Plan struct {
	Feature Type
	Stages  []pipex.Stage[Item]
	Edges   []Edge
	Seeds   map[string][]Item
	// Optional stage-level execution overrides
	Policies StagePolicies
}

type Edge struct {
	From string
	To   string
}

type StagePolicies struct {
	RateLimits map[string]RateLimit
	Retry      map[string]RetryPolicy
	Timeout    map[string]TimeoutPolicy
}
