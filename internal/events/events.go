package events

import "time"

type Kind string

const (
	RunStarted    Kind = "run_started"
	RunFinished   Kind = "run_finished"
	RunFailed     Kind = "run_failed"
	RunCanceled   Kind = "run_canceled"

	StageStarted  Kind = "stage_started"
	StageFinished Kind = "stage_finished"
	StageFailed   Kind = "stage_failed"
	StageRetry    Kind = "stage_retry"

	Finding       Kind = "finding"
	Observation   Kind = "observation"
	Warning       Kind = "warning"
	Log           Kind = "log"
)

type Event struct {
	At      time.Time      `json:"at"`
	Kind    Kind           `json:"kind"`
	RunID   string         `json:"run_id,omitempty"`
	Feature string         `json:"feature,omitempty"`
	Stage   string         `json:"stage,omitempty"`
	Target  string         `json:"target,omitempty"`
	Message string         `json:"message,omitempty"`
	Err     string         `json:"err,omitempty"`
	Data    map[string]any `json:"data,omitempty"`
}
