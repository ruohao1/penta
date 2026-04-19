package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/storage/sqlite"
	"github.com/spf13/cobra"
)

var taskExecutor = executeTask

func newReconCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recon",
		Short: "Run recon commands",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReconCommand(cmd, app, args[0])
		},
	}

	return cmd
}

func runReconCommand(cmd *cobra.Command, app *App, target string) error {
	if app == nil || app.DB == nil {
		return fmt.Errorf("database is not initialized")
	}

	runID, taskID, err := seedRecon(cmd, app, target)
	if err != nil {
		return err
	}

	if err := taskExecutor(cmd, app, taskID); err != nil {
		if updateErr := app.DB.UpdateRunStatus(cmd.Context(), runID, actions.RunStatusFailed); updateErr != nil {
			return fmt.Errorf("%w: mark run failed: %v", err, updateErr)
		}
		return err
	}

	if err := app.DB.UpdateTaskStatus(cmd.Context(), taskID, actions.TaskStatusCompleted); err != nil {
		return err
	}
	if err := app.DB.UpdateRunStatus(cmd.Context(), runID, actions.RunStatusCompleted); err != nil {
		return err
	}

	fmt.Printf("Recon completed for target: %s\n", target)

	return nil
}

func seedRecon(cmd *cobra.Command, app *App, target string) (string, string, error) {
	runID := "run_" + generateID()
	run := sqlite.Run{
		ID:        runID,
		Mode:      "recon",
		Status:    actions.RunStatusPending,
		CreatedAt: time.Now(),
	}
	if err := app.DB.CreateRun(cmd.Context(), run); err != nil {
		return "", "", err
	}
	if err := app.DB.UpdateRunStatus(cmd.Context(), run.ID, actions.RunStatusRunning); err != nil {
		return "", "", err
	}

	inputJSON, err := json.Marshal(actions.SeedTargetInput{Raw: target})
	if err != nil {
		return "", "", err
	}

	taskID := "task_" + generateID()
	task := sqlite.Task{
		ID:         taskID,
		RunID:      runID,
		ActionType: actions.ActionSeedTarget,
		InputJSON:  string(inputJSON),
		Status:     actions.TaskStatusPending,
		CreatedAt:  time.Now(),
	}
	if err := app.DB.CreateTask(cmd.Context(), task); err != nil {
		return "", "", err
	}

	return runID, taskID, nil
}
