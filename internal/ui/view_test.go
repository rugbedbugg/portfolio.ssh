package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/rugbedbugg/portfolio.ssh/internal/content"
	"github.com/rugbedbugg/portfolio.ssh/internal/testutil"
)

func TestSelectedRowInvertsSectionAccentAcrossFullColumn(t *testing.T) {
	const columnWidth = 20
	tests := []struct {
		name    string
		section Section
		want    string
	}{
		{name: "projects", section: SectionProjects, want: "\x1b[38;5;0;48;5;202m"},
		{name: "research", section: SectionResearch, want: "\x1b[38;5;0;48;5;14m"},
		{name: "contact", section: SectionContact, want: "\x1b[38;5;0;48;5;10m"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			row := renderRecordChoice(true, "ReAgent", columnWidth, sectionAccent(test.section))

			if !strings.Contains(row, test.want) {
				t.Errorf("selected %s row = %q; want inverted accent %q", test.name, row, test.want)
			}
			plain := testutil.StripANSI(row)
			if lipgloss.Width(plain) != columnWidth {
				t.Errorf("selected row spans %d columns, want the full %d-column bar: %q", lipgloss.Width(plain), columnWidth, plain)
			}
			if strings.TrimRight(plain, " ") != "> ReAgent" {
				t.Fatalf("selected row lost its plain-text marker: %q", plain)
			}
		})
	}
}

func TestSelectedRowResetsStyleAfterThePaddedBar(t *testing.T) {
	row := renderRecordChoice(true, "ReAgent", 20, sectionAccent(SectionProjects))

	// The reset must follow the padding, otherwise the background bleeds to the
	// end of the terminal line instead of stopping at the column boundary.
	if !strings.HasSuffix(row, " \x1b[m") {
		t.Fatalf("selected row = %q; want the style reset after its trailing padding", row)
	}
}

func TestUnselectedRowPaintsNoBackground(t *testing.T) {
	row := renderRecordChoice(false, "ReAgent", 20, sectionAccent(SectionProjects))

	if strings.Contains(row, "\x1b[48;") || strings.Contains(row, "48;5;") {
		t.Fatalf("unselected row = %q; want no painted background", row)
	}
}

func TestWideViewCentersTerminalShopHeader(t *testing.T) {
	const width = 120
	model := New(content.Default(), width, 36)
	view := testutil.StripANSI(render(model))

	for _, want := range []string{"Partha P.G.", "p projects", "r research", "c contact"} {
		if !strings.Contains(view, want) {
			t.Errorf("header missing %q:\n%s", want, view)
		}
	}

	lines := strings.Split(view, "\n")
	headerLine := ""
	for _, line := range lines {
		if strings.Contains(line, "┌") {
			headerLine = line
			break
		}
	}
	if headerLine == "" {
		t.Fatalf("view has no Terminal Shop header:\n%s", view)
	}
	if got := strings.Index(headerLine, "┌"); got != 21 {
		t.Fatalf("120-column header begins at column %d, want centered column 21:\n%s", got, view)
	}
	if strings.Count(view, "┌") != 1 || strings.Count(view, "└") != 1 {
		t.Fatalf("view contains bordered content panels in addition to the header:\n%s", view)
	}
}

func TestBaseViewLeavesTerminalBackgroundUnpainted(t *testing.T) {
	model := New(content.Default(), 120, 36)
	raw := render(model)

	if strings.Contains(raw, "\x1b[48;") {
		t.Fatalf("base view paints a terminal background: %q", raw)
	}
}

func TestPortfolioViewOmitsDossierJargon(t *testing.T) {
	model := New(content.Default(), 120, 36)
	view := testutil.StripANSI(render(model))

	for _, unwanted := range []string{
		"OXIDE",
		"DOSSIER",
		"SECTION INDEX",
		"CASE //",
		"RECORD //",
		"CHANNEL //",
		"CONTRIBUTION //",
		"STATUS //",
	} {
		if strings.Contains(view, unwanted) {
			t.Errorf("portfolio view contains unnecessary label %q:\n%s", unwanted, view)
		}
	}
}

func TestHeaderIsSeparatedFromBody(t *testing.T) {
	model := New(content.Default(), 120, 36)
	lines := strings.Split(testutil.StripANSI(render(model)), "\n")

	headerBottom := -1
	for index, line := range lines {
		if strings.Contains(line, "└") {
			headerBottom = index
			break
		}
	}
	if headerBottom < 0 || headerBottom+1 >= len(lines) {
		t.Fatalf("view has no complete header:\n%s", strings.Join(lines, "\n"))
	}
	if strings.TrimSpace(lines[headerBottom+1]) != "" {
		t.Fatalf("body touches header without Terminal Shop spacing:\n%s", strings.Join(lines, "\n"))
	}
}

