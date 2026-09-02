package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	responsiveBreakpoint   = 50
	minimumWidth           = 24
	minimumHeight          = 10
	maximumContainerWidth  = 80
	maximumContainerHeight = 30
	wideListWidth          = 20
)

var headerLabels = []string{
	"Partha P.G.",
	"p projects",
	"r research",
	"c contact",
}

// render mirrors Terminal Shop's fixed, centered canvas while preserving a
// compact fallback for small terminals.
func render(model *Model) string {
	width := maxInt(1, model.width)
	height := maxInt(1, model.height)
	if width < minimumWidth || height < minimumHeight {
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, renderMinimumSize())
	}

	containerWidth := minInt(width, maximumContainerWidth)
	containerHeight := minInt(height, maximumContainerHeight)
	content := renderContainer(model, containerWidth, containerHeight)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}

func renderMinimumSize() string {
	return strings.Join([]string{
		titleStyle.Render("terminal too small"),
		primaryCopyStyle.Render("resize to continue"),
		promptStyle.Render("q quit"),
	}, "\n")
}

func renderContainer(model *Model, width, height int) string {
	header := renderHeader(width)
	footer := renderFooter(model)
	bodyHeight := maxInt(1, height-lipgloss.Height(header)-lipgloss.Height(footer)-2)
	bodyWidth := maxInt(1, width-2)
	body := renderBody(model, bodyWidth)
	body = fitBlock(body, bodyWidth, bodyHeight)
	body = lipgloss.PlaceHorizontal(width, lipgloss.Center, body)
	return strings.Join([]string{header, "", body, "", footer}, "\n")
}

func renderHeader(containerWidth int) string {
	tableWidth := maxInt(1, containerWidth-2)
	if tableWidth < responsiveBreakpoint {
		label := "Partha P.G.  p projects  r research  c contact"
		return lipgloss.PlaceHorizontal(containerWidth, lipgloss.Center, ansi.Truncate(label, tableWidth, ""))
	}

	innerWidth := tableWidth - len(headerLabels) - 1
	cellWidths := distributeWidth(innerWidth, len(headerLabels))
	top := make([]string, 0, len(headerLabels))
	middle := make([]string, 0, len(headerLabels))
	bottom := make([]string, 0, len(headerLabels))
	for index, label := range headerLabels {
		top = append(top, strings.Repeat("─", cellWidths[index]))
		middle = append(middle, centerText(label, cellWidths[index]))
		bottom = append(bottom, strings.Repeat("─", cellWidths[index]))
	}

	table := strings.Join([]string{
		"┌" + strings.Join(top, "┬") + "┐",
		"│" + strings.Join(middle, "│") + "│",
		"└" + strings.Join(bottom, "┴") + "┘",
	}, "\n")
	return lipgloss.PlaceHorizontal(containerWidth, lipgloss.Center, table)
}

func distributeWidth(total, count int) []int {
	widths := make([]int, count)
	if count == 0 {
		return widths
	}
	base := total / count
	remainder := total % count
	for index := range widths {
		widths[index] = base
		if index < remainder {
			widths[index]++
		}
	}
	return widths
}

func centerText(value string, width int) string {
	value = ansi.Truncate(value, maxInt(1, width), "")
	padding := maxInt(0, width-lipgloss.Width(value))
	left := padding / 2
	return strings.Repeat(" ", left) + value + strings.Repeat(" ", padding-left)
}

func renderBody(model *Model, width int) string {
	section := activeSection(model)
	var lines []string
	switch section {
	case SectionAbout:
		lines = renderAbout(model, width)
	case SectionProjects:
		lines = renderProjects(model, width)
	case SectionResearch:
		lines = renderResearch(model, width)
	case SectionDispatches:
		lines = renderDispatches(model, width)
	case SectionContact:
		lines = renderContact(model, width)
	}
	if status := renderStatus(model); status != "" {
		lines = append(lines, "", status)
	}
	return strings.Join(lines, "\n")
}

func activeSection(model *Model) Section {
	if model.pane == PaneIndex && model.selected >= 0 && model.selected < len(sections) {
		return sections[model.selected]
	}
	return model.section
}

func renderAbout(model *Model, width int) []string {
	profile := model.data.Profile
	lines := []string{
		titleStyle.Render(profile.Name),
		terminalStateStyle.Render(profile.Tagline),
	}
	for _, paragraph := range profile.Biography {
		lines = append(lines, "", primaryCopyStyle.Render(wrapProse(paragraph, width)))
	}
	return lines
}

func renderProjects(model *Model, width int) []string {
	if len(model.data.Projects) == 0 {
		return []string{"no projects"}
	}
	selected := safeIndex(model.selected, len(model.data.Projects))
	choices := make([]string, 0, len(model.data.Projects))
	for index, project := range model.data.Projects {
		choices = append(choices, renderRecordChoice(index == selected, project.Title))
	}
	project := model.data.Projects[selected]
	detail := []string{
		titleStyle.Render(project.Title),
		primaryCopyStyle.Render(wrapProse(project.Summary, detailWidth(width))),
		secondaryCopyStyle.Render(strings.Join(project.Tags, " · ")),
		primaryCopyStyle.Render(project.URL),
	}
	return renderCollection(model, width, choices, detail)
}

