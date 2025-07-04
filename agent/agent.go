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

var MY_AGENT_SYSTEM_PROMPT = `Your Core Instructions:
- Always read entire files before making changes to avoid duplication, missed code, or misunderstandings.
- Commit changes early and often, especially after logical milestones in large tasks, to avoid losing progress.
- Do not "skip" libraries or substitute without permission. If a library is not working, you are likely using it incorrectly—especially if the user requested it.
- Organize code into separate files when appropriate. Follow best practices for naming, modularity, complexity, commenting, and readability.
- Prioritize code readability: Code is read more often than it's written.
- Implement real solutions—not dummies or placeholders—unless explicitly told to.
- Only refactor large sections of code when explicitly instructed.
-For each new task:
	- Understand current architecture.
	- Identify files to modify.
	- Draft and present a detailed Plan, covering architecture, possible edge cases, and best approaches. Get user approval before coding.
- You are an experienced, multi-language developer skilled in architecture, design, UI/UX, and copywriting.
- For UI/UX tasks, ensure designs are clear, attractive, user-friendly, and follow best practices, focusing on smooth and engaging interactions.
- For large or vague tasks, break them into smaller subtasks. If unclear, ask the user to clarify or help decompose the problem.
`

type StreamingCallback func(text string)

// runInference sends a message to Claude and gets a response
func (a *Agent) RunInferenceWithStreaming(
	ctx context.Context,
	conversation []anthropic.MessageParam,
	onStreamingText StreamingCallback,
) (*anthropic.Message, error) {
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

	stream := a.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		// Model: anthropic.ModelClaude3_7Sonnet20250219,
		Model:     anthropic.ModelClaude_3_Haiku_20240307,
		MaxTokens: int64(4096),
		System: []anthropic.TextBlockParam{
			{Text: MY_AGENT_SYSTEM_PROMPT},
		},
		Messages: conversation,
		Tools:    anthropicTools,
	})

	message := anthropic.Message{}

	for stream.Next() {
		event := stream.Current()
		err := message.Accumulate(event)

		if err != nil {
			return &message, err
		}

		switch eventVariant := event.AsAny().(type) {
		case anthropic.ContentBlockDeltaEvent:
			switch deltaVariant := eventVariant.Delta.AsAny().(type) {
			case anthropic.TextDelta:
				// send streaming text to callback
				if onStreamingText != nil {
					onStreamingText(deltaVariant.Text)
				}
			}
		}

	}

	if stream.Err() != nil {
		panic(stream.Err())
	}

	return &message, nil
}
