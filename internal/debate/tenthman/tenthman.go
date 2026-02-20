package tenthman

import (
	"fmt"

	"github.com/lorenzotomasdiez/tenth-man-rule/internal/debate"
)

// Activator implements debate.TenthManActivator.
type Activator struct{}

// NewActivator creates a new Activator.
func NewActivator() *Activator {
	return &Activator{}
}

// BuildAgent returns an Agent configured as The Tenth Man.
func (a *Activator) BuildAgent(consensusPosition string, agentID int, model string) debate.Agent {
	return debate.Agent{
		ID:    agentID,
		Name:  "The Tenth Man",
		Model: model,
		Role:  "tenth-man",
	}
}

// SystemPrompt returns the contrarian system prompt for the Tenth Man.
func (a *Activator) SystemPrompt(consensusPosition string) string {
	return fmt.Sprintf(
		"You are The Tenth Man. The group has reached consensus on the following position: %s. "+
			"You are OBLIGATED to argue the contrary position — not as token opposition, but with genuine analytical rigor. "+
			"Build the strongest possible case AGAINST the consensus. "+
			"Investigate, find evidence, construct scenarios where the majority is wrong. "+
			"Be thorough but concise.",
		consensusPosition,
	)
}
