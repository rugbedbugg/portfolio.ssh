package ui

import (
	"image/color"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/rugbedbugg/portfolio.ssh/internal/content"
	"github.com/rugbedbugg/portfolio.ssh/internal/testutil"
)

func TestCanonicalCGAStylesResolveToNamedVisualHues(t *testing.T) {
	colors := []struct {
		name string
		got  lipgloss.ANSIColor
		want color.RGBA
	}{
		{name: "black", got: cgaBlack, want: color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}},
		{name: "green", got: cgaGreen, want: color.RGBA{R: 0x00, G: 0x80, B: 0x00, A: 0xff}},
		{name: "cyan", got: cgaCyan, want: color.RGBA{R: 0x00, G: 0x80, B: 0x80, A: 0xff}},
		{name: "red", got: cgaRed, want: color.RGBA{R: 0x80, G: 0x00, B: 0x00, A: 0xff}},
		{name: "gray", got: cgaGray, want: color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}},
		{name: "bright green", got: cgaBrightGreen, want: color.RGBA{R: 0x00, G: 0xff, B: 0x00, A: 0xff}},
		{name: "bright cyan", got: cgaBrightCyan, want: color.RGBA{R: 0x00, G: 0xff, B: 0xff, A: 0xff}},
		{name: "white", got: cgaWhite, want: color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}},
	}

	for _, test := range colors {
		t.Run(test.name, func(t *testing.T) {
			red, green, blue, alpha := test.got.RGBA()
			got := color.RGBA{R: uint8(red >> 8), G: uint8(green >> 8), B: uint8(blue >> 8), A: uint8(alpha >> 8)}
			if got != test.want {
				t.Fatalf("CGA %s resolves to %#v, want %#v", test.name, got, test.want)
			}
		})
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
			name:      "section index",
			configure: func(*Model) {},
			want:      []string{"j/k MOVE", "ENTER OPEN", "? HELP", ": COMMAND", "q EXIT"},
			doNotWant: []string{"ESC BACK"},
		},
		{
			name: "project list",
			configure: func(model *Model) {
				model.section = SectionProjects
				model.pane = PaneSection
			},
			want: []string{"j/k MOVE", "ENTER OPEN", "ESC BACK", "? HELP", ": COMMAND", "q EXIT"},
		},
		{
			name: "about detail",
			configure: func(model *Model) {
				model.section = SectionAbout
				model.pane = PaneSection
			},
			want:      []string{"ESC BACK", "? HELP", ": COMMAND", "q EXIT"},
			doNotWant: []string{"j/k MOVE", "ENTER OPEN"},
		},
		{
			name: "record detail",
			configure: func(model *Model) {
				model.section = SectionProjects
				model.pane = PaneRecord
			},
			want:      []string{"ESC BACK", "? HELP", ": COMMAND", "q EXIT"},
			doNotWant: []string{"j/k MOVE", "ENTER OPEN"},
		},
		{
			name: "command input",
			configure: func(model *Model) {
				model.focus = FocusCommand
				model.commandInput.Focus()
			},
			want:      []string{"TAB COMPLETE", "↑/↓ HISTORY", "ESC CANCEL", "ENTER RUN", ": COMMAND"},
			doNotWant: []string{"q EXIT", "j/k MOVE"},
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

func TestNarrowResearchDetailWrapsNonURLLinesWithinTerminalWidth(t *testing.T) {
	const width = 50
	model := New(content.Default(), width, 24)
	model.section = SectionResearch
	model.pane = PaneRecord
	publicationURL := model.data.Publications[0].URL
	contentWidth := panelContentWidth(width)

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
