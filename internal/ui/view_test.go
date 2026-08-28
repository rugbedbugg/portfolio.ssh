package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/rugbedbugg/portfolio.ssh/internal/content"
	"github.com/rugbedbugg/portfolio.ssh/internal/testutil"
)

func TestCanonicalCGAStylesUseANSIIndexedColors(t *testing.T) {
	colors := []struct {
		name string
		got  lipgloss.ANSIColor
		want lipgloss.ANSIColor
	}{
		{name: "black", got: cgaBlack, want: 0},
		{name: "green", got: cgaGreen, want: 2},
		{name: "cyan", got: cgaCyan, want: 3},
		{name: "red", got: cgaRed, want: 4},
		{name: "gray", got: cgaGray, want: 7},
		{name: "bright green", got: cgaBrightGreen, want: 10},
		{name: "bright cyan", got: cgaBrightCyan, want: 11},
		{name: "white", got: cgaWhite, want: 15},
	}

	for _, test := range colors {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Fatalf("CGA %s index = %d, want %d", test.name, test.got, test.want)
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
		"CASE FILES",
		"> ReAgent",
		"An agentic retrosynthesis framework",
		model.data.Projects[0].URL,
		": COMMAND",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("wide view missing %q:\n%s", want, view)
		}
	}
}

func TestNarrowIndexDoesNotRenderDossierBesideNavigation(t *testing.T) {
	model := New(content.Default(), 50, 24)

	view := testutil.StripANSI(render(model))
	if !strings.Contains(view, "CASE FILES") {
		t.Fatalf("narrow index missing navigation:\n%s", view)
	}
	if strings.Contains(view, model.data.Profile.Biography[0]) {
		t.Fatalf("narrow index unexpectedly contains dossier copy:\n%s", view)
	}
}

func TestNarrowDetailIncludesVisibleBackHint(t *testing.T) {
	model := New(content.Default(), 50, 24)
	model.pane = PaneSection

	view := testutil.StripANSI(render(model))
	if !strings.Contains(view, "ESC BACK") {
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
		{name: "projects", section: SectionProjects, want: "plans reaction routes"},
		{name: "research", section: SectionResearch, want: data.Publications[0].Contribution},
		{name: "dispatches", section: SectionDispatches, want: "averaged over 1000 runs"},
		{name: "contact", section: SectionContact, want: data.Links[0].Description},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := New(data, 120, 32)
			model.section = test.section
			model.pane = PaneSection

			view := testutil.StripANSI(render(model))
			if !strings.Contains(view, test.want) {
				t.Fatalf("%s view missing real portfolio content %q:\n%s", test.name, test.want, view)
			}
		})
	}
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
		{name: "dispatch", section: SectionDispatches, url: data.Dispatches[0].URL},
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
	if !strings.Contains(tiny, "TERMINAL TOO SMALL") || !strings.Contains(tiny, "q EXIT") {
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
