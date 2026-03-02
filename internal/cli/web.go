package cli

import (
	"github.com/spf13/cobra"
)

func newWebCmd() *cobra.Command {
	webCmd := &cobra.Command{
		Use:   "web",
		Short: "Web application security testing",
	}

	webCmd.AddCommand(newXSSCmd())
	webCmd.AddCommand(newFuzzCmd())
	return webCmd
}

func newFuzzCmd() *cobra.Command {
	fuzzCmd := &cobra.Command{
		Use:   "fuzz",
		Short: "Web application fuzzing",
	}

	return fuzzCmd
}
