package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/ruohao1/penta/internal/viewmodel"
	"github.com/spf13/cobra"
)

func newRunsCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{Use: "runs", Short: "Inspect runs", SilenceUsage: true}
	cmd.AddCommand(newRunsListCommand(app))
	return cmd
}

func newRunsListCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List runs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			sinks := commandSinks(cmd, app)
			if app == nil || app.DB == nil {
				return fmt.Errorf("database is not initialized")
			}

			list, err := viewmodel.BuildRunList(cmd.Context(), app.DB)
			if err != nil {
				return err
			}
			if len(list.Runs) == 0 {
				sinks.Printf("No runs\n")
				return nil
			}

			rows := make([][]string, 0, len(list.Runs))
			for _, run := range list.Runs {
				rows = append(rows, []string{fmt.Sprintf("%d", run.Index), displayRunID(run.ID), run.Mode, string(run.Status), run.Session, run.CreatedAt.Format(time.RFC3339)})
			}
			sinks.Printf("%s\n", renderRunsTable(rows))
			return nil
		},
	}
}

func renderRunsTable(rows [][]string) string {
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
		Headers("#", "Run", "Mode", "Status", "Session", "Created").
		Rows(rows...).
		Render()
}

func displayRunID(id string) string {
	const width = len("run_") + 10
	if len(id) <= width {
		return id
	}
	if strings.HasPrefix(id, "run_") {
		token := strings.ReplaceAll(strings.TrimPrefix(id, "run_"), "-", "")
		if len(token) <= 10 {
			return "run_" + token
		}
		return "run_" + token[:9] + "…"
	}
	return id[:width-1] + "…"
}
