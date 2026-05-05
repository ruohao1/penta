package cli

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/ruohao1/penta/internal/viewmodel"
	"github.com/spf13/cobra"
)

func newEvidenceCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{Use: "evidence", Short: "Inspect run evidence", SilenceUsage: true}
	cmd.AddCommand(newEvidenceListCommand(app), newEvidenceShowCommand(app))
	return cmd
}

func newEvidenceListCommand(app *App) *cobra.Command {
	var runSelector string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List evidence for a run",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			sinks := commandSinks(cmd, app)
			if app == nil || app.DB == nil {
				return fmt.Errorf("database is not initialized")
			}
			list, err := viewmodel.BuildEvidenceList(cmd.Context(), app.DB, runSelector)
			if err != nil {
				return err
			}
			sinks.Printf("Run %s\n\n", viewmodel.FormatRunContext(list.Run.ID, list.Latest))
			if len(list.Evidence) == 0 {
				sinks.Printf("No evidence\n")
				return nil
			}
			rows := make([][]string, 0, len(list.Evidence))
			for _, item := range list.Evidence {
				rows = append(rows, []string{fmt.Sprintf("%d", item.Index), item.Kind, item.Label})
			}
			sinks.Printf("%s\n", renderEvidenceListTable(rows))
			return nil
		},
	}
	cmd.Flags().StringVar(&runSelector, "run", "latest", "run id or latest")
	registerRunFlagCompletion(cmd, app)
	return cmd
}

func renderEvidenceListTable(rows [][]string) string {
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
		Headers("#", "Kind", "Label").
		Rows(rows...).
		Render()
}

func newEvidenceShowCommand(app *App) *cobra.Command {
	var runSelector string
	cmd := &cobra.Command{
		Use:   "show <index|selector|id>",
		Short: "Show one evidence item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sinks := commandSinks(cmd, app)
			if app == nil || app.DB == nil {
				return fmt.Errorf("database is not initialized")
			}
			list, item, err := viewmodel.ResolveEvidence(cmd.Context(), app.DB, runSelector, args[0])
			if err != nil {
				return err
			}
			sinks.Printf("Run       %s\n", viewmodel.FormatRunContext(list.Run.ID, list.Latest))
			sinks.Printf("Index     %d\n", item.Index)
			sinks.Printf("ID        %s\n", item.ID)
			sinks.Printf("Kind      %s\n", item.Kind)
			sinks.Printf("Label     %s\n", item.Label)
			if len(item.Details) > 0 {
				sinks.Printf("\nDetails\n")
				for _, detail := range item.Details {
					sinks.Printf("- %s\n", detail)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&runSelector, "run", "latest", "run id or latest")
	registerRunFlagCompletion(cmd, app)
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeEvidenceSelectors(cmd, app, runSelector, args, toComplete)
	}
	return cmd
}
