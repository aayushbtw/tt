package main

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const timeInSeconds = 5
const timeout = time.Second * timeInSeconds

var (
	words = "hi only few people can read this"

	textPrimaryStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "#909090",
		Dark:  "#626262",
	})

	textSecondaryStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "#B2B2B2",
		Dark:  "#4A4A4A",
	})
)

type model struct {
	timer     timer.Model
	help      help.Model
	textInput textinput.Model
	cursor    cursor.Model
	width     int
	height    int
	keymap    keymap

	wpm      float64
	accuracy float64
	started  bool
	ended    bool
}

type keymap struct {
	start key.Binding
	reset key.Binding
	quit  key.Binding
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.SetWindowTitle("tt."),
		textinput.Blink,
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width == 0 && msg.Height == 0 {
			return m, nil
		} else {
			m.width = msg.Width
			m.height = msg.Height
			return m, nil
		}

	case timer.TickMsg:
		m.timer, cmd = m.timer.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.quit):
			return m, tea.Quit
		case key.Matches(msg, m.keymap.start):
			m.timer = timer.NewWithInterval(timeout, time.Millisecond)
			m.keymap.start.SetEnabled(false)
			m.keymap.reset.SetEnabled(true)
			m.textInput.Reset()
			m.cursor.Focus()
			m.started = true
			return m, tea.Batch(m.timer.Init(), m.cursor.BlinkCmd())
		case key.Matches(msg, m.keymap.reset):
			m.timer = timer.NewWithInterval(timeout, time.Millisecond)
			m.keymap.start.SetEnabled(true)
			m.keymap.reset.SetEnabled(false)
			m.textInput.Reset()
			m.cursor.Blur()
			m.started = false
			m.ended = false
			return m, nil
		}

	case timer.TimeoutMsg:
		expectedWords := strings.Fields(words)
		typedWords := strings.Fields(m.textInput.Value())
		correctChars := 0
		totalChars := 0
		correctWords := 0

		for i := 0; i < len(typedWords); i++ {
			if i < len(expectedWords) {
				refWord := expectedWords[i]
				typedWord := typedWords[i]

				for j := 0; j < len(typedWord); j++ {
					if j < len(refWord) && typedWord[j] == refWord[j] {
						correctChars++
					}
					if j < len(refWord) {
						totalChars++
					}
				}

				if typedWord == refWord {
					correctWords++
				}
			}
		}

		m.wpm = float64(correctWords) / (timeInSeconds / 60.0)
		m.accuracy = (float64(correctChars) / float64(totalChars)) * 100
		m.textInput.Reset()
		m.cursor.Blur()
		m.started = false
		m.ended = true

		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)

	m.cursor, cmd = m.cursor.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) helpView() string {
	return m.help.ShortHelpView([]key.Binding{
		m.keymap.start,
		m.keymap.reset,
		m.keymap.quit,
	})
}

func (m model) View() string {
	defaultStyle := lipgloss.NewStyle().Width(m.width).Margin(0, m.width/10)
	contentStyle := defaultStyle.Height(10)

	header := defaultStyle.
		Height(3).
		Align(lipgloss.Left, lipgloss.Top).
		Render(
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#eee")).Render("tt"),
			"\u2014",
			textPrimaryStyle.Bold(true).Render("A minimalist CLI typing speed test."),
		)

	content := contentStyle.Render()

	// Typing Area
	if m.started {
		input := m.textInput.Value()
		var typedChars string

		inputWords := strings.Fields(input)
		expectedWords := strings.Fields(words)

		for i, word := range inputWords {
			if i < len(expectedWords) {
				if word == expectedWords[i] {
					// Correct word
					typedChars += lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Render(word) + " "
				} else {
					// Incorrect word
					typedChars += lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(word) + " "
				}
			} else {
				typedChars += lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(word) + " "
			}
		}

		cursorChar := string(words[len(input)])
		m.cursor.SetChar(cursorChar)
		cursorView := m.cursor.View()
		typedCharsWithCursor := typedChars + cursorView

		remainingPlaceholder := ""
		if len(inputWords) < len(expectedWords) {
			remainingPlaceholder = textSecondaryStyle.Render(strings.Join(expectedWords[len(inputWords):], " "))
		}

		// Combine everything together
		combinedInput := lipgloss.JoinHorizontal(
			lipgloss.Left,
			typedCharsWithCursor,
			remainingPlaceholder,
		)

		typeLayout := lipgloss.JoinVertical(
			lipgloss.Top,
			m.timer.View(),
			combinedInput,
		)
		content = contentStyle.Render(typeLayout)
	}

	// Results
	if m.ended {
		keyStyle := textPrimaryStyle.Bold(true)

		wpmStr := strconv.FormatFloat(m.wpm, 'f', 2, 64)
		accuracyStr := strconv.FormatFloat(m.accuracy, 'f', 2, 64)

		resultLayout := lipgloss.JoinVertical(
			lipgloss.Top,
			lipgloss.NewStyle().Render(keyStyle.Render("WPM:"), textSecondaryStyle.Render(wpmStr)),
			lipgloss.NewStyle().Render(keyStyle.Render("Accuracy:"), textSecondaryStyle.Render(accuracyStr+"%")),
		)

		content = contentStyle.Render(resultLayout)
	}

	footer := lipgloss.NewStyle().
		Width(m.width).
		Height(3).
		Align(lipgloss.Center, lipgloss.Bottom).
		Render(m.helpView())

	// Combine header, content, and footer
	layout := lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		content,
		footer,
	)

	finalView := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Left, lipgloss.Center).
		Render(layout)

	return finalView
}

func main() {
	ti := textinput.New()
	ti.Placeholder = words
	ti.PlaceholderStyle = textSecondaryStyle
	ti.TextStyle = textPrimaryStyle
	// ti.Cursor.Style = textPrimaryStyle
	// ti.Cursor.SetMode(cursor.CursorStatic)
	ti.Prompt = ""
	ti.Focus()

	ti.Cursor.Style = textPrimaryStyle
	cursorModel := cursor.New()
	cursorModel.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	cursorModel.BlinkSpeed = time.Millisecond * 530

	m := model{
		timer:     timer.NewWithInterval(timeout, time.Millisecond),
		textInput: ti,
		help:      help.New(),
		cursor:    cursorModel,
		keymap: keymap{
			start: key.NewBinding(
				key.WithKeys("."),
				key.WithHelp(".", "start"),
			),
			reset: key.NewBinding(
				key.WithKeys("tab"),
				key.WithHelp("tab", "reset"),
			),
			quit: key.NewBinding(
				key.WithKeys("esc", "ctrl+c"),
				key.WithHelp("esc", "quit"),
			),
		},
	}

	m.keymap.reset.SetEnabled(false)

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		log.Fatal(err)
	}
}
