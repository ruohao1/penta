package app

import (
	"github.com/Ruohao1/penta/internal/config"
	"github.com/Ruohao1/penta/internal/model"
	"github.com/Ruohao1/penta/internal/utils"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var rootCmd = newRootCmd()

func Execute() error {
	return rootCmd.Execute()
}

func newRootCmd() *cobra.Command {
	var opts model.GlobalOptions

	cmd := &cobra.Command{
		Use:          "penta",
		Short:        "Ultimate pentest CLI engine",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var lvl zerolog.Level
			switch {
			case opts.Verbosity >= 3:
				lvl = zerolog.TraceLevel // -vvv and beyond
			case opts.Verbosity == 2:
				lvl = zerolog.DebugLevel // -vv
			case opts.Verbosity == 1:
				lvl = zerolog.InfoLevel // -v
			default:
				lvl = zerolog.WarnLevel // no -v
			}

			cfg := config.LoadConfig()

			logger := utils.NewLogger(opts.Human, lvl).
				With().
				Str("cmd", cmd.Name()).
				Logger()
			zerolog.SetGlobalLevel(lvl)

			ctx := cmd.Context()
			ctx = utils.WithLogger(ctx, logger)
			ctx = utils.WithConfig(ctx, cfg)
			cmd.SetContext(ctx)

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// ctx := cmd.Context()
			// return tui.RunTUI(ctx, tui.TuiOptions{})
			return nil
		},
	}

	cmd.PersistentFlags().BoolVar(&opts.Human, "human", true, "human-friendly log output")
	cmd.PersistentFlags().CountVarP(&opts.Verbosity, "verbose", "v", "increase verbosity (-v, -vv, -vvv)")
	cmd.PersistentFlags().BoolVar(&opts.TUI, "tui mode", true, "use tui mode")
	return cmd
}

func init() {
	// rootCmd.AddCommand(NewSessionCmd())
	rootCmd.AddCommand(NewScanCmd())
	rootCmd.AddCommand(newXSSCmd())
	// rootCmd.AddCommand(NewBruteCmd())
}
