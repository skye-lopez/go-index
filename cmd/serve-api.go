package cmd

import (
	"github.com/spf13/cobra"
)

var serveApiCmd = &cobra.Command{
	Use:   "serve-api",
	Short: "serves api @0.0.0.0:8080",
	Long:  "serves api @0.0.0.0:8080",
	Run:   serveApi,
}

func init() {
	rootCmd.AddCommand(serveApiCmd)
}

func serveApi(cmd *cobra.Command, args []string) {
	// api.Open()
}
