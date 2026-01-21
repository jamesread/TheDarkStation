package entities

import "strings"

// PuzzleType represents different types of puzzles
type PuzzleType int

const (
	PuzzleSequence PuzzleType = iota // Sequence of numbers/letters (e.g., "1-2-3-4")
	PuzzlePattern                    // Pattern puzzle (e.g., directional pattern)
)

// PuzzleTerminal represents a terminal that requires solving a puzzle
type PuzzleTerminal struct {
	Name        string
	PuzzleType  PuzzleType
	Solution    string       // The correct solution (e.g., "1-2-3-4" or "up-down-left-right")
	Hint        string       // Hint text shown when examining
	Solved      bool         // Whether the puzzle has been solved
	Reward      PuzzleReward // What the player gets for solving
	Description string       // Description of the terminal
}

// PuzzleReward represents what the player gets for solving a puzzle
type PuzzleReward int

const (
	RewardNone       PuzzleReward = iota
	RewardKeycard                 // Unlocks a door
	RewardBattery                 // Gives a battery
	RewardRevealRoom              // Reveals a room on the map
	RewardUnlockArea              // Unlocks a previously inaccessible area
	RewardMap                     // Gives the map (powerful reward for later levels)
)

// NewPuzzleTerminal creates a new puzzle terminal
func NewPuzzleTerminal(name string, puzzleType PuzzleType, solution string, hint string, reward PuzzleReward, description string) *PuzzleTerminal {
	return &PuzzleTerminal{
		Name:        name,
		PuzzleType:  puzzleType,
		Solution:    solution,
		Hint:        hint,
		Reward:      reward,
		Description: description,
		Solved:      false,
	}
}

// IsSolved returns whether the puzzle has been solved
func (p *PuzzleTerminal) IsSolved() bool {
	return p.Solved
}

// Solve marks the puzzle as solved
func (p *PuzzleTerminal) Solve() {
	p.Solved = true
}

// CheckSolution checks if the provided input matches the solution
func (p *PuzzleTerminal) CheckSolution(input string) bool {
	// Normalize input (trim, lowercase for pattern puzzles)
	var normalizedInput string
	if p.PuzzleType == PuzzlePattern {
		normalizedInput = strings.ToLower(strings.TrimSpace(input))
	} else {
		normalizedInput = strings.TrimSpace(input)
	}
	return normalizedInput == p.Solution
}
