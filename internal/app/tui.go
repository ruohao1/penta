package app

import (
	"github.com/Ruohao1/penta/internal/tui"
	"github.com/spf13/cobra"
)

func newTUICmd() *cobra.Command {
	tuiCmd := &cobra.Command{
		Use:   "tui",
		Short: "Run the interactive terminal user interface (TUI)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.Run()
		}}
	return tuiCmd
}
