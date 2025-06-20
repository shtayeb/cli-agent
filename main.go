package main

import (
	"agent/agent"
	"agent/config"
	"agent/tools"
	"agent/tui"
	"log"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Initialize configuration
	cfg := config.NewConfig()

	// Get all available tools
	availableTools := tools.GetAllTools()

	// Create the agent
	agentInstance := agent.NewAgent(cfg.Client, availableTools)

	_, err := tea.NewProgram(
		tui.InitialChatModel(agentInstance),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	).Run()

	if err != nil {
		log.Fatal(err)
	}
}
