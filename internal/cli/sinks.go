package cli

import (
	"github.com/ruohao1/penta/internal/output"
	"github.com/spf13/cobra"
)

func commandSinks(cmd *cobra.Command, app *App) output.Sinks {
	if app != nil && app.Sinks.Out != nil && app.Sinks.Err != nil {
		return app.Sinks
	}
	return output.New(cmd.OutOrStdout(), cmd.ErrOrStderr())
}
