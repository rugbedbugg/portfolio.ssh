// Package command parses the portfolio's fixed, non-shell command language.
package command

import (
	"sort"
	"strings"

	"github.com/rugbedbugg/portfolio.ssh/internal/content"
)

// Kind identifies a parsed portfolio command.
type Kind uint8

const (
	Unknown Kind = iota
	Help
	About
	Projects
	Project
	Research
	Dispatches
	Contact
	Open
	Clear
	Exit
)

// Result is the side-effect-free result of parsing one command line.
type Result struct {
	Kind        Kind
	Target      string
	Err         string
	Suggestions []string
}

// Completion contains the completed input and the matching candidates.
type Completion struct {
	Input      string
	Candidates []string
}

var commandNames = []string{
	"about",
	"clear",
	"contact",
	"dispatches",
	"exit",
	"help",
	"ls",
	"open",
	"project",
	"projects",
	"research",
	"whoami",
}

// Parse parses only commands in the portfolio command language. It never
// invokes a shell, starts a process, or performs I/O.
func Parse(input string, data content.Portfolio) Result {
	fields := strings.Fields(strings.ToLower(input))
	if len(fields) == 0 {
		return Result{}
	}

	switch fields[0] {
	case "help":
		return noArgumentResult(Help, fields)
	case "about", "whoami":
		return noArgumentResult(About, fields)
	case "projects", "ls":
		return noArgumentResult(Projects, fields)
	case "research":
		return noArgumentResult(Research, fields)
	case "dispatches":
		return noArgumentResult(Dispatches, fields)
	case "contact":
		return noArgumentResult(Contact, fields)
	case "clear":
		return noArgumentResult(Clear, fields)
	case "exit":
		return noArgumentResult(Exit, fields)
	case "project":
		if len(fields) != 2 {
			return Result{Kind: Project, Err: "project expects exactly one project ID or unique prefix."}
		}
		ids := projectIDs(data)
		target, suggestions := resolveID(fields[1], ids)
		if target == "" {
			if len(suggestions) > 1 {
				return Result{Kind: Project, Err: "project prefix is ambiguous; choose one of the suggested IDs.", Suggestions: suggestions}
			}
			if len(suggestions) == 0 {
				suggestions = closestIDs(fields[1], ids)
			}
			return Result{Kind: Project, Err: "unknown project; use a complete ID or unique prefix.", Suggestions: suggestions}
		}
		return Result{Kind: Project, Target: target}
	case "open":
		if len(fields) != 2 {
			return Result{Kind: Open, Err: "open expects exactly one link ID or unique prefix."}
		}
		target, suggestions := resolveID(fields[1], linkIDs(data))
		if target == "" {
			if len(suggestions) > 1 {
				return Result{Kind: Open, Err: "link prefix is ambiguous; choose one of the suggested IDs.", Suggestions: suggestions}
			}
			return Result{Kind: Open, Err: "unknown link; use a complete ID or unique prefix.", Suggestions: suggestions}
		}
		return Result{Kind: Open, Target: target}
	default:
		return Result{Err: "unknown command; try `help` for the available commands.", Suggestions: []string{"help"}}
	}
}

func noArgumentResult(kind Kind, fields []string) Result {
	if len(fields) != 1 {
		return Result{Kind: kind, Err: "this command does not accept arguments; try `help`."}
	}
	return Result{Kind: kind}
}

func projectIDs(data content.Portfolio) []string {
	ids := make([]string, 0, len(data.Projects))
	for _, project := range data.Projects {
		ids = append(ids, project.ID)
	}
	return ids
}

func linkIDs(data content.Portfolio) []string {
	ids := make([]string, 0, len(data.Links))
	for _, link := range data.Links {
		ids = append(ids, link.ID)
	}
	return ids
}

func resolveID(query string, ids []string) (string, []string) {
	query = strings.ToLower(query)
	for _, id := range ids {
		if strings.ToLower(id) == query {
			return id, nil
		}
	}

	matches := make([]string, 0)
	for _, id := range ids {
		if strings.HasPrefix(strings.ToLower(id), query) {
			matches = append(matches, id)
		}
	}
	sort.Strings(matches)
	if len(matches) == 1 {
		return matches[0], nil
	}
	return "", matches
}

