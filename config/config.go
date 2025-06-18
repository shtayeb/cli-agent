package config

import (
	"github.com/anthropics/anthropic-sdk-go"
)

// Config holds the application configuration
type Config struct {
	Client *anthropic.Client
}

// NewConfig creates a new configuration instance
func NewConfig() *Config {
	return &Config{
		Client: setupAnthropicClient(),
	}
}

// setupAnthropicClient creates and configures the Anthropic client
func setupAnthropicClient() *anthropic.Client {
	client := anthropic.NewClient()
	return &client
}

// setupUserInput creates a function for reading user input from stdin
// func setupUserInput() func() (string, bool) {
// 	scanner := bufio.NewScanner(os.Stdin)

// 	return func() (string, bool) {
// 		if !scanner.Scan() {
// 			return "", false
// 		}
// 		return scanner.Text(), true
// 	}
// }
