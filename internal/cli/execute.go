package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/storage/sqlite"
	"github.com/ruohao1/penta/internal/targets"
	"github.com/spf13/cobra"
)

func executeTask(cmd *cobra.Command, app *App, taskID string) error {
	task, err := app.DB.GetTask(cmd.Context(), taskID)
	if err != nil {
		return err
	}

	if err := app.DB.UpdateTaskStatus(cmd.Context(), taskID, actions.TaskStatusRunning); err != nil {
		return err
	}

	var execErr error
	switch task.ActionType {
	case actions.ActionSeedTarget:
		execErr = executeSeedTarget(cmd, app, task)
	default:
		execErr = fmt.Errorf("unsupported action type: %s", task.ActionType)
	}

	if execErr != nil {
		if err := app.DB.UpdateTaskStatus(cmd.Context(), taskID, actions.TaskStatusFailed); err != nil {
			return fmt.Errorf("%w: mark task failed: %v", execErr, err)
		}
		return execErr
	}

	return nil
}

func executeSeedTarget(cmd *cobra.Command, app *App, task *sqlite.Task) error {
	input := make(map[string]string)
	if err := json.Unmarshal([]byte(task.InputJSON), &input); err != nil {
		return err
	}

	target, err := targets.Parse(input["target"])
	if err != nil {
		return err
	}

	evidenceData := map[string]string{
		"value": target.String(),
		"type":  string(target.Type()),
	}

	evidenceJSON, err := json.Marshal(evidenceData)
	if err != nil {
		return err
	}

	evidenceID := "evidence_" + generateID()
	evidence := sqlite.Evidence{
		ID:        evidenceID,
		RunID:     task.RunID,
		Kind:      "target",
		DataJSON:  string(evidenceJSON),
		CreatedAt: time.Now(),
	}
	if err := app.DB.CreateEvidence(cmd.Context(), evidence); err != nil {
		return err
	}

	return nil
}
