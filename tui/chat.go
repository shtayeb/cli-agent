package tui

// A simple program demonstrating the text area component from the Bubbles
// component library.
import (
	"agent/agent"
	"fmt"
	"net/http"
	"strings"

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
	viewport    viewport.Model
	messages    []string
	textarea    textarea.Model
	senderStyle lipgloss.Style
	err         error
	agent       *agent.Agent
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
		textarea:    ta,
		messages:    []string{},
		viewport:    vp,
		senderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		err:         nil,
		agent:       agentApp,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

type apiErrMsg struct {
	err error
}
type fetchedUserMsg string

func getUser(id int) tea.Cmd {
	return func() tea.Msg {
		res, err := http.Get(fmt.Sprintf("http://api.example.com/users?id=%d", id))
		if err != nil {
			return apiErrMsg{err}
		}
		// ...do unmarshaling and stuff...
		return fetchedUserMsg("Agent: " + res.Status)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case fetchedUserMsg:
		// We caught our message like a Pokémon!
		// From here you could save the output to the model
		// to display it later in your view.
		m.messages = append(m.messages, string("test 000"+msg))

		m.viewport.SetContent(
			lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.messages, "\n")),
		)

		return m, nil
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.textarea.SetWidth(msg.Width)
		m.viewport.Height = msg.Height - m.textarea.Height() - lipgloss.Height(gap)

		if len(m.messages) > 0 {
			// Wrap content before setting it.
			m.viewport.SetContent(
				lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.messages, "\n")),
			)
		}
		m.viewport.GotoBottom()
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			fmt.Println(m.textarea.Value())

			return m, tea.Quit
		case tea.KeyEnter:
			// chatMessage := m.senderStyle.Render("You: ") + m.textarea.Value()

			// m.messages = append(m.messages, chatMessage)

			// m.viewport.SetContent(
			// 	lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.messages, "\n")),
			// )

			m.textarea.Reset()
			m.viewport.GotoBottom()

			return m, getUser(2)

			//NOTE: Send the message to AI in here
			// Return something from here
			// _, err := m.agent.Run(context.TODO(), chatMessage)
			// if err != nil {
			// 	errMsg := fmt.Sprintf("Error: %s\n", err.Error())
			// 	m.messages = append(m.messages, errMsg)
			// } else {
			// 	// Thst will handle the changes to the messages viewport
			// 	m.messages = append(m.messages, "agent responded successfully")
			// }

			// m.viewport.SetContent(
			// 	lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.messages, "\n")),
			// )

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
