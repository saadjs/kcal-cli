package kcal

import (
	"database/sql"
	"fmt"

	"github.com/saad/kcal-cli/internal/service"
	"github.com/spf13/cobra"
)

var entryCmd = &cobra.Command{
	Use:   "entry",
	Short: "Manage calorie and macro entries",
}

var (
	entryName     string
	entryCalories int
	entryProtein  float64
	entryCarbs    float64
	entryFat      float64
	entryCategory string
	entryDate     string
	entryTime     string
	entryNotes    string
)

var entryAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new entry",
	RunE: func(cmd *cobra.Command, args []string) error {
		consumed, err := parseDateTimeOrNow(entryDate, entryTime)
		if err != nil {
			return err
		}
		in := service.CreateEntryInput{
			Name:       entryName,
			Calories:   entryCalories,
			ProteinG:   entryProtein,
			CarbsG:     entryCarbs,
			FatG:       entryFat,
			Category:   entryCategory,
			Consumed:   consumed,
			Notes:      entryNotes,
			SourceType: "manual",
		}
		return withDB(func(sqldb *sql.DB) error {
			id, err := service.CreateEntry(sqldb, in)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added entry %d\n", id)
			return nil
		})
	},
}

var (
	listDate     string
	listFromDate string
	listToDate   string
	listCategory string
	listLimit    int
)

var entryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List entries",
	RunE: func(cmd *cobra.Command, args []string) error {
		filter := service.ListEntriesFilter{
			Date:     listDate,
			FromDate: listFromDate,
			ToDate:   listToDate,
			Category: listCategory,
			Limit:    listLimit,
		}
		return withDB(func(sqldb *sql.DB) error {
			entries, err := service.ListEntries(sqldb, filter)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "ID\tDATE\tCATEGORY\tNAME\tKCAL\tP\tC\tF\tSOURCE")
			for _, e := range entries {
				fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\t%s\t%s\t%d\t%.1f\t%.1f\t%.1f\t%s\n", e.ID, e.ConsumedAt.Local().Format("2006-01-02 15:04"), e.Category, e.Name, e.Calories, e.ProteinG, e.CarbsG, e.FatG, e.SourceType)
			}
			return nil
		})
	},
}

var (
	updateName     string
	updateCalories int
	updateProtein  float64
	updateCarbs    float64
	updateFat      float64
	updateCategory string
	updateDate     string
	updateTime     string
	updateNotes    string
)

var entryUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseInt64Arg("entry id", args[0])
		if err != nil {
			return err
		}
		consumed, err := parseDateTime(updateDate, updateTime)
		if err != nil {
			return err
		}

		in := service.UpdateEntryInput{
			ID:       id,
			Name:     updateName,
			Calories: updateCalories,
			ProteinG: updateProtein,
			CarbsG:   updateCarbs,
			FatG:     updateFat,
			Category: updateCategory,
			Consumed: consumed,
			Notes:    updateNotes,
		}
		return withDB(func(sqldb *sql.DB) error {
			if err := service.UpdateEntry(sqldb, in); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated entry %d\n", id)
			return nil
		})
	},
}

var entryDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseInt64Arg("entry id", args[0])
		if err != nil {
			return err
		}
		return withDB(func(sqldb *sql.DB) error {
			if err := service.DeleteEntry(sqldb, id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted entry %d\n", id)
			return nil
		})
	},
}

func addEntryFields(cmd *cobra.Command, prefix string) {
	cmd.Flags().StringVar(&entryName, "name", "", "Entry name")
	cmd.Flags().IntVar(&entryCalories, "calories", 0, "Calories")
	cmd.Flags().Float64Var(&entryProtein, "protein", 0, "Protein grams")
	cmd.Flags().Float64Var(&entryCarbs, "carbs", 0, "Carbs grams")
	cmd.Flags().Float64Var(&entryFat, "fat", 0, "Fat grams")
	cmd.Flags().StringVar(&entryCategory, "category", "", "Category name")
	cmd.Flags().StringVar(&entryDate, "date", "", "Date in YYYY-MM-DD")
	cmd.Flags().StringVar(&entryTime, "time", "", "Time in HH:MM")
	cmd.Flags().StringVar(&entryNotes, "notes", "", "Optional notes")
	_ = prefix
}

func init() {
	rootCmd.AddCommand(entryCmd)
	entryCmd.AddCommand(entryAddCmd, entryListCmd, entryUpdateCmd, entryDeleteCmd)

	addEntryFields(entryAddCmd, "add")
	_ = entryAddCmd.MarkFlagRequired("name")
	_ = entryAddCmd.MarkFlagRequired("calories")
	_ = entryAddCmd.MarkFlagRequired("protein")
	_ = entryAddCmd.MarkFlagRequired("carbs")
	_ = entryAddCmd.MarkFlagRequired("fat")
	_ = entryAddCmd.MarkFlagRequired("category")

	entryListCmd.Flags().StringVar(&listDate, "date", "", "Filter by date YYYY-MM-DD")
	entryListCmd.Flags().StringVar(&listFromDate, "from", "", "Filter from date YYYY-MM-DD")
	entryListCmd.Flags().StringVar(&listToDate, "to", "", "Filter to date YYYY-MM-DD")
	entryListCmd.Flags().StringVar(&listCategory, "category", "", "Filter by category")
	entryListCmd.Flags().IntVar(&listLimit, "limit", 50, "Result limit")

	entryUpdateCmd.Flags().StringVar(&updateName, "name", "", "Entry name")
	entryUpdateCmd.Flags().IntVar(&updateCalories, "calories", 0, "Calories")
	entryUpdateCmd.Flags().Float64Var(&updateProtein, "protein", 0, "Protein grams")
	entryUpdateCmd.Flags().Float64Var(&updateCarbs, "carbs", 0, "Carbs grams")
	entryUpdateCmd.Flags().Float64Var(&updateFat, "fat", 0, "Fat grams")
	entryUpdateCmd.Flags().StringVar(&updateCategory, "category", "", "Category name")
	entryUpdateCmd.Flags().StringVar(&updateDate, "date", "", "Date in YYYY-MM-DD")
	entryUpdateCmd.Flags().StringVar(&updateTime, "time", "", "Time in HH:MM")
	entryUpdateCmd.Flags().StringVar(&updateNotes, "notes", "", "Optional notes")
	_ = entryUpdateCmd.MarkFlagRequired("name")
	_ = entryUpdateCmd.MarkFlagRequired("calories")
	_ = entryUpdateCmd.MarkFlagRequired("protein")
	_ = entryUpdateCmd.MarkFlagRequired("carbs")
	_ = entryUpdateCmd.MarkFlagRequired("fat")
	_ = entryUpdateCmd.MarkFlagRequired("category")
	_ = entryUpdateCmd.MarkFlagRequired("date")
	_ = entryUpdateCmd.MarkFlagRequired("time")
}
