package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newReconCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recon",
		Short: "Run recon commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Running recon on: %v\n", args)

			fmt.Println(app.Config.DBPath)
			return nil
		},
	}
	

	return cmd
}

