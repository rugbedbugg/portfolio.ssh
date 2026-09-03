package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/rugbedbugg/portfolio.ssh/internal/content"
	"github.com/rugbedbugg/portfolio.ssh/internal/testutil"
)

func TestEachFaceDrawsRectangularGlyphs(t *testing.T) {
	faces := map[string]bannerFace{"money": moneyFace, "block": blockFace}
	for name, face := range faces {
		for character, glyph := range face.glyphs {
			if len(glyph) != face.rows {
				t.Errorf("%s face glyph %q has %d rows, want %d", name, character, len(glyph), face.rows)
				continue
			}
			for row, line := range glyph {
				if got, want := lipgloss.Width(line), lipgloss.Width(glyph[0]); got != want {
					t.Errorf("%s face glyph %q row %d is %d columns wide, want %d", name, character, row, got, want)
				}
			}
		}
	}
}

func TestMoneyFaceDrawsTheNameAtTheFullCanvasWidth(t *testing.T) {
	banner := renderBanner(moneyFace, "Partha P.G.", maximumContainerWidth-2)
	if banner == "" {
		t.Fatal("money banner did not render at the full canvas width")
	}

	plain := testutil.StripANSI(banner)
	if got := lipgloss.Height(plain); got != moneyRows {
		t.Fatalf("money banner is %d rows tall, want %d", got, moneyRows)
	}
	if !strings.Contains(plain, "$") {
		t.Fatalf("money banner contains no dollar stems:\n%s", plain)
	}
	for number, line := range strings.Split(plain, "\n") {
		if got := lipgloss.Width(line); got > maximumContainerWidth-2 {
			t.Fatalf("money banner row %d is %d columns wide, want at most %d", number+1, got, maximumContainerWidth-2)
		}
	}
}

func TestBlockFaceCoversTheAlphabetAndStaysShorter(t *testing.T) {
	if blockFace.rows >= moneyFace.rows {
		t.Fatalf("block face is %d rows, want fewer than the money face's %d", blockFace.rows, moneyFace.rows)
	}
	for character := 'A'; character <= 'Z'; character++ {
		if _, ok := blockFace.glyphs[character]; !ok {
			t.Errorf("block face has no glyph for %q", character)
		}
	}

	banner := renderBanner(blockFace, "Partha P.G.", maximumContainerWidth-2)
	if !strings.Contains(banner, "█") {
		t.Fatalf("block banner contains no block glyphs:\n%s", banner)
	}
}

func TestBannerDeclinesRatherThanRenderingBrokenOutput(t *testing.T) {
	if banner := renderBanner(moneyFace, "Partha P.G.", 20); banner != "" {
		t.Fatalf("banner rendered into 20 columns it cannot fit:\n%s", banner)
	}
	// A character with no glyph must make the whole banner stand down rather
	// than render a gap where the character should be.
	if banner := renderBanner(moneyFace, "Partha P.G. 3", maximumContainerWidth-2); banner != "" {
		t.Fatalf("banner rendered an unmapped character:\n%s", banner)
	}
}

// The money face is drawn only for the runes in the profile name. Any other
// name must fall through to the block face rather than lose the nameplate, so
// changing content.Profile.Name degrades the banner instead of breaking it.
func TestAnUnmappedNameStepsDownToTheBlockFace(t *testing.T) {
	const name = "Ada Lovelace"

	if banner := renderBanner(moneyFace, name, 200); banner != "" {
		t.Fatalf("money face rendered %q despite lacking its glyphs:\n%s", name, banner)
	}
	if banner := renderBanner(blockFace, name, 200); banner == "" {
		t.Fatalf("block face failed to render %q, so the nameplate has no fallback", name)
	}

	data := content.Default()
	data.Profile.Name = name
	model := New(data, 100, 30)
	view := testutil.StripANSI(render(model))
	if !strings.Contains(view, "█") {
		t.Fatalf("nameplate did not step down to the block face for %q:\n%s", name, view)
	}
}

func TestNameplateDegradesInsteadOfDisappearing(t *testing.T) {
	tests := []struct {
		name          string
		width, height int
		wantRows      int
	}{
		{name: "roomy", width: 100, height: 30, wantRows: moneyRows},
		// An 80-column terminal cannot fit the money face, so the nameplate
		// steps down to the narrower block face rather than to bare text.
		{name: "too narrow for the money face", width: 80, height: 30, wantRows: bannerRows},
		{name: "too short for any face", width: 100, height: 20, wantRows: 1},
		{name: "too narrow for any face", width: 50, height: 24, wantRows: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := New(content.Default(), test.width, test.height)
			view := testutil.StripANSI(render(model))

			if test.wantRows == 1 && !strings.Contains(view, model.data.Profile.Name) {
				t.Fatalf("nameplate dropped the plain name entirely:\n%s", view)
			}
			// Each face leaves a signature the other cannot: dollar stems for
			// the money face, full blocks for the block face.
			markers := map[int]string{moneyRows: "$", bannerRows: "█"}
			if marker, ok := markers[test.wantRows]; ok && !strings.Contains(view, marker) {
				t.Fatalf("nameplate did not use the %d-row face:\n%s", test.wantRows, view)
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
