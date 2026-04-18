package cli

import (
	"github.com/ruohao1/penta/internal/config"
	"github.com/spf13/cobra"
)

func NewPentaCommand() *cobra.Command {
	app := &App{}

	cmd := &cobra.Command{
		Use:   "penta",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			config, err := config.Load()
			if err != nil {
				return err
			}
			app.Config = config
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error{
			return cmd.Help()
		},
	}

	cmd.AddCommand(newReconCommand(app))

	return cmd
}

func Execute() error {
	cmd := NewPentaCommand()
	return cmd.Execute()
}


