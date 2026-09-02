package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/rugbedbugg/portfolio.ssh/internal/content"
	"github.com/rugbedbugg/portfolio.ssh/internal/testutil"
)

func TestNewStartsAtAboutInSectionIndex(t *testing.T) {
	model := New(content.Default(), 120, 40)

	if model.section != SectionAbout || model.pane != PaneIndex || model.selected != 0 {
		t.Fatalf("new model state = section %v, pane %v, selected %d; want about section index at 0", model.section, model.pane, model.selected)
	}
	if model.focus != FocusNavigation || model.commandInput.Focused() {
		t.Fatal("new model must keep navigation focused")
	}
}

func TestWindowResizeUpdatesSessionAndCommandInputWidth(t *testing.T) {
	model := New(content.Default(), 120, 40)
	model = updateModel(t, model, tea.WindowSizeMsg{Width: 84, Height: 28})

	if model.width != 84 || model.height != 28 {
		t.Fatalf("resize dimensions = %dx%d, want 84x28", model.width, model.height)
	}
	// The command line follows the centered canvas, not the raw terminal, so a
	// wide client cannot push the prompt past the layout.
	if want := commandInputWidth(84); model.commandInput.Width() != want {
		t.Fatalf("resize command input width = %d, want canvas width %d", model.commandInput.Width(), want)
	}
	if model.commandInput.Width() >= 84 {
		t.Fatalf("command input width %d was not clamped below the 84-column terminal", model.commandInput.Width())
	}
}

func TestNavigationWrapsAcrossSectionIndex(t *testing.T) {
	model := New(content.Default(), 120, 40)

	model = updateModel(t, model, key("k"))
	if model.selected != len(sections)-1 || sections[model.selected] != SectionContact {
		t.Fatalf("k from first section selected %d (%v), want final contact section", model.selected, sections[model.selected])
	}
	model = updateModel(t, model, key("j"))
	if model.selected != 0 {
		t.Fatalf("j from final section selected %d, want 0", model.selected)
	}
	model = updateModel(t, model, specialKey(tea.KeyDown))
	if model.selected != int(SectionProjects) {
		t.Fatalf("down arrow selected %d, want projects index %d", model.selected, SectionProjects)
	}
	model = updateModel(t, model, specialKey(tea.KeyUp))
	if model.selected != 0 {
		t.Fatalf("up arrow selected %d, want 0", model.selected)
	}
}

func TestEnterOpensSelectedSectionAndEscapeReturnsToIndex(t *testing.T) {
	model := New(content.Default(), 120, 40)
	model = updateModel(t, model, key("j"))
	model = updateModel(t, model, specialKey(tea.KeyEnter))

	if model.section != SectionProjects || model.pane != PaneSection {
		t.Fatalf("enter state = section %v, pane %v; want projects section pane", model.section, model.pane)
	}
	model = updateModel(t, model, specialKey(tea.KeyEscape))
	if model.pane != PaneIndex || model.selected != int(SectionProjects) {
		t.Fatalf("escape state = pane %v, selected %d; want projects section index", model.pane, model.selected)
	}
}

func TestHeaderShortcutsOpenPrimarySections(t *testing.T) {
	tests := []struct {
		key     string
		section Section
	}{
		{key: "p", section: SectionProjects},
		{key: "r", section: SectionResearch},
		{key: "c", section: SectionContact},
	}

	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			model := New(content.Default(), 120, 36)
			model = updateModel(t, model, key(test.key))
			if model.section != test.section || model.pane != PaneSection {
				t.Fatalf("%s shortcut state = section %v, pane %v; want section %v pane %v", test.key, model.section, model.pane, test.section, PaneSection)
			}
		})
	}
}