func TestContactShowsDestinationsWithoutRedundantDescriptions(t *testing.T) {
	model := New(content.Default(), 120, 36)
	model.section = SectionContact
	model.pane = PaneSection
	view := testutil.StripANSI(render(model))

	if !strings.Contains(view, "GitHub") || !strings.Contains(view, model.data.Links[0].URL) {
		t.Fatalf("contact view omits the selected destination:\n%s", view)
	}
	for _, link := range model.data.Links {
		if strings.Contains(view, link.Description) {
			t.Errorf("contact view includes redundant description %q:\n%s", link.Description, view)
		}
	}
}

func TestFooterAdvertisesOnlyContextuallyAvailableKeys(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*Model)
		want      []string
		doNotWant []string
	}{
		{
			name:      "home",
			configure: func(*Model) {},
			want:      []string{"a about", "p projects", "r research", "c contact", ": command", "? help", "q quit"},
			doNotWant: []string{"select", "open", "back"},
		},
		{
			name: "project list",
			configure: func(model *Model) {
				model.section = SectionProjects
				model.pane = PaneSection
			},
			want:      []string{"↑/↓ select", "enter open", "esc back", ": command", "? help", "q quit"},
			doNotWant: []string{"tab complete"},
		},
		{
			name: "about detail",
			configure: func(model *Model) {
				model.section = SectionAbout
				model.pane = PaneSection
			},
			want:      []string{"esc back", ": command", "? help", "q quit"},
			doNotWant: []string{"select", "enter open"},
		},
		{
			name: "record detail",
			configure: func(model *Model) {
				model.section = SectionProjects
				model.pane = PaneRecord
			},
			want:      []string{"esc back", ": command", "? help", "q quit"},
			doNotWant: []string{"select", "enter open"},
		},
		{
			name: "command input",
			configure: func(model *Model) {
				model.focus = FocusCommand
				model.commandInput.Focus()
			},
			want:      []string{"tab complete", "↑/↓ history", "esc cancel", "enter run"},
			doNotWant: []string{"q quit", "select"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := New(content.Default(), 160, 30)
			test.configure(model)
			footer := testutil.StripANSI(renderFooter(model))
			for _, want := range test.want {
				if !strings.Contains(footer, want) {
					t.Errorf("footer %q missing %q", footer, want)
				}
			}
			for _, unwanted := range test.doNotWant {
				if strings.Contains(footer, unwanted) {
					t.Errorf("footer %q unexpectedly contains %q", footer, unwanted)
				}
			}
		})
	}
}

func TestWideViewContainsNavigationSelectedProjectAndFooter(t *testing.T) {
	model := New(content.Default(), 100, 30)
	model.section = SectionProjects
	model.pane = PaneSection

	view := testutil.StripANSI(render(model))
	for _, want := range []string{
		"p projects",
		"> ReAgent",
		"An agentic retrosynthesis framework",
		model.data.Projects[0].URL,
		"q quit",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("wide view missing %q:\n%s", want, view)
		}
	}
}

func TestNarrowHomeShowsOnlyEssentialProfileContent(t *testing.T) {
	model := New(content.Default(), 50, 24)

	view := testutil.StripANSI(render(model))
	if !strings.Contains(view, "Partha P.G.") || !strings.Contains(view, model.data.Profile.Tagline) {
		t.Fatalf("narrow home missing essential profile content:\n%s", view)
	}
	if strings.Contains(view, model.data.Projects[0].Summary) {
		t.Fatalf("narrow home unexpectedly contains project details:\n%s", view)
	}
}

func TestNarrowDetailIncludesVisibleBackHint(t *testing.T) {
	model := New(content.Default(), 50, 24)
	model.pane = PaneSection

	view := testutil.StripANSI(render(model))
	if !strings.Contains(view, "esc back") {
		t.Fatalf("narrow detail missing back hint:\n%s", view)
	}
	if !strings.Contains(view, "I build things to understand them.") {
		t.Fatalf("narrow detail missing profile biography:\n%s", view)
	}
}

