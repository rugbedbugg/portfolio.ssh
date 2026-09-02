package ui

import (
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
	wideListWidth          = 26
	listGapWidth           = 4
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
	header := renderHeader(model, width)
	footer := renderFooter(model)
	bodyHeight := maxInt(1, height-lipgloss.Height(header)-lipgloss.Height(footer)-2)
	bodyWidth := maxInt(1, width-2)
	body := renderBody(model, bodyWidth)
	body = fitBlock(body, bodyWidth, bodyHeight)
	body = lipgloss.PlaceHorizontal(width, lipgloss.Center, body)
	return strings.Join([]string{header, "", body, "", footer}, "\n")
}

func renderHeader(model *Model, containerWidth int) string {
	tableWidth := maxInt(1, containerWidth-2)
	if tableWidth < responsiveBreakpoint {
		label := "Partha P.G.  p projects  r research  c contact"
		return lipgloss.PlaceHorizontal(containerWidth, lipgloss.Center, ansi.Truncate(label, tableWidth, ""))
	}

	// headerLabels is index-aligned with sections, so the active section marks
	// its own cell.
	active := sectionIndex(activeSection(model))
	accent := sectionAccent(activeSection(model))

	innerWidth := tableWidth - len(headerLabels) - 1
	cellWidths := distributeWidth(innerWidth, len(headerLabels))
	top := make([]string, 0, len(headerLabels))
	middle := make([]string, 0, len(headerLabels))
	bottom := make([]string, 0, len(headerLabels))
	for index, label := range headerLabels {
		top = append(top, strings.Repeat("─", cellWidths[index]))
		cell := centerText(label, cellWidths[index])
		if index == active {
			cell = activeHeaderStyle(accent).Render(cell)
		}
		middle = append(middle, cell)
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
	labels := make([]string, 0, len(model.data.Projects))
	for _, project := range model.data.Projects {
		labels = append(labels, project.Title)
	}
	project := model.data.Projects[selected]
	detail := []string{
		titleStyle.Render(project.Title),
		primaryCopyStyle.Render(wrapProse(project.Summary, detailWidth(width))),
		secondaryCopyStyle.Render(strings.Join(project.Tags, " · ")),
		primaryCopyStyle.Render(project.URL),
	}
	return renderCollection(model, width, collection{labels: labels, selected: selected, section: SectionProjects, detail: detail})
}

func renderResearch(model *Model, width int) []string {
	if len(model.data.Publications) == 0 {
		return []string{"no research"}
	}
	selected := safeIndex(model.selected, len(model.data.Publications))
	labels := make([]string, 0, len(model.data.Publications))
	for _, publication := range model.data.Publications {
		labels = append(labels, publication.Title)
	}
	publication := model.data.Publications[selected]
	detail := []string{
		titleStyle.Render(wrapProse(publication.Title, detailWidth(width))),
		terminalStateStyle.Render(publication.Venue),
		primaryCopyStyle.Render(wrapProse(publication.Contribution, detailWidth(width))),
		secondaryCopyStyle.Render(wrapProse(strings.Join(publication.Authors, ", "), detailWidth(width))),
		primaryCopyStyle.Render(publication.URL),
	}
	return renderCollection(model, width, collection{labels: labels, selected: selected, section: SectionResearch, detail: detail})
}

func renderContact(model *Model, width int) []string {
	if len(model.data.Links) == 0 {
		return []string{"no contact links"}
	}
	selected := safeIndex(model.selected, len(model.data.Links))
	labels := make([]string, 0, len(model.data.Links))
	for _, link := range model.data.Links {
		labels = append(labels, link.Label)
	}
	link := model.data.Links[selected]
	detail := []string{
		titleStyle.Render(link.Label),
		primaryCopyStyle.Render(link.URL),
	}
	return renderCollection(model, width, collection{labels: labels, selected: selected, section: SectionContact, detail: detail})
}

// collection is one section's choice list plus the detail for the current
// selection, carrying the section so the list can pick up its accent colour.
type collection struct {
	labels   []string
	selected int
	section  Section
	detail   []string
}

func renderCollection(model *Model, width int, data collection) []string {
	if model.pane == PaneRecord {
		return data.detail
	}
	if width < responsiveBreakpoint {
		return append(append(renderChoiceList(data, width), ""), data.detail...)
	}

	listWidth := listColumnWidth(width)
	menu := strings.Join(renderChoiceList(data, listWidth), "\n")
	right := strings.Join(data.detail, "\n")
	menu = fitBlock(menu, listWidth, maxInt(lipgloss.Height(menu), lipgloss.Height(right)))
	columns := lipgloss.JoinHorizontal(
		lipgloss.Top,
		menu,
		strings.Repeat(" ", listGapWidth),
		right,
	)
	return strings.Split(columns, "\n")
}

func renderChoiceList(data collection, width int) []string {
	accent := sectionAccent(data.section)
	lines := make([]string, 0, len(data.labels))
	for index, label := range data.labels {
		lines = append(lines, renderRecordChoice(index == data.selected, label, width, accent))
	}
	return lines
}

// listColumnWidth is the single source of truth for the choice/detail split so
// the menu and the detail column can never disagree about the boundary. The
// width is content-independent, keeping the divider stable across sections.
func listColumnWidth(width int) int {
	return minInt(wideListWidth, maxInt(1, width/2))
}

func detailWidth(width int) int {
	if width < responsiveBreakpoint {
		return width
	}
	return maxInt(1, width-listColumnWidth(width)-listGapWidth)
}

// renderRecordChoice fits the row to width as plain text before styling it, so
// the inverted bar spans the whole column instead of hugging the label and no
// truncation ever cuts through an escape sequence.
func renderRecordChoice(selected bool, label string, width int, accent lipgloss.ANSIColor) string {
	marker := "  "
	if selected {
		marker = "> "
	}
	row := ansi.Truncate(marker+label, maxInt(1, width), "…")
	if padding := width - lipgloss.Width(row); padding > 0 {
		row += strings.Repeat(" ", padding)
	}
	if selected {
		return selectionStyle(accent).Render(row)
	}
	return primaryCopyStyle.Render(row)
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

// commandInputWidth sizes the command line to the centered canvas rather than
// the raw terminal, so a wide client cannot push the prompt past the layout.
func commandInputWidth(terminalWidth int) int {
	container := minInt(maxInt(1, terminalWidth), maximumContainerWidth)
	return maxInt(1, container-2)
}

// renderFooter always occupies three rows — rule, command line, hints — so
// entering command mode never shifts the body or clips its last line.
func renderFooter(model *Model) string {
	containerWidth := minInt(maxInt(1, model.width), maximumContainerWidth)
	width := maxInt(1, containerWidth-2)

	commandLine := ""
	hints := navigationHints(model)
	if model.focus == FocusCommand {
		commandLine = model.commandInput.View()
		hints = "tab complete   ↑/↓ history   esc cancel   enter run"
	}

	return strings.Join([]string{
		centerLine(containerWidth, strings.Repeat("─", width)),
		centerLine(containerWidth, ansi.Truncate(commandLine, width, "")),
		centerLine(containerWidth, ansi.Truncate(hints, width, "")),
	}, "\n")
}

func navigationHints(model *Model) string {
	switch {
	case model.pane == PaneIndex:
		return "a about   p projects   r research   c contact   : command   ? help   q quit"
	case model.pane == PaneRecord || activeSection(model) == SectionAbout:
		return "esc back   : command   ? help   q quit"
	default:
		return "↑/↓ select   enter open   esc back   : command   ? help   q quit"
	}
}

func centerLine(containerWidth int, value string) string {
	return lipgloss.PlaceHorizontal(containerWidth, lipgloss.Center, value)
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