func closestIDs(query string, ids []string) []string {
	queryRunes := []rune(strings.ToLower(query))
	limit := closeMatchDistanceLimit(len(queryRunes))
	bestDistance := limit + 1
	matches := make([]string, 0)
	for _, id := range ids {
		distance := editDistance(queryRunes, []rune(strings.ToLower(id)))
		switch {
		case distance > limit || distance > bestDistance:
			continue
		case distance < bestDistance:
			bestDistance = distance
			matches = matches[:0]
		}
		matches = append(matches, id)
	}
	sort.Strings(matches)
	return matches
}

func closeMatchDistanceLimit(length int) int {
	switch {
	case length <= 4:
		return 1
	case length <= 8:
		return 2
	default:
		return 3
	}
}

func editDistance(left, right []rune) int {
	previous := make([]int, len(right)+1)
	for index := range previous {
		previous[index] = index
	}

	for leftIndex, leftRune := range left {
		current := make([]int, len(right)+1)
		current[0] = leftIndex + 1
		for rightIndex, rightRune := range right {
			cost := 0
			if leftRune != rightRune {
				cost = 1
			}
			current[rightIndex+1] = min(
				current[rightIndex]+1,
				previous[rightIndex+1]+1,
				previous[rightIndex]+cost,
			)
		}
		previous = current
	}
	return previous[len(right)]
}

// Complete completes a command name or a project/link ID. A partial command
// with multiple matches is advanced only through their shared prefix.
func Complete(input string, data content.Portfolio) Completion {
	fields := strings.Fields(strings.ToLower(input))
	if len(fields) == 0 {
		return Completion{Input: input, Candidates: append([]string(nil), commandNames...)}
	}

	if len(fields) == 1 {
		if hasTrailingWhitespace(input) && (fields[0] == "project" || fields[0] == "open") {
			if fields[0] == "project" {
				return completeTarget(fields[0], "", projectIDs(data), input)
			}
			return completeTarget(fields[0], "", linkIDs(data), input)
		}
		matches := prefixMatches(fields[0], commandNames)
		if len(matches) == 0 {
			return Completion{Input: input}
		}
		if len(matches) == 1 {
			return Completion{Input: matches[0] + " ", Candidates: matches}
		}
		return Completion{Input: commonPrefix(fields[0], matches), Candidates: matches}
	}

	var ids []string
	switch fields[0] {
	case "project":
		ids = projectIDs(data)
	case "open":
		ids = linkIDs(data)
	default:
		return Completion{Input: input}
	}

	return completeTarget(fields[0], fields[len(fields)-1], ids, input)
}

func completeTarget(command, prefix string, ids []string, original string) Completion {
	matches := prefixMatches(prefix, ids)
	if len(matches) == 0 {
		return Completion{Input: original}
	}
	if len(matches) == 1 {
		return Completion{Input: command + " " + matches[0] + " ", Candidates: matches}
	}
	return Completion{Input: command + " " + commonPrefix(prefix, matches), Candidates: matches}
}

func hasTrailingWhitespace(input string) bool {
	return len(input) > 0 && (input[len(input)-1] == ' ' || input[len(input)-1] == '\t' || input[len(input)-1] == '\n' || input[len(input)-1] == '\r')
}

func prefixMatches(prefix string, values []string) []string {
	prefix = strings.ToLower(prefix)
	matches := make([]string, 0)
	for _, value := range values {
		if strings.HasPrefix(strings.ToLower(value), prefix) {
			matches = append(matches, value)
		}
	}
	sort.Strings(matches)
	return matches
}

func commonPrefix(prefix string, matches []string) string {
	if len(matches) == 0 {
		return prefix
	}
	result := []rune(strings.ToLower(matches[0]))
	for _, match := range matches[1:] {
		candidate := []rune(strings.ToLower(match))
		limit := len(result)
		if len(candidate) < limit {
			limit = len(candidate)
		}
		for i := 0; i < limit; i++ {
			if result[i] != candidate[i] {
				limit = i
				break
			}
		}
		result = result[:limit]
	}
	if len(result) < len([]rune(prefix)) {
		return prefix
	}
	return string(result)
}
