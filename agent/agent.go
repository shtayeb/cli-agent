package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"agent/tools"

	"github.com/anthropics/anthropic-sdk-go"
)

// Agent represents a conversational AI agent that can use tools
type Agent struct {
	client         *anthropic.Client
	getUserMessage func() (string, bool)
	tools          []tools.ToolDefinition
}

// NewAgent creates a new agent instance
func NewAgent(client *anthropic.Client, getUserMessage func() (string, bool), toolDefinitions []tools.ToolDefinition) *Agent {
	return &Agent{
		client:         client,
		getUserMessage: getUserMessage,
		tools:          toolDefinitions,
	}
}

// Run starts the agent's main conversation loop
func (a *Agent) Run(ctx context.Context) error {
	conversation := []anthropic.MessageParam{}

	fmt.Println("Chat with Claude (use 'ctrl+c' to quit)")

	readUserInput := true

	for {
		if readUserInput {
			// colored 'you'
			fmt.Print("\u001b[94mYou\u001b[0m: ")

			userInput, ok := a.getUserMessage()
			if !ok {
				break
			}

			userMessage := anthropic.NewUserMessage(anthropic.NewTextBlock(userInput))
			conversation = append(conversation, userMessage)
		}

		message, err := a.runInference(ctx, conversation)
		if err != nil {
			return err
		}

		conversation = append(conversation, message.ToParam())

		toolResults := []anthropic.ContentBlockParamUnion{}
		for _, content := range message.Content {
			switch content.Type {
			case "text":
				fmt.Printf("\u001b[93mClaude\u001b[0m: %s\n", content.Text)
			case "tool_use":
				result := a.executeTool(content.ID, content.Name, content.Input)
				toolResults = append(toolResults, result)
			}
		}

		if len(toolResults) == 0 {
			readUserInput = true
			continue
		}

		readUserInput = false
		conversation = append(conversation, anthropic.NewUserMessage(toolResults...))
	}

	return nil
}

// executeTool executes a tool by name with the given input
func (a *Agent) executeTool(id, name string, input json.RawMessage) anthropic.ContentBlockParamUnion {
	var toolDef tools.ToolDefinition
	var found bool

	for _, tool := range a.tools {
		if tool.Name == name {
			toolDef = tool
			found = true
			break
		}
	}

	if !found {
		return anthropic.NewToolResultBlock(id, "tool not found", true)
	}

	fmt.Printf("\u001b[92mtool\u001b[0m: %s(%s)\n", name, input)

	response, err := toolDef.Function(input)
	if err != nil {
		return anthropic.NewToolResultBlock(id, err.Error(), true)
	}

	return anthropic.NewToolResultBlock(id, response, false)
}

// runInference sends a message to Claude and gets a response
func (a *Agent) runInference(ctx context.Context, conversation []anthropic.MessageParam) (*anthropic.Message, error) {
	anthropicTools := []anthropic.ToolUnionParam{}

	for _, tool := range a.tools {
		anthropicTools = append(anthropicTools, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        tool.Name,
				Description: anthropic.String(tool.Description),
				InputSchema: tool.InputSchema,
			},
		})
	}

	message, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_7Sonnet20250219,
		MaxTokens: int64(1024),
		Messages:  conversation,
		Tools:     anthropicTools,
	})

	return message, err
}
