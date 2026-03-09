package runtime

import (
	"time"

	"github.com/ruohao1/pipex"
)

type Config struct {
	FailFast     bool
	BufferSize   int
	Workers      int
	MaxRate      float64
	RateBurst    int
	MaxRetries   int
	RetryBackoff time.Duration
	Timeout      time.Duration
}

func DefaultConfig() Config {
	return Config{
		FailFast:     true,
		BufferSize:   1024,
		Workers:      8,
		MaxRate:      200,
		RateBurst:    200,
		MaxRetries:   2,
		RetryBackoff: 100 * time.Millisecond,
		Timeout:      10 * time.Second,
	}
}

func (c Config) ToPipelineOpts(plan Plan) []pipex.Option[Item] {
	if c.BufferSize <= 0 {
		c.BufferSize = 1024
	}
	if c.RateBurst <= 0 && c.MaxRate > 0 {
		c.RateBurst = int(c.MaxRate)
	}
	if c.Workers <= 0 {
		c.Workers = 1
	}
	if c.MaxRetries < 0 {
		c.MaxRetries = 0
	}

	opts := []pipex.Option[Item]{
		pipex.WithFailFast[Item](c.FailFast),
		pipex.WithBufferSize[Item](c.BufferSize),
	}

	stageWorkers := make(map[string]int, len(plan.Stages))
	stageRates := make(map[string]pipex.RateLimit, len(plan.Stages))
	stagePolicies := make(map[string]pipex.StagePolicy, len(plan.Stages))
	for _, stage := range plan.Stages {
		name := stage.Name()
		stageWorkers[name] = c.Workers
		if c.MaxRate > 0 {
			stageRates[name] = pipex.RateLimit{RPS: c.MaxRate, Burst: c.RateBurst}
		}
		if c.MaxRetries > 0 || c.Timeout > 0 {
			stagePolicies[name] = pipex.StagePolicy{
				MaxAttempts: c.MaxRetries + 1,
				Backoff:     c.RetryBackoff,
				Timeout:     c.Timeout,
			}
		}
	}

	if len(stageWorkers) != 0 {
		opts = append(opts, pipex.WithStageWorkers[Item](stageWorkers))
	}
	if len(stageRates) != 0 {
		opts = append(opts, pipex.WithStageRateLimits[Item](stageRates))
	}
	if len(stagePolicies) != 0 {
		opts = append(opts, pipex.WithStagePolicies[Item](stagePolicies))
	}

	return opts
}
