package kcal

import (
	"database/sql"
	"fmt"

	"github.com/saadjs/kcal-cli/internal/service"
	"github.com/spf13/cobra"
)

var bodyCmd = &cobra.Command{
	Use:   "body",
	Short: "Manage body measurements (weight and body-fat)",
}

var (
	bodyWeight float64
	bodyUnit   string
	bodyFat    float64
	bodyDate   string
	bodyTime   string
	bodyNotes  string
)

var bodyAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add body measurement",
	RunE: func(cmd *cobra.Command, args []string) error {
		measuredAt, err := parseDateTimeOrNow(bodyDate, bodyTime)
		if err != nil {
			return err
		}
		in := service.BodyMeasurementInput{
			Weight:     bodyWeight,
			Unit:       bodyUnit,
			BodyFatPct: optionalBodyFat(bodyFat),
			MeasuredAt: measuredAt,
			Notes:      bodyNotes,
		}
		return withDB(func(sqldb *sql.DB) error {
			id, err := service.AddBodyMeasurement(sqldb, in)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added body measurement %d\n", id)
			return nil
		})
	},
}

var (
	bodyListDate string
	bodyFrom     string
	bodyTo       string
	bodyLimit    int
	bodyOutUnit  string
)

var bodyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List body measurements",
	RunE: func(cmd *cobra.Command, args []string) error {
		filter := service.BodyMeasurementFilter{Date: bodyListDate, FromDate: bodyFrom, ToDate: bodyTo, Limit: bodyLimit}
		return withDB(func(sqldb *sql.DB) error {
			items, err := service.ListBodyMeasurements(sqldb, filter)
			if err != nil {
				return err
			}
			if bodyOutUnit == "" {
				bodyOutUnit = "kg"
			}
			fmt.Fprintln(cmd.OutOrStdout(), "ID\tDATE\tWEIGHT\tUNIT\tBODY_FAT%\tNOTES")
			for _, m := range items {
				w, err := service.WeightFromKg(m.WeightKg, bodyOutUnit)
				if err != nil {
					return err
				}
				bf := ""
				if m.BodyFatPct != nil {
					bf = fmt.Sprintf("%.2f", *m.BodyFatPct)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\t%.2f\t%s\t%s\t%s\n", m.ID, m.MeasuredAt.Local().Format("2006-01-02 15:04"), w, bodyOutUnit, bf, m.Notes)
			}
			return nil
		})
	},
}

var bodyUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update body measurement",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseInt64Arg("measurement id", args[0])
		if err != nil {
			return err
		}
		measuredAt, err := parseDateTime(bodyDate, bodyTime)
		if err != nil {
			return err
		}
		in := service.UpdateBodyMeasurementInput{
			ID: id,
			BodyMeasurementInput: service.BodyMeasurementInput{
				Weight:     bodyWeight,
				Unit:       bodyUnit,
				BodyFatPct: optionalBodyFat(bodyFat),
				MeasuredAt: measuredAt,
				Notes:      bodyNotes,
			},
		}
		return withDB(func(sqldb *sql.DB) error {
			if err := service.UpdateBodyMeasurement(sqldb, in); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated body measurement %d\n", id)
			return nil
		})
	},
}

var bodyDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete body measurement",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseInt64Arg("measurement id", args[0])
		if err != nil {
			return err
		}
		return withDB(func(sqldb *sql.DB) error {
			if err := service.DeleteBodyMeasurement(sqldb, id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted body measurement %d\n", id)
			return nil
		})
	},
}

func optionalBodyFat(v float64) *float64 {
	if v < 0 {
		return nil
	}
	return &v
}

func init() {
	rootCmd.AddCommand(bodyCmd)
	bodyCmd.AddCommand(bodyAddCmd, bodyListCmd, bodyUpdateCmd, bodyDeleteCmd)

	for _, c := range []*cobra.Command{bodyAddCmd, bodyUpdateCmd} {
		c.Flags().Float64Var(&bodyWeight, "weight", 0, "Weight value")
		c.Flags().StringVar(&bodyUnit, "unit", "kg", "Weight unit: kg or lb")
		c.Flags().Float64Var(&bodyFat, "body-fat", -1, "Body fat percentage (optional)")
		c.Flags().StringVar(&bodyDate, "date", "", "Date YYYY-MM-DD")
		c.Flags().StringVar(&bodyTime, "time", "", "Time HH:MM")
		c.Flags().StringVar(&bodyNotes, "notes", "", "Optional notes")
		_ = c.MarkFlagRequired("weight")
	}
	_ = bodyUpdateCmd.MarkFlagRequired("date")
	_ = bodyUpdateCmd.MarkFlagRequired("time")

	bodyListCmd.Flags().StringVar(&bodyListDate, "date", "", "Filter by date YYYY-MM-DD")
	bodyListCmd.Flags().StringVar(&bodyFrom, "from", "", "Filter from date YYYY-MM-DD")
	bodyListCmd.Flags().StringVar(&bodyTo, "to", "", "Filter to date YYYY-MM-DD")
	bodyListCmd.Flags().IntVar(&bodyLimit, "limit", 50, "Result limit")
	bodyListCmd.Flags().StringVar(&bodyOutUnit, "unit", "kg", "Output unit: kg or lb")
}
