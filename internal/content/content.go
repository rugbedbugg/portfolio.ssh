// Package content contains the typed, session-independent portfolio dossier.
package content

// Link is a contact or social destination.
type Link struct {
	ID          string
	Label       string
	URL         string
	Description string
}

// Project is a portfolio case file.
type Project struct {
	ID      string
	Title   string
	Summary string
	URL     string
	Tags    []string
}

// Publication is a peer-reviewed research record.
type Publication struct {
	ID           string
	Title        string
	Venue        string
	Contribution string
	URL          string
	Authors      []string
}

// Profile is the subject identity and biography.
type Profile struct {
	Name      string
	Tagline   string
	Biography []string
}

// Portfolio is the complete portfolio dossier.
type Portfolio struct {
	Profile      Profile
	Projects     []Project
	Publications []Publication
	Links        []Link
}

var defaultProfile = Profile{
	Name:    "Partha P.G.",
	Tagline: "AI & Low-Level Systems",
	Biography: []string{
		"I build things to understand them. My instinct with any system is to take it apart, see why it holds together, and put it back cleaner. That is most of how I learn and how I think.",
		"I'm pulled toward the parts of software people treat as a black box, and toward AI, especially agents and systems that act on their own. I would rather know what is happening underneath than trust the surface.",
		"What I make is usually one of two things: a rebuild done to learn how something works, or a tool built because it needed to exist.",
	},
}

var defaultProjects = []Project{
	{ID: "reagent", Title: "ReAgent", Summary: "An agentic retrosynthesis framework that plans reaction routes with evidence-grounded, multi-objective scoring and forward-validating filter-model checks.", URL: "https://github.com/rugbedbugg/ReAgent", Tags: []string{"Python", "Agentic", "LLM", "Scoring"}},
	{ID: "trionda-trifecta-26", Title: "Trionda-Trifecta-26", Summary: "A FIFA World Cup predictor with leakage-safe features, W/D/L and scoreline models, and a full-bracket 2026 simulation that calls Spain to lift the trophy.", URL: "https://github.com/rugbedbugg/Trionda-Trifecta-26", Tags: []string{"Python", "ML", "Modeling", "Simulation"}},
	{ID: "resonanceid-cli", Title: "ResonanceID-cli", Summary: "A Shazam-inspired Rust CLI that fingerprints WAV audio, ranks candidate matches, and backs everything with SQLite for fast, explainable lookup.", URL: "https://github.com/rugbedbugg/ResonanceID-cli", Tags: []string{"Rust", "DSP", "SQLite", "cli"}},
	{ID: "http-svr-200-ok", Title: "HTTP-SVR-200-OK", Summary: "A hand-rolled HTTP/1.0 server in x86_64 assembly for Linux, built to understand networking from first principles.", URL: "https://github.com/rugbedbugg/HTTP-SVR-200-OK", Tags: []string{"x86_64 Assembly", "Linux", "Networking", "HTTP"}},
}

var defaultPublications = []Publication{
	{ID: "chess960-fpga-sisimpact-2025", Title: "Resource-Efficient FPGA Realization of Chess960 Position Generator for Future Covert Communication Systems", Venue: "IEEE SISIMPACT 2025", Contribution: "Conceptualization, Methodology, Software", URL: "https://doi.org/10.1109/SISIMPACT67725.2025.11439749", Authors: []string{"Naman Goyal", "Partha Pratim Gogoi", "Abhishek Narayan Tripathi", "Naushad Manzoor Laskar"}},
}

var defaultLinks = []Link{
	{ID: "github", Label: "GitHub", URL: "https://github.com/rugbedbugg", Description: "Inspect the source code"},
	{ID: "linkedin", Label: "LinkedIn", URL: "https://www.linkedin.com/in/partha-gogoi-736241308/", Description: "Review the professional record"},
	{ID: "email", Label: "Email", URL: "mailto:yes.par781@gmail.com", Description: "Open a direct channel"},
}

// Default returns a complete portfolio with fresh slices on every call.
func Default() Portfolio {
	portfolio := Portfolio{
		Profile:      Profile{Name: defaultProfile.Name, Tagline: defaultProfile.Tagline, Biography: append([]string(nil), defaultProfile.Biography...)},
		Projects:     append([]Project(nil), defaultProjects...),
		Publications: append([]Publication(nil), defaultPublications...),
		Links:        append([]Link(nil), defaultLinks...),
	}
	for i := range portfolio.Projects {
		portfolio.Projects[i].Tags = append([]string(nil), defaultProjects[i].Tags...)
	}
	for i := range portfolio.Publications {
		portfolio.Publications[i].Authors = append([]string(nil), defaultPublications[i].Authors...)
	}
	return portfolio
}
