package main

import (
	"context"
	"fmt"

	"agent/agent"
	"agent/config"
	"agent/tools"
)

func main() {
	// Initialize configuration
	cfg := config.NewConfig()

	// Get all available tools
	availableTools := tools.GetAllTools()

	// Create the agent
	agentInstance := agent.NewAgent(cfg.Client, cfg.GetUserMessage, availableTools)

	// Run the agent
	err := agentInstance.Run(context.TODO())
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}