func renderResearch(model *Model, width int) []string {
	if len(model.data.Publications) == 0 {
		return []string{"no research"}
	}
	selected := safeIndex(model.selected, len(model.data.Publications))
	choices := make([]string, 0, len(model.data.Publications))
	for index, publication := range model.data.Publications {
		choices = append(choices, renderRecordChoice(index == selected, publication.Title))
	}
	publication := model.data.Publications[selected]
	detail := []string{
		titleStyle.Render(wrapProse(publication.Title, detailWidth(width))),
		terminalStateStyle.Render(publication.Venue),
		primaryCopyStyle.Render(wrapProse(publication.Contribution, detailWidth(width))),
		secondaryCopyStyle.Render(wrapProse(strings.Join(publication.Authors, ", "), detailWidth(width))),
		primaryCopyStyle.Render(publication.URL),
	}
	return renderCollection(model, width, choices, detail)
}

func renderDispatches(model *Model, width int) []string {
	if len(model.data.Dispatches) == 0 {
		return []string{"no dispatches"}
	}
	selected := safeIndex(model.selected, len(model.data.Dispatches))
	choices := make([]string, 0, len(model.data.Dispatches))
	for index, dispatch := range model.data.Dispatches {
		choices = append(choices, renderRecordChoice(index == selected, dispatch.Title))
	}
	dispatch := model.data.Dispatches[selected]
	detail := []string{
		titleStyle.Render(wrapProse(dispatch.Title, detailWidth(width))),
		secondaryCopyStyle.Render(fmt.Sprintf("%s · %s", dispatch.Date, dispatch.Topic)),
		primaryCopyStyle.Render(wrapProse(dispatch.Excerpt, detailWidth(width))),
		primaryCopyStyle.Render(dispatch.URL),
	}
	return renderCollection(model, width, choices, detail)
}

func renderContact(model *Model, width int) []string {
	if len(model.data.Links) == 0 {
		return []string{"no contact links"}
	}
	selected := safeIndex(model.selected, len(model.data.Links))
	choices := make([]string, 0, len(model.data.Links))
	for index, link := range model.data.Links {
		choices = append(choices, renderRecordChoice(index == selected, link.Label))
	}
	link := model.data.Links[selected]
	detail := []string{
		titleStyle.Render(link.Label),
		primaryCopyStyle.Render(link.URL),
	}
	return renderCollection(model, width, choices, detail)
}

func renderCollection(model *Model, width int, choices, detail []string) []string {
	if model.pane == PaneRecord {
		return detail
	}
	if width < responsiveBreakpoint {
		return append(append(choices, ""), detail...)
	}

	listWidth := minInt(wideListWidth, maxInt(1, width/3))
	gapWidth := 4
	menu := renderChoiceList(choices, listWidth)
	right := strings.Join(detail, "\n")
	menu = fitBlock(menu, listWidth, maxInt(lipgloss.Height(menu), lipgloss.Height(right)))
	columns := lipgloss.JoinHorizontal(
		lipgloss.Top,
		menu,
		strings.Repeat(" ", gapWidth),
		right,
	)
	return strings.Split(columns, "\n")
}

func renderChoiceList(choices []string, width int) string {
	lines := make([]string, 0, len(choices))
	for _, choice := range choices {
		lines = append(lines, ansi.Truncate(choice, maxInt(1, width), "…"))
	}
	return strings.Join(lines, "\n")
}

func detailWidth(width int) int {
	if width < responsiveBreakpoint {
		return width
	}
	return maxInt(1, width-minInt(wideListWidth, width/3)-4)
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
		return errorStyle.Render(model.status)
	}
	return successStyle.Render(model.status)
}

func renderFooter(model *Model) string {
	containerWidth := minInt(maxInt(1, model.width), maximumContainerWidth)
	width := maxInt(1, containerWidth-2)
	var hints string
	if model.focus == FocusCommand {
		hints = renderCommandFooter(model, width)
	} else {
		switch {
		case model.pane == PaneIndex:
			hints = "p projects   r research   c contact   q quit"
		case model.pane == PaneRecord || activeSection(model) == SectionAbout:
			hints = "esc back   q quit"
		default:
			hints = "↑/↓ select   enter open   esc back   q quit"
		}
	}
	hints = ansi.Truncate(hints, width, "")
	rule := strings.Repeat("─", width)
	return lipgloss.PlaceHorizontal(containerWidth, lipgloss.Center, rule) + "\n" +
		lipgloss.PlaceHorizontal(containerWidth, lipgloss.Center, hints)
}

func renderCommandFooter(model *Model, width int) string {
	input := model.commandInput.View()
	hints := "tab complete   ↑/↓ history   esc cancel   enter run"
	if available := width - lipgloss.Width(hints) - 3; available > 2 {
		return ansi.Truncate(input, available, "") + "   " + hints
	}
	return ansi.Truncate(input, width, "")
}

func fitBlock(content string, width, height int) string {
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for index, line := range lines {
		if padding := width - lipgloss.Width(line); padding > 0 {
			lines[index] += strings.Repeat(" ", padding)
		}
	}
	blank := strings.Repeat(" ", width)
	for len(lines) < height {
		lines = append(lines, blank)
	}
	return strings.Join(lines, "\n")
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
