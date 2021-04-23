package cmd

import "github.com/spf13/cobra"

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gdex",
		Short: "gravity dex backend",
	}
	cmd.AddCommand(TransformerCmd())
	cmd.AddCommand(ServerCmd())
	return cmd
}
