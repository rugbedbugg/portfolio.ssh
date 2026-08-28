package command

import (
	"reflect"
	"testing"

	"github.com/rugbedbugg/portfolio.ssh/internal/content"
)

func TestParse(t *testing.T) {
	data := content.Default()

	tests := []struct {
		name        string
		input       string
		kind        Kind
		target      string
		wantErr     bool
		suggestions []string
	}{
		{name: "blank input is a no-op", input: "   \t", kind: Unknown},
		{name: "help", input: "help", kind: Help},
		{name: "about", input: "about", kind: About},
		{name: "projects", input: "projects", kind: Projects},
		{name: "project", input: "project reagent", kind: Project, target: "reagent"},
		{name: "research", input: "research", kind: Research},
		{name: "dispatches", input: "dispatches", kind: Dispatches},
		{name: "contact", input: "contact", kind: Contact},
		{name: "open", input: "open github", kind: Open, target: "github"},
		{name: "clear", input: "clear", kind: Clear},
		{name: "exit", input: "exit", kind: Exit},
		{name: "whoami alias", input: "whoami", kind: About},
		{name: "ls alias", input: "ls", kind: Projects},
		{name: "case insensitive", input: "  PrOjEcT   ReAgEnT  ", kind: Project, target: "reagent"},
		{name: "unique project prefix", input: "project rea", kind: Project, target: "reagent"},
		{name: "unique link prefix", input: "open git", kind: Open, target: "github"},
		{name: "unknown command", input: "version", kind: Unknown, wantErr: true},
		{name: "shell-like input is rejected", input: "rm -rf /", kind: Unknown, wantErr: true},
		{name: "missing project target", input: "project", kind: Project, wantErr: true},
		{name: "missing open target", input: "open", kind: Open, wantErr: true},
		{name: "extra project arguments", input: "project reagent --all", kind: Project, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Parse(test.input, data)
			if got.Kind != test.kind {
				t.Fatalf("Parse(%q).Kind = %v, want %v", test.input, got.Kind, test.kind)
			}
			if got.Target != test.target {
				t.Errorf("Parse(%q).Target = %q, want %q", test.input, got.Target, test.target)
			}
			if (got.Err != "") != test.wantErr {
				t.Errorf("Parse(%q).Err = %q, want error present: %v", test.input, got.Err, test.wantErr)
			}
		})
	}

	t.Run("ambiguous project prefix returns candidate IDs", func(t *testing.T) {
		portfolio := content.Portfolio{Projects: []content.Project{
			{ID: "alpha"},
			{ID: "alpine"},
		}}
		got := Parse("project al", portfolio)
		if got.Kind != Project || got.Target != "" {
			t.Fatalf("ambiguous project result = %#v, want Project with no target", got)
		}
		if got.Err == "" {
			t.Fatal("ambiguous project should explain how to correct the input")
		}
		if !reflect.DeepEqual(got.Suggestions, []string{"alpha", "alpine"}) {
			t.Fatalf("ambiguous project suggestions = %#v, want [alpha alpine]", got.Suggestions)
		}
	})

	t.Run("unknown project reports corrective error", func(t *testing.T) {
		got := Parse("project missing", data)
		if got.Kind != Project || got.Target != "" || got.Err == "" {
			t.Fatalf("unknown project result = %#v, want project error", got)
		}
	})

	t.Run("misspelled project suggests the closest ID", func(t *testing.T) {
		got := Parse("project reagnt", data)
		if got.Kind != Project || got.Target != "" || got.Err == "" {
			t.Fatalf("misspelled project result = %#v, want project error", got)
		}
		if !reflect.DeepEqual(got.Suggestions, []string{"reagent"}) {
			t.Fatalf("misspelled project suggestions = %#v, want [reagent]", got.Suggestions)
		}
	})

	t.Run("unknown open target reports corrective error", func(t *testing.T) {
		got := Parse("open missing", data)
		if got.Kind != Open || got.Target != "" || got.Err == "" {
			t.Fatalf("unknown open result = %#v, want open error", got)
		}
	})

	t.Run("unknown command suggests help", func(t *testing.T) {
		got := Parse("version", data)
		if !reflect.DeepEqual(got.Suggestions, []string{"help"}) {
			t.Fatalf("unknown command suggestions = %#v, want [help]", got.Suggestions)
		}
	})
}

func TestComplete(t *testing.T) {
	data := content.Default()

	tests := []struct {
		name       string
		input      string
		wantInput  string
		candidates []string
	}{
		{name: "command name", input: "hel", wantInput: "help ", candidates: []string{"help"}},
		{name: "project target list", input: "project ", wantInput: "project ", candidates: []string{"http-svr-200-ok", "reagent", "resonanceid-cli", "trionda-trifecta-26"}},
		{name: "project ID", input: "project rea", wantInput: "project reagent ", candidates: []string{"reagent"}},
		{name: "link ID", input: "open git", wantInput: "open github ", candidates: []string{"github"}},
		{name: "multiple candidates retain common prefix", input: "project re", wantInput: "project re", candidates: []string{"reagent", "resonanceid-cli"}},
		{name: "no match remains unchanged", input: "project zzz", wantInput: "project zzz", candidates: nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Complete(test.input, data)
			if got.Input != test.wantInput {
				t.Errorf("Complete(%q).Input = %q, want %q", test.input, got.Input, test.wantInput)
			}
			if !reflect.DeepEqual(got.Candidates, test.candidates) {
				t.Errorf("Complete(%q).Candidates = %#v, want %#v", test.input, got.Candidates, test.candidates)
			}
		})
	}
}
