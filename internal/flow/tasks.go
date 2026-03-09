package flow

type TaskOptions interface {
	Validate() error
	Kind() Type
}
