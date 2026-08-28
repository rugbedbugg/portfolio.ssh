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
	if model.commandInput.Width() != 84 {
		t.Fatalf("resize command input width = %d, want 84", model.commandInput.Width())
	}
}

func TestNavigationWrapsAcrossSectionIndex(t *testing.T) {
	model := New(content.Default(), 120, 40)

	model = updateModel(t, model, key("k"))
	if model.selected != int(SectionContact) {
		t.Fatalf("k from first section selected %d, want contact index %d", model.selected, SectionContact)
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

func TestEnterOpensSelectedRecordWithinSection(t *testing.T) {
	model := New(content.Default(), 120, 40)
	model = updateModel(t, model, key("j"))
	model = updateModel(t, model, specialKey(tea.KeyEnter))
	model = updateModel(t, model, specialKey(tea.KeyDown))
	model = updateModel(t, model, specialKey(tea.KeyEnter))

	if model.pane != PaneRecord || model.selected != 1 {
		t.Fatalf("record enter state = pane %v, selected %d; want second project record", model.pane, model.selected)
	}
	if !strings.Contains(model.status, "Trionda-Trifecta-26") {
		t.Fatalf("record status = %q, want selected project title", model.status)
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
	if model.section != SectionProjects || model.pane != PaneRecord || model.selected != 2 {
		t.Fatalf("project resonance state = section %v, pane %v, selected %d; want ResonanceID-cli record", model.section, model.pane, model.selected)
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
		"dispatches",
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