func TestEnterCopiesSelectedRecordURLWithoutLeavingTheSection(t *testing.T) {
	model := New(content.Default(), 120, 40)
	model = updateModel(t, model, key("j"))
	model = updateModel(t, model, specialKey(tea.KeyEnter))
	model = updateModel(t, model, specialKey(tea.KeyDown))

	updated, cmd := model.Update(specialKey(tea.KeyEnter))
	model = updated.(*Model)

	// A server cannot open a browser on the far end of an SSH session, so enter
	// hands the visitor the URL instead of navigating to a separate pane.
	if model.pane != PaneSection || model.selected != 1 {
		t.Fatalf("enter state = pane %v, selected %d; want to stay on the second project", model.pane, model.selected)
	}
	if cmd == nil {
		t.Fatal("enter on a record issued no clipboard command")
	}
	wantURL := content.Default().Projects[1].URL
	if !strings.Contains(model.status, wantURL) {
		t.Fatalf("record status = %q, want the copied URL %q", model.status, wantURL)
	}
}

func TestEnterOnAboutSectionIssuesNoClipboardCommand(t *testing.T) {
	model := New(content.Default(), 120, 40)
	model = updateModel(t, model, key("a"))

	_, cmd := model.Update(specialKey(tea.KeyEnter))
	if cmd != nil {
		t.Fatal("enter on the about section issued a clipboard command; want none")
	}
}

func TestColonFocusesInputAndQQuitsOnlyOutsideInput(t *testing.T) {
	model := New(content.Default(), 120, 40)
	model = updateModel(t, model, key(":"))
	if model.focus != FocusCommand || !model.commandInput.Focused() {
		t.Fatal("colon must focus command input")
	}

	model = updateModel(t, model, key("q"))
	if model.commandInput.Value() != "q" {
		t.Fatalf("q while command input is focused = %q, want input text q", model.commandInput.Value())
	}

	model = updateModel(t, model, specialKey(tea.KeyEscape))
	if model.focus != FocusNavigation {
		t.Fatal("escape must restore navigation focus")
	}
	_, quit := model.Update(key("q"))
	if _, ok := quit().(tea.QuitMsg); !ok {
		t.Fatalf("q outside command input command = %T, want tea.QuitMsg", quit())
	}
}

func TestFocusedEnterSubmitsCommandsAndOpensRequestedContent(t *testing.T) {
	model := New(content.Default(), 120, 40)
	model = submit(t, model, "projects")
	if model.section != SectionProjects || model.pane != PaneSection {
		t.Fatalf("projects command state = section %v, pane %v; want projects section", model.section, model.pane)
	}
	if len(model.history) != 1 || model.history[0] != "projects" {
		t.Fatalf("history after projects = %#v, want submitted projects command", model.history)
	}

	model = submit(t, model, "project resonance")
	if model.section != SectionProjects || model.pane != PaneSection || model.selected != 2 {
		t.Fatalf("project resonance state = section %v, pane %v, selected %d; want the ResonanceID-cli record selected in place", model.section, model.pane, model.selected)
	}
	if !strings.Contains(model.status, "ResonanceID-cli") || !strings.Contains(model.status, "https://github.com/rugbedbugg/ResonanceID-cli") {
		t.Fatalf("project resonance status = %q, want title and copyable URL", model.status)
	}
}

func TestCommandHistoryTraversesSubmittedCommands(t *testing.T) {
	model := New(content.Default(), 120, 40)
	model = submit(t, model, "about")
	model = submit(t, model, "projects")
	model = updateModel(t, model, key(":"))

	model = updateModel(t, model, specialKey(tea.KeyUp))
	if model.commandInput.Value() != "projects" {
		t.Fatalf("first up recalled %q, want most recent command", model.commandInput.Value())
	}
	model = updateModel(t, model, specialKey(tea.KeyUp))
	if model.commandInput.Value() != "about" {
		t.Fatalf("second up recalled %q, want oldest command", model.commandInput.Value())
	}
	model = updateModel(t, model, specialKey(tea.KeyDown))
	if model.commandInput.Value() != "projects" {
		t.Fatalf("down recalled %q, want next command", model.commandInput.Value())
	}
}

