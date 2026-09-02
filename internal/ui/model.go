// Package ui owns the interactive Bubble Tea session state for the portfolio.
package ui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/rugbedbugg/portfolio.ssh/internal/command"
	"github.com/rugbedbugg/portfolio.ssh/internal/content"
)

// Section identifies a top-level portfolio section.
type Section uint8

const (
	SectionAbout Section = iota
	SectionProjects
	SectionResearch
	SectionContact
)

// Pane identifies the content currently visible within the session.
type Pane uint8

const (
	PaneIndex Pane = iota
	PaneSection
)

// Focus identifies which part of the session receives keyboard input.
type Focus uint8

const (
	FocusNavigation Focus = iota
	FocusCommand
)

var sections = []Section{
	SectionAbout,
	SectionProjects,
	SectionResearch,
	SectionContact,
}

const helpStatus = "commands: help, about/whoami, projects/ls, project <id>, research, contact, open <id>, clear, exit"

// Model holds the portfolio data and a single interactive session's state.
type Model struct {
	data          content.Portfolio
	width, height int
	section       Section
	pane          Pane
	focus         Focus
	selected      int
	commandInput  textinput.Model
	history       []string
	historyIndex  int
	status        string
}

// New creates a fresh portfolio session at the top-level section index.
func New(data content.Portfolio, width, height int) *Model {
	input := textinput.New()
	input.Prompt = ": "
	styles := input.Styles()
	styles.Focused.Prompt = promptStyle
	input.SetStyles(styles)
	input.SetWidth(commandInputWidth(width))

	return &Model{
		data:         data,
		width:        width,
		height:       height,
		section:      SectionAbout,
		pane:         PaneIndex,
		focus:        FocusNavigation,
		commandInput: input,
		historyIndex: -1,
	}
}

// Init requires no startup command.
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update applies one Bubble Tea message to the session.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if resize, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = resize.Width
		m.height = resize.Height
		m.commandInput.SetWidth(commandInputWidth(resize.Width))
		return m, nil
	}

	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	if m.focus == FocusCommand {
		return m.updateCommandInput(key)
	}

	switch key.String() {
	case "j", "down":
		m.move(1)
	case "k", "up":
		m.move(-1)
	case "a":
		m.openSectionByCommand(SectionAbout)
	case "p":
		m.openSectionByCommand(SectionProjects)
	case "r":
		m.openSectionByCommand(SectionResearch)
	case "c":
		m.openSectionByCommand(SectionContact)
	case "enter":
		return m, m.openSelected()
	case "esc":
		m.back()
	case ":":
		m.focus = FocusCommand
		return m, m.commandInput.Focus()
	case "?":
		m.status = helpStatus
	case "q":
		return m, func() tea.Msg { return tea.Quit() }
	}

	return m, nil
}

func (m *Model) updateCommandInput(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.commandInput.Blur()
		m.focus = FocusNavigation
		return m, nil
	case "enter":
		input := strings.TrimSpace(m.commandInput.Value())
		m.commandInput.Reset()
		m.commandInput.Blur()
		m.focus = FocusNavigation
		return m.executeCommand(input)
	case "tab":
		completion := command.Complete(m.commandInput.Value(), m.data)
		m.commandInput.SetValue(completion.Input)
		if len(completion.Candidates) > 1 {
			m.status = "matches: " + strings.Join(completion.Candidates, ", ")
		}
		return m, nil
	case "up":
		m.recallHistory(-1)
		return m, nil
	case "down":
		m.recallHistory(1)
		return m, nil
	}

	input, cmd := m.commandInput.Update(key)
	m.commandInput = input
	return m, cmd
}

func (m *Model) move(delta int) {
	count := m.selectionCount()
	if count == 0 {
		m.selected = 0
		return
	}
	m.selected = (m.selected + delta + count) % count
}

func (m *Model) selectionCount() int {
	if m.pane == PaneIndex {
		return len(sections)
	}
	if m.pane == PaneSection {
		return m.recordCount()
	}
	return 0
}

// openSelected enters a section from the index, or hands the selected record's
// URL to the visitor's clipboard. A server cannot open a browser on the far end
// of an SSH session, so the link itself is the deliverable.
func (m *Model) openSelected() tea.Cmd {
	switch m.pane {
	case PaneIndex:
		m.section = sections[m.selected]
		m.pane = PaneSection
		m.selected = 0
	case PaneSection:
		return m.copySelectedURL()
	}
	return nil
}

