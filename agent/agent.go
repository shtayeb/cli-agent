package agent

import (
	"context"
	"encoding/json"

	"agent/tools"

	"github.com/anthropics/anthropic-sdk-go"
)

// Agent represents a conversational AI agent that can use tools
type Agent struct {
	client *anthropic.Client
	tools  []tools.ToolDefinition
}

// NewAgent creates a new agent instance
func NewAgent(client *anthropic.Client, toolDefinitions []tools.ToolDefinition) *Agent {
	return &Agent{
		client: client,
		tools:  toolDefinitions,
	}
}

// executeTool executes a tool by name with the given input
func (a *Agent) ExecuteTool(id, name string, input json.RawMessage) anthropic.ContentBlockParamUnion {
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

	// fmt.Printf("\u001b[92mtool\u001b[0m: %s(%s)\n", name, input)

	response, err := toolDef.Function(input)
	if err != nil {
		return anthropic.NewToolResultBlock(id, err.Error(), true)
	}

	return anthropic.NewToolResultBlock(id, response, false)
}

// runInference sends a message to Claude and gets a response
func (a *Agent) RunInference(ctx context.Context, conversation []anthropic.MessageParam) (*anthropic.Message, error) {
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
		Model: anthropic.ModelClaude3_7Sonnet20250219,
		// Model:     anthropic.ModelClaude_3_Haiku_20240307,
		MaxTokens: int64(1024),
		Messages:  conversation,
		Tools:     anthropicTools,
	})

	return message, err
}
