package kcal

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var dbPath string
var showVersion bool

var (
	version = "v1.1.0"
	commit  = "unknown"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "kcal",
	Short: "kcal tracks calories and macros from your terminal",
	Long:  "kcal is a local-first calorie and macro tracking CLI with categories, recipes, goals, and analytics.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			printVersion(cmd)
			return nil
		}
		return cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "Path to SQLite database")
	rootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "Show version/build metadata")
}

func printVersion(cmd *cobra.Command) {
	fmt.Fprintf(cmd.OutOrStdout(), "kcal %s\ncommit: %s\ndate: %s\n", version, commit, date)
}
