package cli

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/ruohao1/penta/internal/viewmodel"
	"github.com/spf13/cobra"
)

func newArtifactsCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{Use: "artifacts", Short: "Inspect run artifacts", SilenceUsage: true}
	cmd.AddCommand(newArtifactsListCommand(app), newArtifactsShowCommand(app))
	return cmd
}

func newArtifactsListCommand(app *App) *cobra.Command {
	var runSelector string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List artifact metadata for a run",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			sinks := commandSinks(cmd, app)
			if app == nil || app.DB == nil {
				return fmt.Errorf("database is not initialized")
			}
			list, err := viewmodel.BuildArtifactList(cmd.Context(), app.DB, runSelector)
			if err != nil {
				return err
			}
			sinks.Printf("Run %s\n\n", viewmodel.FormatRunContext(list.Run.ID, list.Latest))
			if len(list.Artifacts) == 0 {
				sinks.Printf("No artifacts\n")
				return nil
			}
			rows := make([][]string, 0, len(list.Artifacts))
			for _, item := range list.Artifacts {
				rows = append(rows, []string{fmt.Sprintf("%d", item.Index), item.Kind, item.Source, item.Row.Path})
			}
			sinks.Printf("%s\n", renderArtifactListTable(rows))
			return nil
		},
	}
	cmd.Flags().StringVar(&runSelector, "run", "latest", "run id or latest")
	registerRunFlagCompletion(cmd, app)
	return cmd
}

func renderArtifactListTable(rows [][]string) string {
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
		Headers("#", "Kind", "Source", "Path").
		Rows(rows...).
		Render()
}

func newArtifactsShowCommand(app *App) *cobra.Command {
	var runSelector string
	cmd := &cobra.Command{
		Use:   "show <index|selector|id>",
		Short: "Show artifact metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sinks := commandSinks(cmd, app)
			if app == nil || app.DB == nil {
				return fmt.Errorf("database is not initialized")
			}
			list, item, err := viewmodel.ResolveArtifact(cmd.Context(), app.DB, runSelector, args[0])
			if err != nil {
				return err
			}
			sinks.Printf("Run       %s\n", viewmodel.FormatRunContext(list.Run.ID, list.Latest))
			sinks.Printf("Index     %d\n", item.Index)
			sinks.Printf("ID        %s\n", item.Row.ID)
			sinks.Printf("Task      %s\n", item.Task.ID)
			sinks.Printf("Kind      %s\n", item.Kind)
			if item.Source != "" {
				sinks.Printf("Source    %s\n", item.Source)
			}
			sinks.Printf("Path      %s\n", item.Row.Path)
			sinks.Printf("Created   %s\n", item.Row.CreatedAt.Format(time.RFC3339))
			return nil
		},
	}
	cmd.Flags().StringVar(&runSelector, "run", "latest", "run id or latest")
	registerRunFlagCompletion(cmd, app)
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeArtifactSelectors(cmd, app, runSelector, args, toComplete)
	}
	return cmd
}
