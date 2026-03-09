package cli

import (
	"github.com/ruohao1/penta/internal/config"
	"github.com/ruohao1/penta/internal/runtime"
	"github.com/spf13/cobra"
)

var rootCmd = newRootCmd()

func Execute() error {
	return rootCmd.Execute()
}

func newRootCmd() *cobra.Command {
	var (
		configPath string
		runCfg     = runtime.DefaultConfig()
	)

	cmd := &cobra.Command{
		Use:          "penta",
		Short:        "Ultimate pentest CLI engine",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			resolved := cfg.ToRuntimeConfig()
			if cmd.Flags().Changed("fail-fast") {
				resolved.FailFast = runCfg.FailFast
			}
			if cmd.Flags().Changed("buffer-size") {
				resolved.BufferSize = runCfg.BufferSize
			}
			if cmd.Flags().Changed("workers") {
				resolved.Workers = runCfg.Workers
			}
			if cmd.Flags().Changed("max-rate") {
				resolved.MaxRate = runCfg.MaxRate
			}
			if cmd.Flags().Changed("rate-burst") {
				resolved.RateBurst = runCfg.RateBurst
			}
			if cmd.Flags().Changed("max-retries") {
				resolved.MaxRetries = runCfg.MaxRetries
			}
			if cmd.Flags().Changed("retry-backoff") {
				resolved.RetryBackoff = runCfg.RetryBackoff
			}
			if cmd.Flags().Changed("timeout") {
				resolved.Timeout = runCfg.Timeout
			}

			ctx := config.WithRuntimeConfig(cmd.Context(), resolved)
			cmd.SetContext(ctx)
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&configPath, "config", "", "config file path (default: ~/.config/penta/config.yaml)")
	cmd.PersistentFlags().BoolVar(&runCfg.FailFast, "fail-fast", runCfg.FailFast, "stop run on first stage error")
	cmd.PersistentFlags().IntVar(&runCfg.BufferSize, "buffer-size", runCfg.BufferSize, "pipeline buffer size")
	cmd.PersistentFlags().IntVarP(&runCfg.Workers, "workers", "w", runCfg.Workers, "default workers per stage")
	cmd.PersistentFlags().Float64Var(&runCfg.MaxRate, "max-rate", runCfg.MaxRate, "default max stage rate (RPS); 0 disables")
	cmd.PersistentFlags().IntVar(&runCfg.RateBurst, "rate-burst", runCfg.RateBurst, "default stage rate burst")
	cmd.PersistentFlags().IntVar(&runCfg.MaxRetries, "max-retries", runCfg.MaxRetries, "default max retries per stage item")
	cmd.PersistentFlags().DurationVar(&runCfg.RetryBackoff, "retry-backoff", runCfg.RetryBackoff, "default retry backoff per stage item")
	cmd.PersistentFlags().DurationVar(&runCfg.Timeout, "timeout", runCfg.Timeout, "default timeout per stage item")

	return cmd
}

func init() {
	// rootCmd.AddCommand(NewScanCmd())
	// rootCmd.AddCommand(newXSSCmd())
	rootCmd.AddCommand(newTUICmd())
	rootCmd.AddCommand(newWebCmd())
}