func TestEverySectionRendersItsPortfolioContent(t *testing.T) {
	data := content.Default()
	tests := []struct {
		name    string
		section Section
		want    string
	}{
		{name: "about", section: SectionAbout, want: "parts of software people treat as a black box"},
		{name: "projects", section: SectionProjects, want: "reaction routes"},
		{name: "research", section: SectionResearch, want: data.Publications[0].Contribution},
		{name: "contact", section: SectionContact, want: data.Links[0].URL},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := New(data, 120, 32)
			model.section = test.section
			model.pane = PaneSection

			view := testutil.StripANSI(render(model))
			if !strings.Contains(normalizeWhitespace(view), normalizeWhitespace(test.want)) {
				t.Fatalf("%s view missing real portfolio content %q:\n%s", test.name, test.want, view)
			}
		})
	}
}

func normalizeWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func TestRecordDetailsKeepPlainURLsCopyable(t *testing.T) {
	data := content.Default()
	tests := []struct {
		name    string
		section Section
		url     string
	}{
		{name: "project", section: SectionProjects, url: data.Projects[0].URL},
		{name: "research", section: SectionResearch, url: data.Publications[0].URL},
		{name: "contact", section: SectionContact, url: data.Links[0].URL},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := New(data, 180, 36)
			model.section = test.section
			model.pane = PaneRecord

			raw := render(model)
			view := testutil.StripANSI(raw)
			if !strings.Contains(view, test.url) {
				t.Fatalf("%s detail missing contiguous URL %q:\n%s", test.name, test.url, view)
			}
			if strings.Contains(raw, "\x1b]8;") {
				t.Fatalf("%s detail wraps URL in an OSC 8 hyperlink", test.name)
			}
		})
	}
}

func TestResizeRenderingHandlesWideNarrowAndTinyTerminals(t *testing.T) {
	model := New(content.Default(), 120, 40)
	sizes := []tea.WindowSizeMsg{
		{Width: 100, Height: 30},
		{Width: 60, Height: 20},
		{Width: 20, Height: 8},
	}

	for _, size := range sizes {
		model = updateModel(t, model, size)
		view := model.View().Content
		if view == "" {
			t.Fatalf("%dx%d resize rendered an empty view", size.Width, size.Height)
		}
		if strings.Contains(view, "%!") || strings.Contains(view, "\x1b[-") {
			t.Fatalf("%dx%d resize contains a negative-width artifact:\n%s", size.Width, size.Height, testutil.StripANSI(view))
		}
	}

	tiny := testutil.StripANSI(model.View().Content)
	if !strings.Contains(tiny, "terminal too small") || !strings.Contains(tiny, "q quit") {
		t.Fatalf("20x8 view must explain the minimum size and retain an exit hint:\n%s", tiny)
	}
}

func TestResponsiveViewsDoNotOverflowTerminalWidth(t *testing.T) {
	for _, width := range []int{100, 60, 50, 24, 20} {
		for _, commandFocused := range []bool{false, true} {
			model := New(content.Default(), width, 20)
			if width == 20 {
				model.height = 8
			}
			if commandFocused {
				model.focus = FocusCommand
				model.commandInput.Focus()
				model.commandInput.SetValue("project trionda-trifecta-26")
			}

			for lineNumber, line := range strings.Split(render(model), "\n") {
				if got := lipgloss.Width(line); got > width {
					t.Fatalf("%d-column view (command focused: %t) line %d is %d columns wide:\n%s", width, commandFocused, lineNumber+1, got, testutil.StripANSI(line))
				}
			}
		}
	}
}

func TestNarrowResearchDetailWrapsNonURLLinesWithinTerminalWidth(t *testing.T) {
	const width = 50
	model := New(content.Default(), width, 24)
	model.section = SectionResearch
	model.pane = PaneRecord
	publicationURL := model.data.Publications[0].URL
	contentWidth := width

	detail := strings.Join(renderResearch(model, contentWidth), "\n")
	if !strings.Contains(testutil.StripANSI(detail), publicationURL) {
		t.Fatalf("narrow research detail missing copyable URL %q:\n%s", publicationURL, testutil.StripANSI(detail))
	}
	for lineNumber, line := range strings.Split(detail, "\n") {
		plainLine := testutil.StripANSI(line)
		if strings.Contains(plainLine, publicationURL) {
			continue
		}
		if got := lipgloss.Width(line); got > contentWidth {
			t.Fatalf("narrow research detail non-URL line %d is %d columns wide, want at most %d:\n%s", lineNumber+1, got, contentWidth, plainLine)
		}
	}

	view := testutil.StripANSI(render(model))
	if !strings.Contains(view, "https://doi.org/") || !strings.Contains(view, "5.11439749") {
		t.Fatalf("rendered narrow research detail obscures the publication URL:\n%s", view)
	}
}
