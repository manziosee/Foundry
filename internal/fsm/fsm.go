package fsm

type State string

const (
	StateNew         State = "NEW"
	StateDownloading State = "DOWNLOADING"
	StateDownloaded  State = "DOWNLOADED"
	StateUnpacking   State = "UNPACKING"
	StateUnpacked    State = "UNPACKED"
	StateStored      State = "STORED"
	StateActivating  State = "ACTIVATING"
	StateActive      State = "ACTIVE"
	StateFailed      State = "FAILED"
)

type Transition struct {
	From State
	To   State
}

var validTransitions = map[Transition]bool{
	{StateNew, StateDownloading}:         true,
	{StateDownloading, StateDownloaded}:  true,
	{StateDownloaded, StateUnpacking}:    true,
	{StateUnpacking, StateUnpacked}:      true,
	{StateUnpacked, StateStored}:         true,
	{StateStored, StateActivating}:       true,
	{StateActivating, StateActive}:       true,
	{StateNew, StateFailed}:              true,
	{StateDownloading, StateFailed}:      true,
	{StateDownloaded, StateFailed}:       true,
	{StateUnpacking, StateFailed}:        true,
	{StateUnpacked, StateFailed}:         true,
	{StateStored, StateFailed}:           true,
	{StateActivating, StateFailed}:       true,
}

func CanTransition(from, to State) bool {
	return validTransitions[Transition{from, to}]
}

func NextState(current State) State {
	switch current {
	case StateNew:
		return StateDownloading
	case StateDownloading:
		return StateDownloaded
	case StateDownloaded:
		return StateUnpacking
	case StateUnpacking:
		return StateUnpacked
	case StateUnpacked:
		return StateStored
	case StateStored:
		return StateActivating
	case StateActivating:
		return StateActive
	default:
		return current
	}
}