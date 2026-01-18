package world

// CCTVTerminal represents a security terminal that can reveal nearby rooms
type CCTVTerminal struct {
	Name       string
	Used       bool
	TargetRoom string // Name of the room this terminal reveals
	Cell       *Cell
}

// NewCCTVTerminal creates a new CCTV terminal
func NewCCTVTerminal(name string) *CCTVTerminal {
	return &CCTVTerminal{
		Name: name,
		Used: false,
	}
}

// Activate activates the terminal, returns true if it revealed something new
func (t *CCTVTerminal) Activate() bool {
	if t.Used {
		return false
	}
	t.Used = true
	return true
}

// IsUsed returns whether the terminal has been used
func (t *CCTVTerminal) IsUsed() bool {
	return t.Used
}
