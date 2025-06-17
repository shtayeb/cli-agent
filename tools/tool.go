package tools

import (
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/invopop/jsonschema"
)

// ToolDefinition represents a tool that can be used by the agent
type ToolDefinition struct {
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	InputSchema anthropic.ToolInputSchemaParam `json:"input_schema"`
	Function    func(input json.RawMessage) (string, error)
}

// GenerateSchema creates a JSON schema for the given type T
func GenerateSchema[T any]() anthropic.ToolInputSchemaParam {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}

	var v T
	schema := reflector.Reflect(v)

	return anthropic.ToolInputSchemaParam{
		Properties: schema.Properties,
	}
}

// GetAllTools returns all available tools
func GetAllTools() []ToolDefinition {
	return []ToolDefinition{
		ReadFileDefinition,
		ListFilesDefinition,
		EditFileDefinition,
	}
}
