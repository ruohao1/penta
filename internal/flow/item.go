package flow

type Item struct {
	Feature string
	Stage   string
	Target  string
	Key     string
	Payload any
	Meta    map[string]any
}
