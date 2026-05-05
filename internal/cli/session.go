package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/ids"
	"github.com/ruohao1/penta/internal/reporting"
	"github.com/ruohao1/penta/internal/storage/sqlite"
	"github.com/ruohao1/penta/internal/viewmodel"
	"github.com/spf13/cobra"
)

func newSessionCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{Use: "session", Short: "Manage scanning sessions", SilenceUsage: true}
	cmd.AddCommand(newSessionCreateCommand(app), newSessionListCommand(app), newSessionShowCommand(app), newSessionArchiveCommand(app), newSessionScopeCommand(app))
	return cmd
}

func newSessionCreateCommand(app *App) *cobra.Command {
	var kind string
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a scanning session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sinks := commandSinks(cmd, app)
			if app == nil || app.DB == nil {
				return fmt.Errorf("database is not initialized")
			}
			sessionKind, err := parseSessionKind(kind)
			if err != nil {
				return err
			}
			now := time.Now()
			session := sqlite.Session{ID: ids.New(ids.PrefixSession), Name: args[0], Kind: sessionKind, Status: sqlite.SessionStatusActive, CreatedAt: now, UpdatedAt: now}
			if err := app.DB.CreateSession(cmd.Context(), session); err != nil {
				return err
			}
			sinks.Printf("Session created\nID      %s\nName    %s\nKind    %s\nStatus  %s\n", session.ID, session.Name, session.Kind, session.Status)
			return nil
		},
	}
	cmd.Flags().StringVar(&kind, "kind", string(sqlite.SessionKindOther), "session kind: bugbounty, ctf, pentest, lab, other")
	return cmd
}

func newSessionListCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List scanning sessions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			sinks := commandSinks(cmd, app)
			if app == nil || app.DB == nil {
				return fmt.Errorf("database is not initialized")
			}
			sessions, err := app.DB.ListSessions(cmd.Context())
			if err != nil {
				return err
			}
			if len(sessions) == 0 {
				sinks.Printf("No sessions\n")
				return nil
			}
			rows := make([][]string, 0, len(sessions))
			for _, session := range sessions {
				rows = append(rows, []string{session.ID, session.Name, string(session.Kind), string(session.Status)})
			}
			sinks.Printf("%s\n", renderSessionListTable(rows))
			return nil
		},
	}
}

func renderSessionListTable(rows [][]string) string {
	return table.New().
		Border(lipgloss.HiddenBorder()).
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderColumn(false).
		BorderRow(false).
		BorderHeader(false).
		StyleFunc(func(row, col int) lipgloss.Style {
			return lipgloss.NewStyle().PaddingRight(2)
		}).
		Headers("ID", "Name", "Kind", "Status").
		Rows(rows...).
		Render()
}

func newSessionShowCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "show <session-id>",
		Short: "Show a scanning session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sinks := commandSinks(cmd, app)
			if app == nil || app.DB == nil {
				return fmt.Errorf("database is not initialized")
			}
			summary, err := viewmodel.BuildSessionSummary(cmd.Context(), app.DB, args[0])
			if err != nil {
				return err
			}
			session := summary.Session
			sinks.Printf("ID        %s\nName      %s\nKind      %s\nStatus    %s\nRuns      %s\nTasks     %s\nEvidence  %s\n", session.ID, session.Name, session.Kind, session.Status, formatRunCounts(summary.RunCounts), reporting.FormatTaskCounts(summary.TaskCounts), reporting.FormatEvidenceCounts(summary.EvidenceCounts))
			if !summary.LatestRunAt.IsZero() {
				sinks.Printf("Latest   %s\n", summary.LatestRunAt.Format(time.RFC3339))
			}
			if len(summary.ScopeRules) > 0 {
				sinks.Printf("\nScope\n")
				for _, rule := range summary.ScopeRules {
					sinks.Printf("- %s %s %s (%s)\n", rule.Effect, rule.TargetType, rule.Value, rule.ID)
				}
			}
			if len(summary.Runs) > 0 {
				sinks.Printf("\nRuns\n")
				for _, run := range summary.Runs {
					sinks.Printf("- %s %s %s %s\n", run.ID, run.Mode, run.Status, run.CreatedAt.Format(time.RFC3339))
				}
			}
			return nil
		},
	}
}

func newSessionArchiveCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "archive <session-id>",
		Short: "Archive a scanning session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sinks := commandSinks(cmd, app)
			if app == nil || app.DB == nil {
				return fmt.Errorf("database is not initialized")
			}
			if err := app.DB.ArchiveSession(cmd.Context(), args[0], time.Now()); err != nil {
				return err
			}
			sinks.Printf("Session archived: %s\n", args[0])
			return nil
		},
	}
}

func newSessionScopeCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{Use: "scope", Short: "Manage session scope rules", SilenceUsage: true}
	cmd.AddCommand(newSessionScopeAddCommand(app), newSessionScopeListCommand(app), newSessionScopeRemoveCommand(app))
	return cmd
}

func newSessionScopeAddCommand(app *App) *cobra.Command {
	var include bool
	var exclude bool
	cmd := &cobra.Command{
		Use:   "add <session-id> <target-type> <value>",
		Short: "Add a session scope rule",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			sinks := commandSinks(cmd, app)
			if app == nil || app.DB == nil {
				return fmt.Errorf("database is not initialized")
			}
			effect, err := scopeEffectFromFlags(include, exclude)
			if err != nil {
				return err
			}
			targetType, err := parseScopeTargetType(args[1])
			if err != nil {
				return err
			}
			if _, err := app.DB.GetSession(cmd.Context(), args[0]); err != nil {
				return err
			}
			rule := sqlite.ScopeRule{ID: ids.New(ids.PrefixScope), SessionID: args[0], Effect: effect, TargetType: targetType, Value: args[2], CreatedAt: time.Now()}
			if err := app.DB.CreateScopeRule(cmd.Context(), rule); err != nil {
				return err
			}
			sinks.Printf("Scope rule added\nID      %s\nEffect  %s\nType    %s\nValue   %s\n", rule.ID, rule.Effect, rule.TargetType, rule.Value)
			return nil
		},
	}
	cmd.Flags().BoolVar(&include, "include", false, "add an include scope rule")
	cmd.Flags().BoolVar(&exclude, "exclude", false, "add an exclude scope rule")
	return cmd
}

func newSessionScopeListCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list <session-id>",
		Short: "List session scope rules",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sinks := commandSinks(cmd, app)
			if app == nil || app.DB == nil {
				return fmt.Errorf("database is not initialized")
			}
			rules, err := app.DB.ListScopeRulesBySession(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if len(rules) == 0 {
				sinks.Printf("No scope rules\n")
				return nil
			}
			rows := make([][]string, 0, len(rules))
			for _, rule := range rules {
				rows = append(rows, []string{rule.ID, string(rule.Effect), string(rule.TargetType), rule.Value})
			}
			sinks.Printf("%s\n", renderScopeRuleListTable(rows))
			return nil
		},
	}
}

func renderScopeRuleListTable(rows [][]string) string {
	return table.New().
		Border(lipgloss.HiddenBorder()).
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderColumn(false).
		BorderRow(false).
		BorderHeader(false).
		StyleFunc(func(row, col int) lipgloss.Style {
			return lipgloss.NewStyle().PaddingRight(2)
		}).
		Headers("ID", "Effect", "Type", "Value").
		Rows(rows...).
		Render()
}

func newSessionScopeRemoveCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:     "remove <rule-id>",
		Aliases: []string{"rm"},
		Short:   "Remove a session scope rule",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sinks := commandSinks(cmd, app)
			if app == nil || app.DB == nil {
				return fmt.Errorf("database is not initialized")
			}
			if err := app.DB.DeleteScopeRule(cmd.Context(), args[0]); err != nil {
				return err
			}
			sinks.Printf("Scope rule removed: %s\n", args[0])
			return nil
		},
	}
}

func parseSessionKind(value string) (sqlite.SessionKind, error) {
	switch sqlite.SessionKind(strings.ToLower(value)) {
	case sqlite.SessionKindBugBounty, sqlite.SessionKindCTF, sqlite.SessionKindPentest, sqlite.SessionKindLab, sqlite.SessionKindOther:
		return sqlite.SessionKind(strings.ToLower(value)), nil
	default:
		return "", fmt.Errorf("invalid session kind: %s", value)
	}
}

func parseScopeTargetType(value string) (sqlite.ScopeTargetType, error) {
	switch sqlite.ScopeTargetType(strings.ToLower(value)) {
	case sqlite.ScopeTargetDomain, sqlite.ScopeTargetIP, sqlite.ScopeTargetCIDR, sqlite.ScopeTargetURL, sqlite.ScopeTargetService, sqlite.ScopeTargetWildcard:
		return sqlite.ScopeTargetType(strings.ToLower(value)), nil
	default:
		return "", fmt.Errorf("invalid scope target type: %s", value)
	}
}

func scopeEffectFromFlags(include, exclude bool) (sqlite.ScopeEffect, error) {
	if include == exclude {
		return "", fmt.Errorf("set exactly one of --include or --exclude")
	}
	if include {
		return sqlite.ScopeEffectInclude, nil
	}
	return sqlite.ScopeEffectExclude, nil
}

func formatRunCounts(counts map[actions.RunStatus]int) string {
	return fmt.Sprintf("%d completed / %d failed / %d running / %d pending", counts[actions.RunStatusCompleted], counts[actions.RunStatusFailed], counts[actions.RunStatusRunning], counts[actions.RunStatusPending])
}
