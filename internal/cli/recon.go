package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/events"
	"github.com/ruohao1/penta/internal/execute"
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

	runID, err := createRun(cmd, app)
	if err != nil {
		return err
	}
	sink := &events.SQLiteSink{DB: app.DB}
	executor := &execute.Executor{DB: app.DB, RunID: runID, Events: sink}
	if err := sink.Append(cmd.Context(), events.Event{
		RunID:       runID,
		EventType:   events.EventRunCreated,
		EntityKind:  events.EntityRun,
		EntityID:    runID,
		PayloadJSON: mustPayloadJSON(events.RunCreatedPayload{Mode: "recon"}),
		CreatedAt:   time.Now(),
	}); err != nil {
		return err
	}
	if err := executor.Resolve(cmd.Context(), runID, execute.Request{Action: actions.ActionProbeHTTP, Raw: target}); err != nil {
		if updateErr := app.DB.UpdateRunStatus(cmd.Context(), runID, actions.RunStatusFailed); updateErr != nil {
			return fmt.Errorf("%w: mark run failed: %v", err, updateErr)
		}
		_ = sink.Append(cmd.Context(), events.Event{RunID: runID, EventType: events.EventRunFailed, EntityKind: events.EntityRun, EntityID: runID, PayloadJSON: mustPayloadJSON(map[string]string{"error": err.Error()}), CreatedAt: time.Now()})
		return err
	}

	if err := executor.RunUntilIdle(cmd.Context()); err != nil {
		if updateErr := app.DB.UpdateRunStatus(cmd.Context(), runID, actions.RunStatusFailed); updateErr != nil {
			return fmt.Errorf("%w: mark run failed: %v", err, updateErr)
		}
		_ = sink.Append(cmd.Context(), events.Event{RunID: runID, EventType: events.EventRunFailed, EntityKind: events.EntityRun, EntityID: runID, PayloadJSON: mustPayloadJSON(map[string]string{"error": err.Error()}), CreatedAt: time.Now()})
		return err
	}
	if err := app.DB.UpdateRunStatus(cmd.Context(), runID, actions.RunStatusCompleted); err != nil {
		return err
	}
	if err := sink.Append(cmd.Context(), events.Event{RunID: runID, EventType: events.EventRunCompleted, EntityKind: events.EntityRun, EntityID: runID, PayloadJSON: mustPayloadJSON(map[string]string{}), CreatedAt: time.Now()}); err != nil {
		return err
	}

	fmt.Printf("Recon completed for target: %s\n", target)

	return nil
}

func createRun(cmd *cobra.Command, app *App) (string, error) {
	runID := "run_" + generateID()
	run := sqlite.Run{
		ID:        runID,
		Mode:      "recon",
		Status:    actions.RunStatusPending,
		CreatedAt: time.Now(),
	}
	if err := app.DB.CreateRun(cmd.Context(), run); err != nil {
		return "", err
	}
	if err := app.DB.UpdateRunStatus(cmd.Context(), run.ID, actions.RunStatusRunning); err != nil {
		return "", err
	}
	return runID, nil
}

func mustPayloadJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}
