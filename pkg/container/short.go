package container

type State string

const (
	Green  State = "green"
	YELLOW State = "yellow"
	Red    State = "red"
)

func StateFrom(status string) State {
	switch status {
	case "running":
		return Green
	case "exited":
		return Red
	default:
		return YELLOW
	}
}

func (s State) String() string {
	return string(s)
}

type Short struct {
	ID     string
	Name   string
	Image  string
	Status string
	State  State
}
