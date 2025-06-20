package tui

// A simple program demonstrating the text area component from the Bubbles
// component library.
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

type model struct {
	viewport                viewport.Model
	conversation            []anthropic.MessageParam
	renderedMessages        []string
	currentStreamingMessage string
	isStreaming             bool
	streamingChan           chan string
	textarea                textarea.Model
	senderStyle             lipgloss.Style
	err                     error
	agent                   *agent.Agent
}

func InitialChatModel(agentApp *agent.Agent) model {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 280

	ta.SetWidth(30)
	ta.SetHeight(3)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	ta.ShowLineNumbers = false

	vp := viewport.New(30, 5)
	vp.SetContent(`Welcome to the chat room!
Type a message and press Enter to send.`)

	ta.KeyMap.InsertNewline.SetEnabled(false)

	return model{
		textarea:     ta,
		conversation: []anthropic.MessageParam{},
		viewport:     vp,
		senderStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		err:          nil,
		agent:        agentApp,
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

	// channel for streaming updates

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
				// case "text":
				// newMessages = append(newMessages, fmt.Sprintf("\u001b[93mClaude\u001b[0m: %s\n", content.Text))
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

func (m *model) RenderConversationMessages() {
	m.viewport.SetContent(
		lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.renderedMessages, "\n")),
	)
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
		// accumulate streaming text
		m.currentStreamingMessage += string(msg)

		// Update or add Claude message
		claudeMsg := fmt.Sprintf("\u001b[93mClaude\u001b[0m: %s", m.currentStreamingMessage)

		if len(m.renderedMessages) > 0 && strings.HasPrefix(m.renderedMessages[len(m.renderedMessages)-1], "\u001b[93mClaude\u001b[0m:") {
			// Update existing message
			m.renderedMessages[len(m.renderedMessages)-1] = claudeMsg
		} else {
			// Add new message
			m.renderedMessages = append(m.renderedMessages, claudeMsg)
		}

		m.RenderConversationMessages()
		m.viewport.GotoBottom()

		// Continue listening for more streaming updates
		return m, m.waitForStreamingText()

	case streamingCompleteMsg:
		m.isStreaming = false
		m.streamingChan = nil
		m.currentStreamingMessage = ""

		return m, nil

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.textarea.SetWidth(msg.Width)
		m.viewport.Height = msg.Height - m.textarea.Height() - lipgloss.Height(gap)

		if len(m.conversation) > 0 {
			m.RenderConversationMessages()
		}

		m.viewport.GotoBottom()
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			fmt.Println(m.textarea.Value())

			return m, tea.Quit
		case tea.KeyEnter:
			inputMsg := m.textarea.Value()
			chatMessage := m.senderStyle.Render("You: ") + inputMsg
			m.renderedMessages = append(m.renderedMessages, chatMessage)

			m.RenderConversationMessages()

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
	return fmt.Sprintf(
		"%s%s%s",
		m.viewport.View(),
		gap,
		m.textarea.View(),
	)
}
