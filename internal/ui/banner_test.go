package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/rugbedbugg/portfolio.ssh/internal/content"
	"github.com/rugbedbugg/portfolio.ssh/internal/testutil"
)

func TestBannerDrawsTheNameInBlockGlyphs(t *testing.T) {
	banner := renderBanner("Partha P.G.", maximumContainerWidth-2)
	if banner == "" {
		t.Fatal("banner did not render at the full canvas width")
	}

	plain := testutil.StripANSI(banner)
	if got := lipgloss.Height(plain); got != bannerRows {
		t.Fatalf("banner is %d rows tall, want %d", got, bannerRows)
	}
	if !strings.Contains(plain, "█") {
		t.Fatalf("banner contains no block glyphs:\n%s", plain)
	}
	for number, line := range strings.Split(plain, "\n") {
		if got := lipgloss.Width(line); got > maximumContainerWidth-2 {
			t.Fatalf("banner row %d is %d columns wide, want at most %d", number+1, got, maximumContainerWidth-2)
		}
	}
}

func TestBannerDeclinesRatherThanRenderingBrokenOutput(t *testing.T) {
	if banner := renderBanner("Partha P.G.", 20); banner != "" {
		t.Fatalf("banner rendered into 20 columns it cannot fit:\n%s", banner)
	}
	// A character with no glyph must make the whole banner stand down rather
	// than render a gap where the character should be.
	if banner := renderBanner("Partha P.G. 3", maximumContainerWidth-2); banner != "" {
		t.Fatalf("banner rendered an unmapped character:\n%s", banner)
	}
}

func TestNameplateDegradesInsteadOfDisappearing(t *testing.T) {
	tests := []struct {
		name          string
		width, height int
		wantRows      int
	}{
		{name: "roomy", width: 100, height: 30, wantRows: bannerRows},
		{name: "too short for block glyphs", width: 100, height: 20, wantRows: 1},
		{name: "too narrow for block glyphs", width: 50, height: 24, wantRows: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := New(content.Default(), test.width, test.height)
			view := testutil.StripANSI(render(model))

			if test.wantRows == 1 && !strings.Contains(view, model.data.Profile.Name) {
				t.Fatalf("nameplate dropped the plain name entirely:\n%s", view)
			}
			if test.wantRows > 1 && !strings.Contains(view, "█") {
				t.Fatalf("nameplate did not use block glyphs:\n%s", view)
			}
		})
	}
}

func TestNameplateNeverGrowsTheViewport(t *testing.T) {
	for _, height := range []int{12, 20, 24, 30, 40} {
		model := New(content.Default(), 100, height)
		view := render(model)

		if got := lipgloss.Height(view); got != height {
			t.Fatalf("at terminal height %d the view is %d rows, want exactly %d", height, got, height)
		}
	}
}

func TestNameplateLeavesTheBodyReadable(t *testing.T) {
	// The banner spends the body's slack, never its minimum.
	model := New(content.Default(), 100, 30)
	model.openSectionByCommand(SectionResearch)
	view := testutil.StripANSI(render(model))

	for _, want := range []string{"IEEE SISIMPACT 2025", model.data.Publications[0].URL} {
		if !strings.Contains(view, want) {
			t.Fatalf("banner squeezed %q out of the research detail:\n%s", want, view)
		}
	}
}
