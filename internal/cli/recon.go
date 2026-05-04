package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/events"
	"github.com/ruohao1/penta/internal/execute"
	"github.com/ruohao1/penta/internal/reporting"
	"github.com/ruohao1/penta/internal/scope"
	"github.com/ruohao1/penta/internal/storage/sqlite"
	"github.com/ruohao1/penta/internal/targets"
	"github.com/ruohao1/penta/internal/viewmodel"
	"github.com/spf13/cobra"
)

func newReconCommand(app *App) *cobra.Command {
	var verboseCount int
	var quiet bool
	var noColor bool
	var outputPath string
	var sessionID string

	cmd := &cobra.Command{
		Use:          "recon",
		Short:        "Run recon commands",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReconCommand(cmd, app, args[0])
		},
	}
	cmd.Flags().CountVarP(&verboseCount, "verbose", "v", "increase output verbosity")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "only print final status and errors")
	cmd.Flags().BoolVar(&noColor, "no-color", false, "disable colored output")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "write markdown report to file")
	cmd.Flags().StringVar(&sessionID, "session", "", "attach run to an explicit session")

	return cmd
}

func runReconCommand(cmd *cobra.Command, app *App, target string) error {
	if app == nil || app.DB == nil {
		return fmt.Errorf("database is not initialized")
	}
	outputPath := flagString(cmd, "output")
	if outputPath != "" {
		if err := validateReportOutputPath(outputPath); err != nil {
			return err
		}
	}
	session, err := validateReconSession(cmd, app, target)
	if err != nil {
		return err
	}

	runID, err := createRun(cmd, app, sessionIDFromSession(session))
	if err != nil {
		return err
	}
	verbosity := verbosityFromFlags(flagBool(cmd, "quiet"), flagCount(cmd, "verbose"))
	reporter := newStdoutReporter(cmd.OutOrStdout(), verbosity, !flagBool(cmd, "no-color"))
	reporter.RunStarted(runID, target)
	if session != nil {
		reporter.SessionSelected(*session)
	}
	sink := reportingSink{inner: &events.SQLiteSink{DB: app.DB}, reporter: reporter}
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
		reporter.RunFailed(runID, err)
		return err
	}

	if err := executor.RunUntilIdle(cmd.Context()); err != nil {
		if updateErr := app.DB.UpdateRunStatus(cmd.Context(), runID, actions.RunStatusFailed); updateErr != nil {
			return fmt.Errorf("%w: mark run failed: %v", err, updateErr)
		}
		_ = sink.Append(cmd.Context(), events.Event{RunID: runID, EventType: events.EventRunFailed, EntityKind: events.EntityRun, EntityID: runID, PayloadJSON: mustPayloadJSON(map[string]string{"error": err.Error()}), CreatedAt: time.Now()})
		reporter.RunFailed(runID, err)
		return err
	}
	if err := app.DB.UpdateRunStatus(cmd.Context(), runID, actions.RunStatusCompleted); err != nil {
		return err
	}
	if err := sink.Append(cmd.Context(), events.Event{RunID: runID, EventType: events.EventRunCompleted, EntityKind: events.EntityRun, EntityID: runID, PayloadJSON: mustPayloadJSON(map[string]string{}), CreatedAt: time.Now()}); err != nil {
		return err
	}

	dbPath := ""
	if app.Config != nil {
		dbPath = app.Config.Storage.DBPath
	}
	summary, err := viewmodel.BuildRunSummary(cmd.Context(), app.DB, runID, dbPath)
	if err != nil {
		return err
	}
	reporter.RunCompleted(summary)
	if outputPath != "" {
		if err := writeMarkdownReport(outputPath, summary); err != nil {
			return err
		}
		if verbosity != VerbosityQuiet {
			fmt.Fprintf(cmd.OutOrStdout(), "\nReport written: %s\n", outputPath)
		}
	}

	return nil
}

func validateReportOutputPath(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return reportFileExistsError(path)
	}
	if os.IsNotExist(err) {
		return nil
	}
	return fmt.Errorf("check report path %s: %w", path, err)
}

func writeMarkdownReport(path string, summary *viewmodel.RunSummary) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return reportFileExistsError(path)
		}
		return fmt.Errorf("write report %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()
	if _, err := file.WriteString(reporting.RenderMarkdownReport(summary)); err != nil {
		return fmt.Errorf("write report %s: %w", path, err)
	}
	return nil
}

func reportFileExistsError(path string) error {
	return fmt.Errorf("report file already exists: %s; choose a different path or remove the file", path)
}

func flagBool(cmd *cobra.Command, name string) bool {
	value, err := cmd.Flags().GetBool(name)
	if err != nil {
		return false
	}
	return value
}

func flagCount(cmd *cobra.Command, name string) int {
	value, err := cmd.Flags().GetCount(name)
	if err != nil {
		return 0
	}
	return value
}

func flagString(cmd *cobra.Command, name string) string {
	value, err := cmd.Flags().GetString(name)
	if err != nil {
		return ""
	}
	return value
}

func validateReconSession(cmd *cobra.Command, app *App, rawTarget string) (*sqlite.Session, error) {
	sessionID := flagString(cmd, "session")
	if sessionID == "" {
		return nil, nil
	}
	session, err := app.DB.GetSession(cmd.Context(), sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session %s: %w", sessionID, err)
	}
	if session.Status != sqlite.SessionStatusActive {
		return nil, fmt.Errorf("session %s is %s", session.ID, session.Status)
	}
	target, err := targets.Parse(rawTarget)
	if err != nil {
		return nil, err
	}
	rules, err := app.DB.ListScopeRulesBySession(cmd.Context(), session.ID)
	if err != nil {
		return nil, err
	}
	decision := scope.EvaluateTarget(target, rules)
	if !decision.Allowed {
		return nil, fmt.Errorf("target outside session scope: %s", decision.Reason)
	}
	return session, nil
}

func sessionIDFromSession(session *sqlite.Session) string {
	if session == nil {
		return ""
	}
	return session.ID
}

func createRun(cmd *cobra.Command, app *App, sessionID string) (string, error) {
	runID := "run_" + generateID()
	run := sqlite.Run{
		ID:        runID,
		SessionID: sessionID,
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
