package cmd

import (
	"github.com/skye-lopez/go-index/idx"
	"github.com/spf13/cobra"
)

var updateDbCmd = &cobra.Command{
	Use:   "update-db",
	Short: "Updates db to newest version of index",
	Long:  "Updates db to newest version of index",
	Run:   updateDB,
}

func init() {
	rootCmd.AddCommand(updateDbCmd)
}

func updateDB(cmd *cobra.Command, args []string) {
	idx.Fetch()
}
