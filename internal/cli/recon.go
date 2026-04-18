package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ruohao1/penta/internal/storage/sqlite"
	"github.com/spf13/cobra"
)

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

	inputJSON, err := json.Marshal(map[string]string{"target": target})
	if err != nil {
		return err
	}

	evidenceJSON, err := json.Marshal(map[string]string{"output": "recon output data"})
	if err != nil {
		return err
	}

	runID := "run_" + generateID()
	run := sqlite.Run{
		ID:        runID,
		Mode:      "recon",
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	if err := app.DB.CreateRun(cmd.Context(), run); err != nil {
		return err
	}
	if err := app.DB.UpdateRunStatus(cmd.Context(), run.ID, "running"); err != nil {
		return err
	}

	taskID := "task_" + generateID()
	task := sqlite.Task{
		ID:         taskID,
		RunID:      runID,
		ActionType: "recon",
		InputJSON:  string(inputJSON),
		Status:     "pending",
		CreatedAt:  time.Now(),
	}
	if err := app.DB.CreateTask(cmd.Context(), task); err != nil {
		return err
	}
	if err := app.DB.UpdateTaskStatus(cmd.Context(), task.ID, "running"); err != nil {
		return err
	}

	artifactID := "artifact_" + generateID()
	artifact := sqlite.Artifact{
		ID:        artifactID,
		TaskID:    taskID,
		Path:      "recon_output.txt",
		CreatedAt: time.Now(),
	}
	if err := app.DB.CreateArtifact(cmd.Context(), artifact); err != nil {
		return err
	}

	evidenceID := "evidence_" + generateID()
	evidence := sqlite.Evidence{
		ID:        evidenceID,
		RunID:     runID,
		Kind:      "recon_output",
		DataJSON:  string(evidenceJSON),
		CreatedAt: time.Now(),
	}
	if err := app.DB.CreateEvidence(cmd.Context(), evidence); err != nil {
		return err
	}

	if err := app.DB.UpdateTaskStatus(cmd.Context(), task.ID, "completed"); err != nil {
		return err
	}
	if err := app.DB.UpdateRunStatus(cmd.Context(), run.ID, "completed"); err != nil {
		return err
	}

	fmt.Printf("Recon completed for target: %s\n", target)

	return nil
}
