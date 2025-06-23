package tui

import (
	"agent/agent"
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const gap = "\n\n"

type (
	errMsg               error
	streamingTextMsg     string
	streamingCompleteMsg struct{}
)

type ChatMessage struct {
	Content string
	IsUser  bool
}

type model struct {
	viewport                viewport.Model
	conversation            []anthropic.MessageParam
	messages                []ChatMessage
	currentStreamingMessage string
	isStreaming             bool
	streamingChan           chan string
	textarea                textarea.Model
	userStyle               lipgloss.Style
	claudeStyle             lipgloss.Style
	userBubbleStyle         lipgloss.Style
	claudeBubbleStyle       lipgloss.Style
	err                     error
	agent                   *agent.Agent
	width                   int
	height                  int
}

func InitialChatModel(agentApp *agent.Agent) model {
	ta := textarea.New()
	ta.Placeholder = "Type your message here..."
	ta.Prompt = ""
	ta.SetWidth(80)
	ta.SetHeight(4)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(true)

	ta.Focus()

	vp := viewport.New(100, 20)

	// Chat bubble styles - User on right, Claude on left
	userBubbleStyle := lipgloss.NewStyle()
	claudeBubbleStyle := lipgloss.NewStyle()

	userStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#007AFF")).
		Bold(true)

	claudeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF6B35")).
		Bold(true)

	return model{
		textarea:          ta,
		conversation:      []anthropic.MessageParam{},
		messages:          []ChatMessage{},
		viewport:          vp,
		userStyle:         userStyle,
		claudeStyle:       claudeStyle,
		userBubbleStyle:   userBubbleStyle,
		claudeBubbleStyle: claudeBubbleStyle,
		err:               nil,
		agent:             agentApp,
		width:             100,
		height:            25,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m *model) waitForStreamingText() tea.Cmd {
	return func() tea.Msg {
		if m.streamingChan == nil {
			return streamingCompleteMsg{}
		}

		text, ok := <-m.streamingChan
		if !ok {
			return streamingCompleteMsg{}
		}

		return streamingTextMsg(text)
	}
}

func (m *model) Run(ctx context.Context, userInput string) tea.Cmd {
	currentInput := userInput
	m.streamingChan = make(chan string, 100)

	if currentInput != "" {
		userMessage := anthropic.NewUserMessage(anthropic.NewTextBlock(userInput))
		m.conversation = append(m.conversation, userMessage)
	}

	// streaming in a go routine
	go func() {
		defer close(m.streamingChan)

		hasToolCalls := true

		for hasToolCalls {
			hasToolCalls = false // Reset flag

			message, err := m.agent.RunInferenceWithStreaming(ctx, m.conversation, func(text string) {
				m.streamingChan <- text
			})

			if err != nil {
				m.streamingChan <- fmt.Sprintf("Error: %s", err.Error())
				return
			}

			m.conversation = append(m.conversation, message.ToParam())

			// handle tool call
			toolResults := []anthropic.ContentBlockParamUnion{}
			for _, content := range message.Content {
				switch content.Type {
				case "tool_use":
					// Continue the loop: we have tool calls
					hasToolCalls = true

					// Send tool call notification
					m.streamingChan <- fmt.Sprintf("\n🔧 Using tool: %s\n", content.Name)

					result := m.agent.ExecuteTool(content.ID, content.Name, content.Input)
					toolResults = append(toolResults, result)
				}
			}

			if hasToolCalls {
				m.conversation = append(m.conversation, anthropic.NewUserMessage(toolResults...))
			}
		}
	}()

	return m.waitForStreamingText()
}

func (m *model) renderMessages() string {
	var rendered []string

	// Calculate centered width for message alignment
	centeredWidth := min(int(float64(m.width)*0.8), 180)

	// Set the bubble width to ensure text wrapping
	// m.userBubbleStyle = m.userBubbleStyle.Width(centeredWidth)
	m.claudeBubbleStyle = m.claudeBubbleStyle.Width(centeredWidth)

	for _, msg := range m.messages {
		if msg.IsUser {
			// User message - aligned to the right
			userLine := lipgloss.NewStyle().
				Align(lipgloss.Right).
				Width(centeredWidth).
				Render(
					m.userStyle.Render("You") + "\n" +
						m.userBubbleStyle.Render(msg.Content))

			rendered = append(rendered, userLine)
		} else {
			// Claude message - aligned to the left
			claudeLine := m.claudeStyle.Render("Claude") + "\n" + m.claudeBubbleStyle.Render(msg.Content)

			rendered = append(rendered, claudeLine)
		}
	}

	// Add current streaming message if any
	if m.isStreaming && m.currentStreamingMessage != "" {
		claudeLine := m.claudeStyle.Render("Claude") + "\n" + m.claudeBubbleStyle.Render(m.currentStreamingMessage+"▋")

		rendered = append(rendered, claudeLine)
	}

	return strings.Join(rendered, "\n\n")
}

func (m *model) renderWelcomeMessage() string {
	centeredWidth := min(int(float64(m.width)*0.8), 180)

	welcomeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Italic(true).
		Align(lipgloss.Center).
		Width(centeredWidth)

	return welcomeStyle.Render("Welcome to Claude Chat! 🤖\nType a message and press Enter to start chatting.")
}

func (m *model) updateViewport() {
	// Show welcome message when no conversation has started
	content := ""

	if len(m.messages) == 0 && !m.isStreaming {
		content = m.renderWelcomeMessage()
	} else {
		content = m.renderMessages()
	}

	m.viewport.SetContent(content)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case streamingTextMsg:
		if !m.isStreaming {
			m.isStreaming = true
			m.currentStreamingMessage = ""
		}

		// accumulate streaming text
		m.currentStreamingMessage += string(msg)

		m.updateViewport()
		m.viewport.GotoBottom()

		// Continue listening for more streaming updates
		return m, m.waitForStreamingText()

	case streamingCompleteMsg:
		if m.currentStreamingMessage != "" {
			// Add the completed Claude message
			m.messages = append(m.messages, ChatMessage{
				Content: m.currentStreamingMessage,
				IsUser:  false,
			})
		}

		m.isStreaming = false
		m.streamingChan = nil
		m.currentStreamingMessage = ""

		m.updateViewport()
		m.viewport.GotoBottom()

		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate centered dimensions
		centeredWidth := min(int(float64(msg.Width)*0.8), 180)

		m.viewport.Width = centeredWidth
		m.textarea.SetWidth(centeredWidth)

		// Calculate heights
		headerHeight := 3                     // header + blank line
		footerHeight := 1                     // footer
		gapHeight := lipgloss.Height(gap)     // gap between viewport and textarea
		textareaHeight := m.textarea.Height() // textarea

		// Set viewport height accounting for all other elements
		m.viewport.Height = msg.Height - headerHeight - footerHeight - gapHeight - textareaHeight - 2 // extra padding

		// Update bubble styles with new width (100% of centered width)
		maxBubbleWidth := (centeredWidth * 10) / 10
		m.userBubbleStyle = m.userBubbleStyle.MaxWidth(maxBubbleWidth)
		m.claudeBubbleStyle = m.claudeBubbleStyle.MaxWidth(maxBubbleWidth)

		m.updateViewport()
		m.viewport.GotoBottom()

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyCtrlJ:
			value := m.textarea.Value()
			m.textarea.SetValue(value + "\n")
			return m, nil
		case tea.KeyEnter:
			if msg.Alt {
				break
			}

			inputMsg := strings.TrimSpace(m.textarea.Value())
			if inputMsg == "" {
				return m, nil
			}

			// Add user message
			m.messages = append(m.messages, ChatMessage{
				Content: inputMsg,
				IsUser:  true,
			})

			m.updateViewport()
			m.textarea.Reset()
			m.viewport.GotoBottom()

			return m, m.Run(context.TODO(), inputMsg)
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m model) View() string {
	// Calculate centered width (80% of terminal width, max 180 chars)
	centeredWidth := min(int(float64(m.width)*0.8), 180)
	leftPadding := (m.width - centeredWidth) / 2

	header := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 4).
		Width(centeredWidth).
		Align(lipgloss.Center).
		Render("🤖 Coding Agent")

	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Width(centeredWidth).
		Align(lipgloss.Center).
		Render("Press Ctrl+C or Esc to quit • Enter to send message • Ctrl+j insert new line")

	// Center the viewport content
	centeredViewport := lipgloss.NewStyle().
		Width(centeredWidth).
		Render(m.viewport.View())

		// Center the textarea with styling
	centeredTextarea := lipgloss.NewStyle().
		Width(centeredWidth).
		Background(lipgloss.Color("#1e1e1e")).
		Foreground(lipgloss.Color("#ffffff")).
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#404040")).
		Render(m.textarea.View())

	// Create the main content
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		centeredViewport,
		gap,
		centeredTextarea,
		footer,
	)

	// Center everything horizontally
	return lipgloss.NewStyle().
		PaddingLeft(leftPadding).
		Render(content)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
