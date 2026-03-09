package runtime

import "errors"

var (
	ErrStageNotFound   = errors.New("stage not found in plan")
	ErrInvalidPlan     = errors.New("invalid execution plan")
	ErrExecutionFailed = errors.New("execution failed")
)
