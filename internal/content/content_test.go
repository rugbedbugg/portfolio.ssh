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
	counts := []struct {
		name string
		got  int
		want int
	}{
		{name: "projects", got: len(portfolio.Projects), want: 4},
		{name: "publications", got: len(portfolio.Publications), want: 1},
		{name: "dispatches", got: len(portfolio.Dispatches), want: 2},
		{name: "contact links", got: len(portfolio.Links), want: 3},
	}
	for _, test := range counts {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Fatalf("expected %d %s, got %d", test.want, test.name, test.got)
			}
		})
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
