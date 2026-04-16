package cmd

import (
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart [workspace]",
	Short: "Restart a workspace environment",
	Long:  "Stop and restart the Docker Compose environment for a workspace. Regenerates the compose configuration from the current config. If the workspace is not running, starts it.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if err := runDownCtx(ctx, args); err != nil {
			return err
		}

		return runUp(ctx, args)
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)
}
