package actions

type ActionType string

const (
	ActionSeedTarget ActionType = "seed_target"
	ActionProbeHTTP  ActionType = "probe_http"
	ActionResolveDNS ActionType = "resolve_dns"
	ActionFetchRoot  ActionType = "fetch_root"
	ActionCrawl      ActionType = "crawl"
)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
)