func TestTabAppliesCommandCompletion(t *testing.T) {
	model := New(content.Default(), 120, 40)
	model = updateModel(t, model, key(":"))
	model.commandInput.SetValue("hel")
	model = updateModel(t, model, specialKey(tea.KeyTab))

	if model.commandInput.Value() != "help " {
		t.Fatalf("tab completion = %q, want help with trailing space", model.commandInput.Value())
	}
}

func TestQuestionMarkShowsCompleteHelpWithoutEnteringCommandMode(t *testing.T) {
	model := New(content.Default(), 120, 40)
	model = updateModel(t, model, key("?"))

	if model.focus != FocusNavigation || model.commandInput.Focused() {
		t.Fatal("question mark help must leave navigation focused")
	}
	if len(model.history) != 0 {
		t.Fatalf("question mark help added command history %#v; want none", model.history)
	}
	view := testutil.StripANSI(model.View().Content)
	for _, want := range []string{
		"help",
		"about/whoami",
		"projects/ls",
		"project <id>",
		"research",
		"contact",
		"open <id>",
		"clear",
		"exit",
	} {
		if !strings.Contains(model.status, want) {
			t.Fatalf("question mark help status %q missing %q", model.status, want)
		}
		if !strings.Contains(view, want) {
			t.Fatalf("question mark help view missing %q:\n%s", want, view)
		}
	}
}

func TestClearResetsTransientSessionStateWithoutChangingPortfolio(t *testing.T) {
	model := New(content.Default(), 120, 40)
	projects := len(model.data.Projects)
	model = submit(t, model, "about")
	if model.status == "" || len(model.history) == 0 {
		t.Fatal("about command must produce transient state before clear")
	}

	model = submit(t, model, "clear")
	if model.status != "" || len(model.history) != 0 || model.historyIndex != -1 {
		t.Fatalf("clear state = status %q, history %#v, index %d; want no transient state", model.status, model.history, model.historyIndex)
	}
	if len(model.data.Projects) != projects {
		t.Fatalf("clear changed portfolio projects to %d, want %d", len(model.data.Projects), projects)
	}
}

func TestExitCommandReturnsTeaQuit(t *testing.T) {
	model := New(content.Default(), 120, 40)
	model = updateModel(t, model, key(":"))
	model.commandInput.SetValue("exit")
	updated, quit := model.Update(specialKey(tea.KeyEnter))
	if _, ok := updated.(*Model); !ok {
		t.Fatalf("exit Update returned %T, want *Model", updated)
	}
	if quit == nil {
		t.Fatal("exit command returned no Bubble Tea command")
	}
	if _, ok := quit().(tea.QuitMsg); !ok {
		t.Fatalf("exit command = %T, want tea.QuitMsg", quit())
	}
}

func TestInvalidCommandsSetVisibleStatus(t *testing.T) {
	model := New(content.Default(), 120, 40)
	model = submit(t, model, "version")

	if model.status == "" || !strings.Contains(model.View().Content, model.status) {
		t.Fatalf("invalid command status = %q; want a visible correction", model.status)
	}

	ambiguous := content.Portfolio{Projects: []content.Project{{ID: "alpha"}, {ID: "alpine"}}}
	model = New(ambiguous, 120, 40)
	model = submit(t, model, "project al")
	if !strings.Contains(model.status, "ambiguous") || !strings.Contains(model.status, "alpha") || !strings.Contains(model.status, "alpine") {
		t.Fatalf("ambiguous command status = %q; want visible corrective candidates", model.status)
	}
}

func updateModel(t *testing.T, model *Model, msg tea.Msg) *Model {
	t.Helper()
	updated, _ := model.Update(msg)
	result, ok := updated.(*Model)
	if !ok {
		t.Fatalf("Update returned %T, want *Model", updated)
	}
	return result
}

func key(text string) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: text, Code: []rune(text)[0]})
}

func specialKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code})
}

func submit(t *testing.T, model *Model, input string) *Model {
	t.Helper()
	if model.focus != FocusCommand {
		model = updateModel(t, model, key(":"))
	}
	model.commandInput.SetValue(input)
	return updateModel(t, model, specialKey(tea.KeyEnter))
}
