package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestSessionCommandsCreateListShowAndArchive(t *testing.T) {
	app := openTestApp(t)

	createOut := executeSessionCommand(t, app, "create", "Acme Program", "--kind", "bugbounty")
	if !strings.Contains(createOut, "Session created") || !strings.Contains(createOut, "Acme Program") || !strings.Contains(createOut, "bugbounty") {
		t.Fatalf("unexpected create output: %q", createOut)
	}
	sessionID := firstSessionID(t, app)

	listOut := executeSessionCommand(t, app, "list")
	if !strings.Contains(listOut, sessionID) || !strings.Contains(listOut, "Acme Program") {
		t.Fatalf("unexpected list output: %q", listOut)
	}

	showOut := executeSessionCommand(t, app, "show", sessionID)
	if !strings.Contains(showOut, "Runs      0 completed / 0 failed / 0 running / 0 pending") || !strings.Contains(showOut, "Status    active") {
		t.Fatalf("unexpected show output: %q", showOut)
	}

	archiveOut := executeSessionCommand(t, app, "archive", sessionID)
	if !strings.Contains(archiveOut, "Session archived: "+sessionID) {
		t.Fatalf("unexpected archive output: %q", archiveOut)
	}
	showOut = executeSessionCommand(t, app, "show", sessionID)
	if !strings.Contains(showOut, "Status    archived") {
		t.Fatalf("unexpected archived show output: %q", showOut)
	}
}

func TestSessionShowIncludesRunTaskAndEvidenceSummary(t *testing.T) {
	app := openTestApp(t)
	executeSessionCommand(t, app, "create", "Acme", "--kind", "bugbounty")
	sessionID := firstSessionID(t, app)
	target := newReconHTTPServer(t)
	createTestScopeRule(t, app, sessionID, "scope_include", "include", "url", target)
	createTestScopeRule(t, app, sessionID, "scope_local", "include", "ip", hostFromURL(t, target))
	cmd := newReconCommand(app)
	cmd.SetArgs([]string{"--session", sessionID, target, "-q"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute recon command: %v", err)
	}

	showOut := executeSessionCommand(t, app, "show", sessionID)
	for _, want := range []string{"Runs      1 completed / 0 failed / 0 running / 0 pending", "Tasks     4 completed / 0 failed / 0 pending", "Evidence  1 target / 1 service / 1 http_response", "Runs\n- run_"} {
		if !strings.Contains(showOut, want) {
			t.Fatalf("session show missing %q in %q", want, showOut)
		}
	}
}

func TestSessionScopeCommandsAddListAndRemove(t *testing.T) {
	app := openTestApp(t)
	executeSessionCommand(t, app, "create", "CTF Lab", "--kind", "ctf")
	sessionID := firstSessionID(t, app)

	addOut := executeSessionCommand(t, app, "scope", "add", sessionID, "domain", "*.example.com", "--include")
	if !strings.Contains(addOut, "Scope rule added") || !strings.Contains(addOut, "*.example.com") {
		t.Fatalf("unexpected scope add output: %q", addOut)
	}
	ruleID := firstScopeRuleID(t, app, sessionID)

	listOut := executeSessionCommand(t, app, "scope", "list", sessionID)
	if !strings.Contains(listOut, ruleID) || !strings.Contains(listOut, "include") || !strings.Contains(listOut, "domain") {
		t.Fatalf("unexpected scope list output: %q", listOut)
	}

	showOut := executeSessionCommand(t, app, "show", sessionID)
	if !strings.Contains(showOut, "Scope") || !strings.Contains(showOut, "include domain *.example.com") {
		t.Fatalf("unexpected session show scope output: %q", showOut)
	}

	removeOut := executeSessionCommand(t, app, "scope", "remove", ruleID)
	if !strings.Contains(removeOut, "Scope rule removed: "+ruleID) {
		t.Fatalf("unexpected scope remove output: %q", removeOut)
	}
	listOut = executeSessionCommand(t, app, "scope", "list", sessionID)
	if !strings.Contains(listOut, "No scope rules") {
		t.Fatalf("unexpected empty scope list output: %q", listOut)
	}
}

func TestSessionScopeAddRequiresExactlyOneEffect(t *testing.T) {
	app := openTestApp(t)
	executeSessionCommand(t, app, "create", "CTF Lab", "--kind", "ctf")
	sessionID := firstSessionID(t, app)
	cmd := newSessionCommand(app)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"scope", "add", sessionID, "domain", "example.com"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected missing effect to fail")
	}
	if !strings.Contains(err.Error(), "set exactly one of --include or --exclude") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func executeSessionCommand(t *testing.T, app *App, args ...string) string {
	t.Helper()
	cmd := newSessionCommand(app)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute session %v: %v\noutput: %s", args, err, out.String())
	}
	return out.String()
}

func firstSessionID(t *testing.T, app *App) string {
	t.Helper()
	sessions, err := app.DB.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("unexpected session count: %d", len(sessions))
	}
	return sessions[0].ID
}

func firstScopeRuleID(t *testing.T, app *App, sessionID string) string {
	t.Helper()
	rules, err := app.DB.ListScopeRulesBySession(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("list scope rules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("unexpected scope rule count: %d", len(rules))
	}
	return rules[0].ID
}