func (m *Model) copySelectedURL() tea.Cmd {
	url := m.selectedURL()
	if url == "" {
		return nil
	}
	// Worded as an attempt: OSC 52 is refused by some terminals and by tmux
	// without set-clipboard, and the session cannot observe the outcome.
	m.status = "sent to clipboard: " + url
	return tea.SetClipboard(url)
}

func (m *Model) recordCount() int {
	switch m.section {
	case SectionProjects:
		return len(m.data.Projects)
	case SectionResearch:
		return len(m.data.Publications)
	case SectionContact:
		return len(m.data.Links)
	default:
		return 0
	}
}

func (m *Model) back() {
	m.commandInput.Blur()
	m.focus = FocusNavigation
	m.status = ""
	m.pane = PaneIndex
	m.selected = sectionIndex(m.section)
}

func sectionIndex(section Section) int {
	for index, candidate := range sections {
		if candidate == section {
			return index
		}
	}
	return 0
}

func (m *Model) executeCommand(input string) (tea.Model, tea.Cmd) {
	if input == "" {
		return m, nil
	}
	m.history = append(m.history, input)
	m.historyIndex = -1

	result := command.Parse(input, m.data)
	if result.Err != "" {
		m.status = result.Err
		if len(result.Suggestions) > 0 {
			m.status += " suggestions: " + strings.Join(result.Suggestions, ", ")
		}
		return m, nil
	}

	switch result.Kind {
	case command.Help:
		m.status = helpStatus
	case command.About:
		m.openSectionByCommand(SectionAbout)
		m.status = m.data.Profile.Name + " — " + m.data.Profile.Tagline
	case command.Projects:
		m.openSectionByCommand(SectionProjects)
	case command.Project:
		m.openProject(result.Target)
	case command.Research:
		m.openSectionByCommand(SectionResearch)
	case command.Contact:
		m.openSectionByCommand(SectionContact)
	case command.Open:
		m.openLink(result.Target)
	case command.Clear:
		m.status = ""
		m.history = nil
		m.historyIndex = -1
	case command.Exit:
		return m, func() tea.Msg { return tea.Quit() }
	case command.Unknown:
		m.status = "unknown command; try `help` for the available commands."
	}
	return m, nil
}

func (m *Model) openSectionByCommand(section Section) {
	m.section = section
	m.pane = PaneSection
	m.selected = 0
}

func (m *Model) openProject(id string) {
	for index, project := range m.data.Projects {
		if project.ID == id {
			m.section = SectionProjects
			m.pane = PaneSection
			m.selected = index
			m.status = project.Title + " — " + project.URL
			return
		}
	}
	m.status = "unknown project; try `projects` to inspect available records."
}

func (m *Model) openLink(id string) {
	for index, link := range m.data.Links {
		if link.ID == id {
			m.section = SectionContact
			m.pane = PaneSection
			m.selected = index
			m.status = link.Label + " — " + link.URL
			return
		}
	}
	m.status = "unknown contact link; try `contact` to inspect available records."
}

func (m *Model) recallHistory(delta int) {
	if len(m.history) == 0 {
		return
	}
	if delta < 0 {
		if m.historyIndex < 0 {
			m.historyIndex = len(m.history) - 1
		} else if m.historyIndex > 0 {
			m.historyIndex--
		}
	} else if m.historyIndex >= 0 {
		if m.historyIndex == len(m.history)-1 {
			m.historyIndex = -1
			m.commandInput.Reset()
			return
		}
		m.historyIndex++
	}
	m.commandInput.SetValue(m.history[m.historyIndex])
}

// selectedURL is the destination for the currently selected record, or an
// empty string for sections whose records have none.
func (m *Model) selectedURL() string {
	switch m.section {
	case SectionProjects:
		if m.selected < len(m.data.Projects) {
			return m.data.Projects[m.selected].URL
		}
	case SectionResearch:
		if m.selected < len(m.data.Publications) {
			return m.data.Publications[m.selected].URL
		}
	case SectionContact:
		if m.selected < len(m.data.Links) {
			return m.data.Links[m.selected].URL
		}
	}
	return ""
}

// View renders the portfolio in an alternate screen so updates never leave
// partial frames in the visitor's terminal scrollback.
func (m *Model) View() tea.View {
	view := tea.NewView(render(m))
	view.AltScreen = true
	return view
}
