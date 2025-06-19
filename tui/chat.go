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
	errMsg error
)

type model struct {
	viewport         viewport.Model
	conversation     []anthropic.MessageParam
	renderedMessages []string
	textarea         textarea.Model
	senderStyle      lipgloss.Style
	err              error
	agent            *agent.Agent
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

type chatMsgResponse struct {
	messages []string
}

func (m *model) Run(ctx context.Context, userInput string) tea.Cmd {
	return func() tea.Msg {
		currentInput := userInput
		hasToolCalls := true
		newMessages := []string{}

		for hasToolCalls {
			if currentInput != "" {
				userMessage := anthropic.NewUserMessage(anthropic.NewTextBlock(userInput))
				m.conversation = append(m.conversation, userMessage)
			}

			hasToolCalls = false // Reset flag
			message, err := m.agent.RunInference(ctx, m.conversation)
			if err != nil {
				newMessages = append(newMessages, fmt.Sprintf("\u001b[93mClaude\u001b[0m: %s\n", err.Error()))
				return chatMsgResponse{messages: newMessages}
			}

			m.conversation = append(m.conversation, message.ToParam())

			toolResults := []anthropic.ContentBlockParamUnion{}

			for _, content := range message.Content {
				switch content.Type {
				case "text":
					newMessages = append(newMessages, fmt.Sprintf("\u001b[93mClaude\u001b[0m: %s\n", content.Text))
				case "tool_use":
					hasToolCalls = true // Set flag when tools are used
					newMessages = append(newMessages, fmt.Sprintf("\u001b[92mtool\u001b[0m: %s(%s)\n", content.Name, content.Input))
					result := m.agent.ExecuteTool(content.ID, content.Name, content.Input)
					toolResults = append(toolResults, result)
				}
			}

			if hasToolCalls {
				currentInput = "" // Clear input for next iteration
				m.conversation = append(m.conversation, anthropic.NewUserMessage(toolResults...))
			}

		}

		return chatMsgResponse{messages: newMessages}
	}
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
	case chatMsgResponse:
		m.renderedMessages = append(m.renderedMessages, msg.messages...)
		m.RenderConversationMessages()

		m.viewport.GotoBottom()

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
