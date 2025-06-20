package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ReadFile tool definition and implementation
var ReadFileDefinition = ToolDefinition{
	Name:        "read_file",
	Description: "Read the contents of a given relative file path. Use this when you want to see what's inside a file. Do not use this with directory names.",
	InputSchema: ReadFileInputSchema,
	Function:    ReadFile,
}

type ReadFileInput struct {
	Path      string `json:"path" jsonschema_description:"The relative path of a file in the working directory."`
	StartLine *int   `json:"start_line,omitempty" jsonschema_description:"Optional starting line number (1-based). If provided, only reads from this line onwards."`
	EndLine   *int   `json:"end_line,omitempty" jsonschema_description:"Optional ending line number (1-based). If provided with start_line, reads only the specified range."`
}

var ReadFileInputSchema = GenerateSchema[ReadFileInput]()

func ReadFile(input json.RawMessage) (string, error) {
	readFileInput := ReadFileInput{}

	err := json.Unmarshal(input, &readFileInput)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	if readFileInput.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	content, err := os.ReadFile(readFileInput.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// If no line range specified, return full content
	if readFileInput.StartLine == nil && readFileInput.EndLine == nil {
		return string(content), nil
	}

	// Split content into lines for range reading
	lines := strings.Split(string(content), "\n")
	totalLines := len(lines)

	startLine := 1
	endLine := totalLines

	if readFileInput.StartLine != nil {
		if *readFileInput.StartLine < 1 {
			return "", fmt.Errorf("start_line must be >= 1")
		}
		startLine = *readFileInput.StartLine
	}

	if readFileInput.EndLine != nil {
		if *readFileInput.EndLine < 1 {
			return "", fmt.Errorf("end_line must be >= 1")
		}
		endLine = *readFileInput.EndLine
	}

	if startLine > endLine {
		return "", fmt.Errorf("start_line cannot be greater than end_line")
	}

	if startLine > totalLines {
		return "", fmt.Errorf("start_line (%d) exceeds total lines (%d)", startLine, totalLines)
	}

	// Adjust for 0-based indexing
	startIdx := startLine - 1
	endIdx := endLine
	if endIdx > totalLines {
		endIdx = totalLines
	}

	selectedLines := lines[startIdx:endIdx]
	return strings.Join(selectedLines, "\n"), nil
}

// ListFiles tool definition and implementation
var ListFilesDefinition = ToolDefinition{
	Name:        "list_files",
	Description: "List files and directories at a given path. If no path is provided, lists files in the current directory.",
	InputSchema: ListFilesInputSchema,
	Function:    ListFiles,
}

type ListFilesInput struct {
	Path      string `json:"path,omitempty" jsonschema_description:"Optional relative path to list files from. Defaults to current directory if not provided."`
	Recursive bool   `json:"recursive,omitempty" jsonschema_description:"Whether to list files recursively. Defaults to true."`
	MaxDepth  *int   `json:"max_depth,omitempty" jsonschema_description:"Maximum depth to recurse. Only applies if recursive is true."`
}

var ListFilesInputSchema = GenerateSchema[ListFilesInput]()

func ListFiles(input json.RawMessage) (string, error) {
	listFilesInput := ListFilesInput{}
	err := json.Unmarshal(input, &listFilesInput)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	dir := "."
	if listFilesInput.Path != "" {
		dir = listFilesInput.Path
	}

	// Default to recursive if not specified
	recursive := true
	// Only override if explicitly specified in the input
	if !listFilesInput.Recursive {
		recursive = false
	}

	var files []string

	if !recursive {
		// Non-recursive listing
		entries, err := os.ReadDir(dir)
		if err != nil {
			return "", fmt.Errorf("failed to read directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				files = append(files, entry.Name()+"/")
			} else {
				files = append(files, entry.Name())
			}
		}
	} else {
		// Recursive listing with optional depth limit
		maxDepth := -1
		if listFilesInput.MaxDepth != nil {
			maxDepth = *listFilesInput.MaxDepth
		}

		err = filepath.Walk(dir, func(filePath string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(dir, filePath)
			if err != nil {
				return err
			}

			// Skip the root directory itself
			if relPath == "." {
				return nil
			}

			// Check depth limit
			if maxDepth >= 0 {
				depth := strings.Count(relPath, string(filepath.Separator))
				if depth > maxDepth {
					if info.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}

			if info.IsDir() {
				files = append(files, relPath+"/")
			} else {
				files = append(files, relPath)
			}
			return nil
		})

		if err != nil {
			return "", fmt.Errorf("failed to walk directory: %w", err)
		}
	}

	result, err := json.Marshal(files)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(result), nil
}

// CreateFile tool definition and implementation
var CreateFileDefinition = ToolDefinition{
	Name:        "create_file",
	Description: "Create a new file with the specified content. If the file already exists, it will return an error unless overwrite is true.",
	InputSchema: CreateFileInputSchema,
	Function:    CreateFile,
}

type CreateFileInput struct {
	Path      string `json:"path" jsonschema_description:"The path where the file should be created."`
	Content   string `json:"content" jsonschema_description:"The content to write to the file."`
	Overwrite bool   `json:"overwrite,omitempty" jsonschema_description:"Whether to overwrite the file if it already exists. Defaults to false."`
}

var CreateFileInputSchema = GenerateSchema[CreateFileInput]()

func CreateFile(input json.RawMessage) (string, error) {
	createFileInput := CreateFileInput{}
	err := json.Unmarshal(input, &createFileInput)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	if createFileInput.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	// Check if file exists
	if _, err := os.Stat(createFileInput.Path); err == nil {
		if !createFileInput.Overwrite {
			return "", fmt.Errorf("file already exists: %s (use overwrite=true to replace)", createFileInput.Path)
		}
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(createFileInput.Path)
	if dir != "." && dir != "" {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return "", fmt.Errorf("failed to create directory: %w", err)
		}
	}

	err = os.WriteFile(createFileInput.Path, []byte(createFileInput.Content), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}

	return fmt.Sprintf("Successfully created file: %s", createFileInput.Path), nil
}

// EditFile tool definition and implementation
var EditFileDefinition = ToolDefinition{
	Name: "edit_file",
	Description: `Make edits to an existing file using various modes:
	- 'replace': Replace old_str with new_str (must match exactly once)
	- 'insert_after': Insert new_str after the line containing old_str
	- 'insert_before': Insert new_str before the line containing old_str
	- 'append': Append new_str to the end of the file
	- 'prepend': Prepend new_str to the beginning of the file
	- 'delete_line': Delete the line containing old_str
	`,
	InputSchema: EditFileInputSchema,
	Function:    EditFile,
}

type EditFileInput struct {
	Path       string `json:"path" jsonschema_description:"The path to the file to edit."`
	Mode       string `json:"mode" jsonschema_description:"Edit mode: 'replace', 'insert_after', 'insert_before', 'append', 'prepend', or 'delete_line'."`
	OldStr     string `json:"old_str,omitempty" jsonschema_description:"Text to search for (required for replace, insert_after, insert_before, delete_line modes)."`
	NewStr     string `json:"new_str,omitempty" jsonschema_description:"Text to insert/replace with (required for replace, insert_after, insert_before, append, prepend modes)."`
	LineNumber *int   `json:"line_number,omitempty" jsonschema_description:"Specific line number for insert operations (1-based, optional alternative to old_str)."`
}

var EditFileInputSchema = GenerateSchema[EditFileInput]()

func EditFile(input json.RawMessage) (string, error) {
	editFileInput := EditFileInput{}
	err := json.Unmarshal(input, &editFileInput)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	if editFileInput.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	if editFileInput.Mode == "" {
		return "", fmt.Errorf("mode is required")
	}

	// Validate mode
	validModes := []string{"replace", "insert_after", "insert_before", "append", "prepend", "delete_line"}
	isValidMode := false
	for _, mode := range validModes {
		if editFileInput.Mode == mode {
			isValidMode = true
			break
		}
	}
	if !isValidMode {
		return "", fmt.Errorf("invalid mode: %s. Valid modes are: %s", editFileInput.Mode, strings.Join(validModes, ", "))
	}

	// Read existing file
	content, err := os.ReadFile(editFileInput.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	switch editFileInput.Mode {
	case "append":
		if editFileInput.NewStr == "" {
			return "", fmt.Errorf("new_str is required for append mode")
		}
		lines = append(lines, editFileInput.NewStr)

	case "prepend":
		if editFileInput.NewStr == "" {
			return "", fmt.Errorf("new_str is required for prepend mode")
		}
		lines = append([]string{editFileInput.NewStr}, lines...)

	case "replace":
		if editFileInput.OldStr == "" || editFileInput.NewStr == "" {
			return "", fmt.Errorf("both old_str and new_str are required for replace mode")
		}
		if editFileInput.OldStr == editFileInput.NewStr {
			return "", fmt.Errorf("old_str and new_str must be different")
		}

		originalContent := string(content)
		newContent := strings.Replace(originalContent, editFileInput.OldStr, editFileInput.NewStr, -1)

		// Count occurrences to ensure exactly one match
		occurrences := strings.Count(originalContent, editFileInput.OldStr)
		if occurrences == 0 {
			return "", fmt.Errorf("old_str not found in file")
		}
		if occurrences > 1 {
			return "", fmt.Errorf("old_str found %d times, expected exactly 1 occurrence for safety", occurrences)
		}

		err = os.WriteFile(editFileInput.Path, []byte(newContent), 0644)
		if err != nil {
			return "", fmt.Errorf("failed to write file: %w", err)
		}
		return "Successfully replaced text in file", nil

	case "insert_after", "insert_before", "delete_line":
		if editFileInput.NewStr == "" && editFileInput.Mode != "delete_line" {
			return "", fmt.Errorf("new_str is required for %s mode", editFileInput.Mode)
		}

		var targetLine int = -1

		// Use line number if provided, otherwise search for old_str
		if editFileInput.LineNumber != nil {
			if *editFileInput.LineNumber < 1 || *editFileInput.LineNumber > len(lines) {
				return "", fmt.Errorf("line_number %d is out of range (1-%d)", *editFileInput.LineNumber, len(lines))
			}
			targetLine = *editFileInput.LineNumber - 1 // Convert to 0-based
		} else {
			if editFileInput.OldStr == "" {
				return "", fmt.Errorf("either old_str or line_number is required for %s mode", editFileInput.Mode)
			}

			// Find the line containing old_str
			matchCount := 0
			for i, line := range lines {
				if strings.Contains(line, editFileInput.OldStr) {
					targetLine = i
					matchCount++
				}
			}

			if matchCount == 0 {
				return "", fmt.Errorf("old_str not found in file")
			}
			if matchCount > 1 {
				return "", fmt.Errorf("old_str found in %d lines, expected exactly 1 for safety", matchCount)
			}
		}

		// Perform the operation
		switch editFileInput.Mode {
		case "insert_after":
			newLines := make([]string, 0, len(lines)+1)
			newLines = append(newLines, lines[:targetLine+1]...)
			newLines = append(newLines, editFileInput.NewStr)
			newLines = append(newLines, lines[targetLine+1:]...)
			lines = newLines

		case "insert_before":
			newLines := make([]string, 0, len(lines)+1)
			newLines = append(newLines, lines[:targetLine]...)
			newLines = append(newLines, editFileInput.NewStr)
			newLines = append(newLines, lines[targetLine:]...)
			lines = newLines

		case "delete_line":
			newLines := make([]string, 0, len(lines)-1)
			newLines = append(newLines, lines[:targetLine]...)
			newLines = append(newLines, lines[targetLine+1:]...)
			lines = newLines
		}

	default:
		return "", fmt.Errorf("unsupported mode: %s", editFileInput.Mode)
	}

	// Write the modified content back to file
	newContent := strings.Join(lines, "\n")
	err = os.WriteFile(editFileInput.Path, []byte(newContent), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("Successfully edited file using %s mode", editFileInput.Mode), nil
}

// AppendToFile tool definition and implementation
var AppendToFileDefinition = ToolDefinition{
	Name:        "append_to_file",
	Description: "Append content to the end of an existing file. Creates the file if it doesn't exist.",
	InputSchema: AppendToFileInputSchema,
	Function:    AppendToFile,
}

type AppendToFileInput struct {
	Path    string `json:"path" jsonschema_description:"The path to the file to append to."`
	Content string `json:"content" jsonschema_description:"The content to append to the file."`
	NewLine bool   `json:"newline,omitempty" jsonschema_description:"Whether to add a newline before the content. Defaults to true."`
}

var AppendToFileInputSchema = GenerateSchema[AppendToFileInput]()

func AppendToFile(input json.RawMessage) (string, error) {
	appendInput := AppendToFileInput{}
	err := json.Unmarshal(input, &appendInput)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	if appendInput.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(appendInput.Path)
	if dir != "." && dir != "" {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return "", fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Open file for appending, create if it doesn't exist
	file, err := os.OpenFile(appendInput.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Check if we need to add a newline (default to true)
	addNewline := true
	if !appendInput.NewLine {
		addNewline = appendInput.NewLine
	}

	// Check if file has content and doesn't end with newline
	if addNewline {
		stat, err := file.Stat()
		if err == nil && stat.Size() > 0 {
			// Read the last byte to check if it's a newline
			_, err = file.Seek(-1, 2) // Seek to last byte
			if err == nil {
				lastByte := make([]byte, 1)
				_, err = file.Read(lastByte)
				if err == nil && lastByte[0] != '\n' {
					_, err = file.WriteString("\n")
					if err != nil {
						return "", fmt.Errorf("failed to write newline: %w", err)
					}
				}
			}
			// Seek back to end for appending
			_, err = file.Seek(0, 2)
			if err != nil {
				return "", fmt.Errorf("failed to seek to end: %w", err)
			}
		}
	}

	_, err = file.WriteString(appendInput.Content)
	if err != nil {
		return "", fmt.Errorf("failed to append content: %w", err)
	}

	return fmt.Sprintf("Successfully appended content to: %s", appendInput.Path), nil
}

// GetFileInfo tool definition and implementation
var GetFileInfoDefinition = ToolDefinition{
	Name:        "get_file_info",
	Description: "Get information about a file or directory (size, permissions, modification time, etc.).",
	InputSchema: GetFileInfoInputSchema,
	Function:    GetFileInfo,
}

type GetFileInfoInput struct {
	Path string `json:"path" jsonschema_description:"The path to get information about."`
}

var GetFileInfoInputSchema = GenerateSchema[GetFileInfoInput]()

type FileInfo struct {
	Path        string `json:"path"`
	IsDirectory bool   `json:"is_directory"`
	Size        int64  `json:"size"`
	Mode        string `json:"mode"`
	ModTime     string `json:"mod_time"`
	LineCount   *int   `json:"line_count,omitempty"`
	Exists      bool   `json:"exists"`
}

func GetFileInfo(input json.RawMessage) (string, error) {
	getFileInfoInput := GetFileInfoInput{}
	err := json.Unmarshal(input, &getFileInfoInput)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	if getFileInfoInput.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	info, err := os.Stat(getFileInfoInput.Path)
	fileInfo := FileInfo{
		Path:   getFileInfoInput.Path,
		Exists: err == nil,
	}

	if err != nil {
		if os.IsNotExist(err) {
			result, marshalErr := json.Marshal(fileInfo)
			if marshalErr != nil {
				return "", fmt.Errorf("failed to marshal result: %w", marshalErr)
			}
			return string(result), nil
		}
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	fileInfo.IsDirectory = info.IsDir()
	fileInfo.Size = info.Size()
	fileInfo.Mode = info.Mode().String()
	fileInfo.ModTime = info.ModTime().Format("2006-01-02 15:04:05")

	// Get line count for text files
	if !info.IsDir() && info.Size() > 0 {
		file, err := os.Open(getFileInfoInput.Path)
		if err == nil {
			defer file.Close()
			scanner := bufio.NewScanner(file)
			lineCount := 0
			for scanner.Scan() {
				lineCount++
			}
			if scanner.Err() == nil {
				fileInfo.LineCount = &lineCount
			}
		}
	}

	result, err := json.Marshal(fileInfo)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(result), nil
}
