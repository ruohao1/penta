package cli

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/ruohao1/penta/internal/ids"
	"github.com/ruohao1/penta/internal/viewmodel"
	"github.com/spf13/cobra"
)

func registerRunFlagCompletion(cmd *cobra.Command, app *App) {
	_ = cmd.RegisterFlagCompletionFunc("run", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeRuns(cmd, app, toComplete)
	})
}

func completeRuns(cmd *cobra.Command, app *App, toComplete string) ([]string, cobra.ShellCompDirective) {
	if app == nil || app.DB == nil {
		return nil, completionNoFileKeepOrder
	}

	list, err := viewmodel.BuildRunList(commandContext(cmd), app.DB)
	if err != nil {
		return nil, completionNoFileKeepOrder
	}

	suggestions := make([]string, 0, len(list.Runs)+1)
	addCompletion(&suggestions, toComplete, "latest", "latest run")
	for _, run := range list.Runs {
		description := strings.TrimSpace(fmt.Sprintf("%s %s", run.Status, run.Mode))
		if run.Session != "" && run.Session != "-" {
			description += " " + run.Session
		}
		addCompletion(&suggestions, toComplete, run.ID, description)
	}
	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

func completeEvidenceSelectors(cmd *cobra.Command, app *App, runSelector string, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 || app == nil || app.DB == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	list, err := viewmodel.BuildEvidenceList(commandContext(cmd), app.DB, runSelector)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	candidates := make([]completionCandidate, 0, len(list.Evidence)*3)
	seen := map[string]bool{}
	for _, item := range list.Evidence {
		addUniqueCompletionCandidate(&candidates, seen, toComplete, completionCandidate{value: fmt.Sprintf("%d", item.Index), description: strings.TrimSpace(item.Kind + " " + item.Label), kind: completionKindIndex, numeric: item.Index})
		if selector := evidenceSemanticSelector(item); completingEvidenceSemanticSelector(toComplete, selector) {
			addUniqueCompletionCandidate(&candidates, seen, toComplete, completionCandidate{value: selector, description: "semantic selector", kind: completionKindSelector})
		}
		if completingEvidenceID(toComplete) {
			addUniqueCompletionCandidate(&candidates, seen, toComplete, completionCandidate{value: item.ID, description: "exact evidence id", kind: completionKindID})
		}
	}
	return renderCompletionCandidates(candidates), completionNoFileKeepOrder
}

func completeArtifactSelectors(cmd *cobra.Command, app *App, runSelector string, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 || app == nil || app.DB == nil {
		return nil, completionNoFileKeepOrder
	}

	list, err := viewmodel.BuildArtifactList(commandContext(cmd), app.DB, runSelector)
	if err != nil {
		return nil, completionNoFileKeepOrder
	}

	candidates := make([]completionCandidate, 0, len(list.Artifacts)*3)
	seen := map[string]bool{}
	for _, item := range list.Artifacts {
		addUniqueCompletionCandidate(&candidates, seen, toComplete, completionCandidate{value: fmt.Sprintf("%d", item.Index), description: artifactIndexDescription(item), kind: completionKindIndex, numeric: item.Index})
		if item.Kind == "body" && item.Source != "" && completingArtifactSemanticSelector(toComplete) {
			addUniqueCompletionCandidate(&candidates, seen, toComplete, completionCandidate{value: "body:" + item.Source, description: "semantic selector", kind: completionKindSelector})
		}
		if completingArtifactID(toComplete) {
			addUniqueCompletionCandidate(&candidates, seen, toComplete, completionCandidate{value: item.Row.ID, description: "exact artifact id", kind: completionKindID})
		}
	}
	return renderCompletionCandidates(candidates), completionNoFileKeepOrder
}

const completionNoFileKeepOrder = cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder

func evidenceSemanticSelector(item viewmodel.IndexedEvidence) string {
	switch item.Kind {
	case "http_response":
		value := cliURLPath(item.URL)
		if value == "" {
			value = item.URL
		}
		if value == "" {
			return ""
		}
		return "http_response:" + value
	case "service":
		value := item.URL
		if value == "" {
			value = item.Label
		}
		if value == "" {
			return ""
		}
		return "service:" + value
	case "crawl":
		value := cliURLPath(item.URL)
		if value == "" {
			value = item.URL
		}
		if value == "" {
			return ""
		}
		return "crawl:" + value
	default:
		return ""
	}
}

func artifactIndexDescription(item viewmodel.IndexedArtifact) string {
	if item.Source != "" {
		return strings.TrimSpace(item.Kind + " " + item.Source)
	}
	if item.Row.Path != "" {
		return strings.TrimSpace(item.Kind + " " + item.Row.Path)
	}
	return item.Kind
}

func cliURLPath(value string) string {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return value
	}
	path := parsed.EscapedPath()
	if path == "" {
		path = "/"
	}
	if parsed.RawQuery != "" {
		path += "?" + parsed.RawQuery
	}
	return path
}

func commandContext(cmd *cobra.Command) context.Context {
	if cmd == nil || cmd.Context() == nil {
		return context.Background()
	}
	return cmd.Context()
}

func addUniqueCompletion(suggestions *[]string, seen map[string]bool, toComplete, value, description string) {
	if seen[value] {
		return
	}
	seen[value] = true
	addCompletion(suggestions, toComplete, value, description)
}

func addCompletion(suggestions *[]string, toComplete, value, description string) {
	if toComplete != "" && !strings.HasPrefix(value, toComplete) {
		return
	}
	if description == "" {
		*suggestions = append(*suggestions, value)
		return
	}
	*suggestions = append(*suggestions, value+"\t"+description)
}

type completionKind int

const (
	completionKindIndex completionKind = iota
	completionKindSelector
	completionKindID
)

type completionCandidate struct {
	value       string
	description string
	kind        completionKind
	numeric     int
}

func addUniqueCompletionCandidate(candidates *[]completionCandidate, seen map[string]bool, toComplete string, candidate completionCandidate) {
	if seen[candidate.value] {
		return
	}
	seen[candidate.value] = true
	if toComplete != "" && !strings.HasPrefix(candidate.value, toComplete) {
		return
	}
	*candidates = append(*candidates, candidate)
}

func renderCompletionCandidates(candidates []completionCandidate) []string {
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].kind != candidates[j].kind {
			return candidates[i].kind < candidates[j].kind
		}
		if candidates[i].kind == completionKindIndex {
			return candidates[i].numeric < candidates[j].numeric
		}
		return candidates[i].value < candidates[j].value
	})
	suggestions := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.description == "" {
			suggestions = append(suggestions, candidate.value)
			continue
		}
		suggestions = append(suggestions, candidate.value+"\t"+candidate.description)
	}
	return suggestions
}

func completingEvidenceID(toComplete string) bool {
	return strings.HasPrefix(toComplete, ids.PrefixEvidence) || strings.HasPrefix(toComplete, ids.LegacyPrefixEvidence)
}

func completingArtifactID(toComplete string) bool {
	return strings.HasPrefix(toComplete, ids.PrefixArtifact) || strings.HasPrefix(toComplete, ids.LegacyPrefixArtifact)
}

func completingEvidenceSemanticSelector(toComplete, selector string) bool {
	if toComplete == "" {
		return false
	}
	return strings.HasPrefix(selector, toComplete)
}

func completingArtifactSemanticSelector(toComplete string) bool {
	return toComplete != ""
}
