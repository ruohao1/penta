package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
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
	case actions.ActionProbeHTTP:
		execErr = executeProbeHTTP(cmd, app, task)
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

func executeProbeHTTP(cmd *cobra.Command, app *App, task *sqlite.Task) error {
	var input actions.ProbeHTTPInput
	if err := json.Unmarshal([]byte(task.InputJSON), &input); err != nil {
		return err
	}
	service := actions.ServiceEvidence{}
	switch input.Type {
	case targets.TypeURL:
		parsed, err := targets.Parse(input.Value)
		if err != nil {
			return err
		}
		urlTarget, ok := parsed.(*targets.URL)
		if !ok {
			return fmt.Errorf("expected url target")
		}
		service.Host = urlTarget.Host
		service.Scheme = urlTarget.Scheme
		service.Port, err = defaultPort(urlTarget.Scheme, urlTarget.Port)
		if err != nil {
			return err
		}
	case targets.TypeDomain, targets.TypeIP:
		service.Host = input.Value
		service.Scheme = "https"
		service.Port = 443
	default:
		return fmt.Errorf("unsupported target type: %s", input.Type)
	}

	evidenceJSON, err := json.Marshal(service)
	if err != nil {
		return err
	}

	evidenceID := "evidence_" + generateID()
	evidence := sqlite.Evidence{
		ID:        evidenceID,
		RunID:     task.RunID,
		Kind:      "service",
		DataJSON:  string(evidenceJSON),
		CreatedAt: time.Now(),
	}
	if err := app.DB.CreateEvidence(cmd.Context(), evidence); err != nil {
		return err
	}
	return nil
}

func defaultPort(scheme, port string) (int, error) {
	if port != "" {
		parsed, err := strconv.Atoi(port)
		if err != nil {
			return 0, fmt.Errorf("invalid port %q", port)
		}
		if parsed < 1 || parsed > 65535 {
			return 0, fmt.Errorf("port out of range: %d", parsed)
		}
		return parsed, nil
	}
	switch scheme {
	case "http":
		return 80, nil
	case "https":
		return 443, nil
	default:
		return 0, fmt.Errorf("unsupported scheme %q", scheme)
	}
}
func executeSeedTarget(cmd *cobra.Command, app *App, task *sqlite.Task) error {
	var input actions.SeedTargetInput
	if err := json.Unmarshal([]byte(task.InputJSON), &input); err != nil {
		return err
	}

	target, err := targets.Parse(input.Raw)
	if err != nil {
		return err
	}

	evidenceData := actions.SeedTargetEvidence{
		Value: target.String(),
		Type:  target.Type(),
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
