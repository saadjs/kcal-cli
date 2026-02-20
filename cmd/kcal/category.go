package kcal

import (
	"database/sql"
	"fmt"

	"github.com/saadjs/kcal-cli/internal/service"
	"github.com/spf13/cobra"
)

var categoryCmd = &cobra.Command{
	Use:   "category",
	Short: "Manage meal categories",
}

var categoryAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a custom category",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			if err := service.AddCategory(sqldb, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added category %q\n", args[0])
			return nil
		})
	},
}

var categoryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List categories",
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			categories, err := service.ListCategories(sqldb)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "ID\tNAME\tDEFAULT")
			for _, c := range categories {
				fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\t%t\n", c.ID, c.Name, c.IsDefault)
			}
			return nil
		})
	},
}

var categoryRenameCmd = &cobra.Command{
	Use:   "rename <old> <new>",
	Short: "Rename a category",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			if err := service.RenameCategory(sqldb, args[0], args[1]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Renamed category %q to %q\n", args[0], args[1])
			return nil
		})
	},
}

var categoryReassign string

var categoryDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a category; use --reassign when entries exist",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			if err := service.DeleteCategory(sqldb, args[0], categoryReassign); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted category %q\n", args[0])
			return nil
		})
	},
}

func init() {
	rootCmd.AddCommand(categoryCmd)
	categoryCmd.AddCommand(categoryAddCmd, categoryListCmd, categoryRenameCmd, categoryDeleteCmd)
	categoryDeleteCmd.Flags().StringVar(&categoryReassign, "reassign", "", "Category to move existing entries into")
}
