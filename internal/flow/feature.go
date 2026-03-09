package flow

type Type string

const (
	ContentDiscovery Type = "content-discovery"
)

type BuildInput struct {
	Task TaskOptions
	Global map[string]any
}

