package content

import (
	"strings"
	"testing"
)

func TestDefaultPortfolioContent(t *testing.T) {
	portfolio := Default()

	if portfolio.Profile.Name == "" || portfolio.Profile.Biography == nil || len(portfolio.Profile.Biography) == 0 {
		t.Fatal("profile name and biography must be populated")
	}
	if got := len(portfolio.Projects); got != 4 {
		t.Fatalf("expected 4 projects, got %d", got)
	}
	if got := len(portfolio.Publications); got != 1 {
		t.Fatalf("expected 1 publication, got %d", got)
	}
	if got := len(portfolio.Dispatches); got != 2 {
		t.Fatalf("expected 2 dispatches, got %d", got)
	}
	if got := len(portfolio.Links); got != 3 {
		t.Fatalf("expected 3 contact links, got %d", got)
	}

	seenIDs := make(map[string]bool)
	seenURLs := make(map[string]bool)
	check := func(kind, id, url string) {
		t.Helper()
		if id == "" {
			t.Errorf("%s has an empty ID", kind)
		}
		if id != strings.ToLower(id) {
			t.Errorf("%s ID %q must be lowercase", kind, id)
		}
		if seenIDs[id] {
			t.Errorf("duplicate ID %q", id)
		}
		seenIDs[id] = true
		if url == "" {
			t.Errorf("%s %q has an empty URL", kind, id)
		}
		if seenURLs[url] {
			t.Errorf("duplicate URL %q", url)
		}
		seenURLs[url] = true
	}
	for _, project := range portfolio.Projects {
		check("project", project.ID, project.URL)
	}
	for _, publication := range portfolio.Publications {
		check("publication", publication.ID, publication.URL)
	}
	for _, dispatch := range portfolio.Dispatches {
		check("dispatch", dispatch.ID, dispatch.URL)
	}
	for _, link := range portfolio.Links {
		check("link", link.ID, link.URL)
	}
}

func TestDefaultReturnsFreshSlices(t *testing.T) {
	first := Default()
	first.Profile.Biography[0] = "changed"
	first.Projects[0].Tags[0] = "changed"
	first.Profile.Biography = append(first.Profile.Biography, "changed")
	first.Projects = append(first.Projects, Project{ID: "changed"})

	second := Default()
	if second.Profile.Biography[0] == "changed" {
		t.Fatal("biography must not be shared between Default results")
	}
	if second.Projects[0].Tags[0] == "changed" {
		t.Fatal("project tags must not be shared between Default results")
	}
	if len(second.Profile.Biography) != 3 || len(second.Projects) != 4 {
		t.Fatal("Default must return fresh top-level slices")
	}
}
