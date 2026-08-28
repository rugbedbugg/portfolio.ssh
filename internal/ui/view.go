package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	responsiveBreakpoint = 72
	minimumWidth         = 24
	minimumHeight        = 10
	wideNavigationWidth  = 28
)

var sectionLabels = []string{
	"ABOUT",
	"CASE FILES",
	"RESEARCH",
	"DISPATCHES",
	"CONTACT",
}

// render selects the layout appropriate for the current terminal dimensions.
func render(model *Model) string {
	if model.width < minimumWidth || model.height < minimumHeight {
		return renderMinimumSize()
	}
	if model.width >= responsiveBreakpoint {
		return renderWide(model)
	}
	return renderNarrow(model)
}

func renderMinimumSize() string {
	return strings.Join([]string{
		titleStyle.Render("OXIDE"),
		errorStyle.Render("TERMINAL TOO SMALL"),
		primaryCopyStyle.Render("Resize to continue."),
		promptStyle.Render("q EXIT"),
	}, "\n")
}

// renderWide places a bounded navigation pane beside a flexible dossier pane.
func renderWide(model *Model) string {
	totalWidth := maxInt(1, model.width)
	navigationWidth := minInt(wideNavigationWidth, maxInt(1, totalWidth/3))
	dossierWidth := maxInt(1, totalWidth-navigationWidth-1)

	navigation := renderPanel(renderNavigation(model), navigationWidth)
	dossier := renderPanel(renderDossier(model, panelContentWidth(dossierWidth)), dossierWidth)
	body := lipgloss.JoinHorizontal(lipgloss.Top, navigation, " ", dossier)
	return body + "\n" + renderFooter(model)
}

// renderNarrow shows either the section index or the active dossier.
func renderNarrow(model *Model) string {
	width := maxInt(1, model.width)
	var body string
	if model.pane == PaneIndex {
		body = renderNavigation(model)
	} else {
		body = secondaryCopyStyle.Render("ESC BACK") + "\n\n" +
			renderDossier(model, panelContentWidth(width))
	}
	return renderPanel(body, width) + "\n" + renderFooter(model)
}

func renderPanel(content string, totalWidth int) string {
	style := borderStyle
	contentWidth := maxInt(1, totalWidth-style.GetHorizontalFrameSize())
	return style.Width(contentWidth).Render(content)
}

func panelContentWidth(totalWidth int) int {
	return maxInt(1, totalWidth-borderStyle.GetHorizontalFrameSize())
}

func renderNavigation(model *Model) string {
	selectedSection := int(activeSection(model))
	lines := []string{
		titleStyle.Render("OXIDE // DOSSIER"),
		secondaryCopyStyle.Render("SECTION INDEX"),
		"",
	}
	for index, label := range sectionLabels {
		prefix := "  "
		style := primaryCopyStyle
		if index == selectedSection {
			prefix = "> "
			style = selectionStyle
		}
		lines = append(lines, style.Render(prefix+label))
	}
	return strings.Join(lines, "\n")
}

func activeSection(model *Model) Section {
	if model.pane == PaneIndex && model.selected >= 0 && model.selected < len(sections) {
		return sections[model.selected]
	}
	if int(model.section) >= 0 && int(model.section) < len(sectionLabels) {
		return model.section
	}
	return SectionAbout
}

func renderDossier(model *Model, width int) string {
	section := activeSection(model)
	lines := []string{titleStyle.Render(sectionLabels[section])}

	switch section {
	case SectionAbout:
		lines = append(lines, renderAbout(model, width)...)
	case SectionProjects:
		lines = append(lines, renderProjects(model, width)...)
	case SectionResearch:
		lines = append(lines, renderResearch(model, width)...)
	case SectionDispatches:
		lines = append(lines, renderDispatches(model, width)...)
	case SectionContact:
		lines = append(lines, renderContact(model, width)...)
	}

	if status := renderStatus(model); status != "" {
		lines = append(lines, "", status)
	}
	return strings.Join(lines, "\n")
}

func renderAbout(model *Model, width int) []string {
	profile := model.data.Profile
	lines := []string{
		primaryCopyStyle.Render(profile.Name),
		secondaryCopyStyle.Render(wrapProse(profile.Aliases, width)),
		terminalStateStyle.Render(wrapProse(profile.Tagline, width)),
	}
	for _, paragraph := range profile.Biography {
		lines = append(lines, "", primaryCopyStyle.Render(wrapProse(paragraph, width)))
	}
	return lines
}

func renderProjects(model *Model, width int) []string {
	if len(model.data.Projects) == 0 {
		return []string{secondaryCopyStyle.Render("NO CASE FILES")}
	}
	selected := safeIndex(model.selected, len(model.data.Projects))
	lines := make([]string, 0, len(model.data.Projects)+8)
	if model.pane != PaneRecord {
		for index, project := range model.data.Projects {
			lines = append(lines, renderRecordChoice(index == selected, project.Title))
		}
		lines = append(lines, "")
	}
	project := model.data.Projects[selected]
	return append(lines,
		secondaryCopyStyle.Render("CASE // "+strings.ToUpper(project.ID)),
		primaryCopyStyle.Render(project.Title),
		primaryCopyStyle.Render(wrapProse(project.Summary, width)),
		secondaryCopyStyle.Render("TAGS // "+strings.Join(project.Tags, " · ")),
		primaryCopyStyle.Render(project.URL),
	)
}

