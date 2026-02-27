package app

import (
	"io"
	"os"

	"github.com/Ruohao1/penta/internal/config"
	"github.com/Ruohao1/penta/internal/model"
	"github.com/Ruohao1/penta/internal/sinks"
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

			runSink := sinks.NewPentaSink(sinks.SinkOptions{
				Human:   opts.Human,
				Verbose: opts.Verbosity,
				Out:     cmd.OutOrStdout(),
				Err:     cmd.ErrOrStderr(),
				// NDJSON: file writer if --log-file/--output is set
			})

			logFile, err := os.OpenFile("/tmp/penta.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if err != nil {
				return err
			}
			logger := zerolog.New(io.MultiWriter(cmd.ErrOrStderr(), logFile)).
				Level(lvl).
				With().
				Timestamp().
				Logger()

			ctx := cmd.Context()
			ctx = utils.WithLogger(ctx, logger)
			ctx = utils.WithConfig(ctx, cfg)
			ctx = utils.WithSink(ctx, runSink)
			cmd.SetContext(ctx)
			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			if s := utils.SinkFrom(cmd.Context()); s != nil {
				return s.Close()
			}
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
	cmd.PersistentFlags().StringVar(&opts.LogFile, "log-file", "", "optional log file path")

	return cmd
}

func init() {
	// rootCmd.AddCommand(NewSessionCmd())
	rootCmd.AddCommand(NewScanCmd())
	rootCmd.AddCommand(newXSSCmd())
	rootCmd.AddCommand(newTUICmd())
	// rootCmd.AddCommand(NewBruteCmd())
}
