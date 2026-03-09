package messages

type CommandSubmittedMsg struct {
	Raw string
}
type NotifyLevel string

const (
	NotifyInfo  NotifyLevel = "info"
	NotifyWarn  NotifyLevel = "warn"
	NotifyError NotifyLevel = "error"
)

type NotifyMsg struct {
	Level   NotifyLevel
	Title   string
	Message string
}