func renderResearch(model *Model, width int) []string {
	if len(model.data.Publications) == 0 {
		return []string{secondaryCopyStyle.Render("NO RESEARCH RECORDS")}
	}
	selected := safeIndex(model.selected, len(model.data.Publications))
	lines := make([]string, 0, len(model.data.Publications)+9)
	if model.pane != PaneRecord {
		for index, publication := range model.data.Publications {
			lines = append(lines, renderRecordChoice(index == selected, publication.Title))
		}
		lines = append(lines, "")
	}
	publication := model.data.Publications[selected]
	return append(lines,
		secondaryCopyStyle.Render("RECORD // "+strings.ToUpper(publication.ID)),
		primaryCopyStyle.Render(wrapProse(publication.Title, width)),
		terminalStateStyle.Render(publication.Venue),
		primaryCopyStyle.Render(wrapProse("CONTRIBUTION // "+publication.Contribution, width)),
		secondaryCopyStyle.Render(wrapProse("AUTHORS // "+strings.Join(publication.Authors, ", "), width)),
		primaryCopyStyle.Render(publication.URL),
	)
}

func renderDispatches(model *Model, width int) []string {
	if len(model.data.Dispatches) == 0 {
		return []string{secondaryCopyStyle.Render("NO DISPATCHES")}
	}
	selected := safeIndex(model.selected, len(model.data.Dispatches))
	lines := make([]string, 0, len(model.data.Dispatches)+9)
	if model.pane != PaneRecord {
		for index, dispatch := range model.data.Dispatches {
			lines = append(lines, renderRecordChoice(index == selected, dispatch.Title))
		}
		lines = append(lines, "")
	}
	dispatch := model.data.Dispatches[selected]
	return append(lines,
		secondaryCopyStyle.Render(fmt.Sprintf("%s // %s", dispatch.Date, dispatch.Topic)),
		primaryCopyStyle.Render(wrapProse(dispatch.Title, width)),
		primaryCopyStyle.Render(wrapProse(dispatch.Excerpt, width)),
		primaryCopyStyle.Render(dispatch.URL),
	)
}

func renderContact(model *Model, width int) []string {
	if len(model.data.Links) == 0 {
		return []string{secondaryCopyStyle.Render("NO OPEN CHANNELS")}
	}
	selected := safeIndex(model.selected, len(model.data.Links))
	lines := make([]string, 0, len(model.data.Links)+7)
	if model.pane != PaneRecord {
		for index, link := range model.data.Links {
			lines = append(lines, renderRecordChoice(index == selected, link.Label))
		}
		lines = append(lines, "")
	}
	link := model.data.Links[selected]
	return append(lines,
		secondaryCopyStyle.Render("CHANNEL // "+strings.ToUpper(link.ID)),
		primaryCopyStyle.Render(link.Label),
		primaryCopyStyle.Render(wrapProse(link.Description, width)),
		primaryCopyStyle.Render(link.URL),
	)
}

func renderRecordChoice(selected bool, label string) string {
	if selected {
		return selectionStyle.Render("> " + label)
	}
	return primaryCopyStyle.Render("  " + label)
}

func renderStatus(model *Model) string {
	if model.status == "" || model.status == model.recordDescription() {
		return ""
	}
	lower := strings.ToLower(model.status)
	if strings.Contains(lower, "unknown") || strings.Contains(lower, "ambiguous") || strings.Contains(lower, "error") {
		return errorStyle.Render("ERROR // " + model.status)
	}
	return successStyle.Render("STATUS // " + model.status)
}

func renderFooter(model *Model) string {
	if model.focus == FocusCommand {
		label := promptStyle.Render(": COMMAND")
		available := maxInt(1, model.width-lipgloss.Width(label)-1)
		return label + " " + ansi.Truncate(model.commandInput.View(), available, "")
	}

	full := strings.Join([]string{
		terminalStateStyle.Render("↑/k ↓/j MOVE"),
		terminalStateStyle.Render("ENTER OPEN"),
		secondaryCopyStyle.Render("ESC BACK"),
		promptStyle.Render(": COMMAND"),
		secondaryCopyStyle.Render("q EXIT"),
	}, "  ")
	if lipgloss.Width(full) <= model.width {
		return full
	}

	compact := strings.Join([]string{
		terminalStateStyle.Render("j/k MOVE"),
		terminalStateStyle.Render("ENTER OPEN"),
		secondaryCopyStyle.Render("ESC BACK"),
		promptStyle.Render(": COMMAND"),
		secondaryCopyStyle.Render("q EXIT"),
	}, "  ")
	if lipgloss.Width(compact) <= model.width {
		return compact
	}

	contextHint := "j/k"
	if model.pane != PaneIndex {
		contextHint = "ESC"
	}
	return terminalStateStyle.Render(contextHint) + " " +
		promptStyle.Render(": COMMAND") + " " +
		secondaryCopyStyle.Render("q EXIT")
}

func wrapProse(value string, width int) string {
	return ansi.Wordwrap(value, maxInt(1, width), "")
}

func safeIndex(index, length int) int {
	if length <= 0 || index < 0 {
		return 0
	}
	if index >= length {
		return length - 1
	}
	return index
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
