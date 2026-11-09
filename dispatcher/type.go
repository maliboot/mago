package dispatcher

type Type int

const (
	CPU Type = iota
	IO
	HeavyIO
)

func (t Type) String() string {
	switch t {
	case CPU:
		return "CPU"
	case IO:
		return "IO"
	case HeavyIO:
		return "HeavyIO"
	default:
		return "unknown"
	}
}
