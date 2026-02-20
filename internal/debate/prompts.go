package debate

import (
	"fmt"

	"github.com/lorenzotomasdiez/tenth-man-rule/internal/openrouter"
)

func agentSystemPrompt(agent Agent, topic string) string {
	return fmt.Sprintf("You are %s, a debate participant. The topic is: %s. Provide your analysis and perspective. Be concise but thorough.", agent.Name, topic)
}

func phase2SystemPrompt(agent Agent, topic string) string {
	return fmt.Sprintf("You are %s, a debate participant. The topic is: %s. The Tenth Man has been activated and is arguing against the group consensus. You MUST directly engage with the Tenth Man's arguments — address them specifically, refute or acknowledge them. Be concise but thorough.", agent.Name, topic)
}

func buildMessages(agent Agent, topic string, transcript *Transcript, tenthMan TenthManActivator, consensusPosition string) []openrouter.Message {
	var systemPrompt string
	if agent.Role == "tenth-man" && tenthMan != nil {
		systemPrompt = tenthMan.SystemPrompt(consensusPosition)
	} else if transcript.Phase == TenthManPhase && agent.Role != "tenth-man" {
		systemPrompt = phase2SystemPrompt(agent, topic)
	} else {
		systemPrompt = agentSystemPrompt(agent, topic)
	}

	msgs := []openrouter.Message{
		{Role: "system", Content: systemPrompt},
	}
	for _, turn := range transcript.Turns {
		msgs = append(msgs, openrouter.Message{
			Role:    "user",
			Content: fmt.Sprintf("%s: %s", turn.Agent.Name, turn.Content),
		})
	}
	msgs = append(msgs, openrouter.Message{
		Role:    "user",
		Content: "It's your turn to speak. Provide your perspective on the topic.",
	})
	return msgs
}
