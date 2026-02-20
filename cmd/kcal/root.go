package kcal

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var dbPath string

var rootCmd = &cobra.Command{
	Use:   "kcal",
	Short: "kcal tracks calories and macros from your terminal",
	Long:  "kcal is a local-first calorie and macro tracking CLI with categories, recipes, goals, and analytics.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "Path to SQLite database")
}
