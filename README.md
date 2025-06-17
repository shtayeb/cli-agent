# Agent CLI with Go

A conversational AI CLI agent built with Go that uses the Anthropic Claude API and supports tool usage for file operations.

## Project Structure

```
agent-cli-with-go/
├── main.go              # Entry point - minimal, just wires everything together
├── agent/
│   └── agent.go         # Core agent logic and conversation handling
├── config/
│   └── config.go        # Configuration setup and client initialization
├── tools/
│   ├── tool.go          # Tool definition types and utilities
│   └── file_tools.go    # File operation tools (read, list, edit)
├── go.mod
├── go.sum
└── README.md
```

## Building and Running

### Prerequisites
- Go 1.23.5 or later
- Anthropic API key (set via environment variable `ANTHROPIC_API_KEY`)

### Build
```bash
go build -o cli-agent .
```

### Run
```bash
./cli-agent
```

The agent will start an interactive conversation where you can:
- Ask questions and get responses from Claude
- Request file operations (reading, listing, editing files)
- Use natural language to interact with your file system

### Available Tools
- **read_file**: Read the contents of any file
- **list_files**: List files and directories (recursively)
- **edit_file**: Edit files using find/replace operations

## Adding New Tools

To add a new tool:

1. Create the tool definition in `tools/` (or a new file):
```go
var MyNewTool = ToolDefinition{
    Name:        "my_tool",
    Description: "Description of what this tool does",
    InputSchema: MyToolInputSchema,
    Function:    MyToolFunction,
}
```

2. Add input type and schema:
```go
type MyToolInput struct {
    Param string `json:"param" jsonschema_description:"Parameter description"`
}

var MyToolInputSchema = GenerateSchema[MyToolInput]()
```

3. Implement the function:
```go
func MyToolFunction(input json.RawMessage) (string, error) {
    // Implementation
}
```

4. Add to `GetAllTools()` in `tools/tool.go`

## Dependencies

- `github.com/anthropics/anthropic-sdk-go`: Anthropic Claude API client
- `github.com/invopop/jsonschema`: JSON schema generation for tool definitions
