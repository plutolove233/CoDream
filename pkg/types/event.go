package types

const (
	EventMessage = "message"
	EventDone    = "done"
	EventError   = "error"
)

type Event struct {
	Type string
	Data string
}
