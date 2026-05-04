package cli

import (
	"github.com/ruohao1/penta/internal/config"
	"github.com/ruohao1/penta/internal/storage/sqlite"
	"github.com/spf13/cobra"
)

func NewPentaCommand() *cobra.Command {
	app := &App{}

	cmd := &cobra.Command{
		Use:           "penta",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			config, err := config.Load()
			if err != nil {
				return err
			}
			app.Config = config
			app.DB, err = sqlite.Open(cmd.Context(), config.Storage.DBPath)
			if err != nil {
				return err
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			if app.DB != nil {
				return app.DB.Close()
			}
			return nil
		},
	}

	cmd.AddCommand(newReconCommand(app), newSessionCommand(app))

	return cmd
}

func Execute() error {
	cmd := NewPentaCommand()
	return cmd.Execute()
}
