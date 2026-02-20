package kcal

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/saad/kcal-cli/internal/service"
	"github.com/spf13/cobra"
)

var exerciseCmd = &cobra.Command{
	Use:   "exercise",
	Short: "Manage exercise logs",
}

var (
	exerciseType         string
	exerciseCalories     int
	exerciseDurationMin  int
	exerciseDistance     float64
	exerciseDistanceUnit string
	exerciseDate         string
	exerciseTime         string
	exerciseNotes        string
	exerciseMetadata     string
)

var exerciseAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add exercise log",
	RunE: func(cmd *cobra.Command, args []string) error {
		performedAt, err := parseDateTimeOrNow(exerciseDate, exerciseTime)
		if err != nil {
			return err
		}
		in, err := buildExerciseInput(cmd, performedAt)
		if err != nil {
			return err
		}
		return withDB(func(sqldb *sql.DB) error {
			id, err := service.CreateExerciseLog(sqldb, in)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added exercise log %d\n", id)
			return nil
		})
	},
}

var (
	exerciseListDate string
	exerciseFromDate string
	exerciseToDate   string
	exerciseListType string
	exerciseLimit    int
)

var exerciseListCmd = &cobra.Command{
	Use:   "list",
	Short: "List exercise logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		filter := service.ListExerciseFilter{
			Date:         exerciseListDate,
			FromDate:     exerciseFromDate,
			ToDate:       exerciseToDate,
			ExerciseType: strings.TrimSpace(exerciseListType),
			Limit:        exerciseLimit,
		}
		return withDB(func(sqldb *sql.DB) error {
			items, err := service.ListExerciseLogs(sqldb, filter)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "ID\tDATE\tTYPE\tKCAL_BURNED\tDURATION_MIN\tDISTANCE\tUNIT\tNOTES")
			for _, item := range items {
				duration := ""
				if item.DurationMin != nil {
					duration = fmt.Sprintf("%d", *item.DurationMin)
				}
				distance := ""
				if item.Distance != nil {
					distance = fmt.Sprintf("%.2f", *item.Distance)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\t%s\t%d\t%s\t%s\t%s\t%s\n", item.ID, item.PerformedAt.Local().Format("2006-01-02 15:04"), item.ExerciseType, item.CaloriesBurned, duration, distance, item.DistanceUnit, item.Notes)
			}
			return nil
		})
	},
}

var exerciseUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update exercise log",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseInt64Arg("exercise id", args[0])
		if err != nil {
			return err
		}
		performedAt, err := parseDateTime(exerciseDate, exerciseTime)
		if err != nil {
			return err
		}
		in, err := buildExerciseInput(cmd, performedAt)
		if err != nil {
			return err
		}
		return withDB(func(sqldb *sql.DB) error {
			if err := service.UpdateExerciseLog(sqldb, service.UpdateExerciseInput{ID: id, ExerciseLogInput: in}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated exercise log %d\n", id)
			return nil
		})
	},
}

var exerciseDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete exercise log",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseInt64Arg("exercise id", args[0])
		if err != nil {
			return err
		}
		return withDB(func(sqldb *sql.DB) error {
			if err := service.DeleteExerciseLog(sqldb, id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted exercise log %d\n", id)
			return nil
		})
	},
}

func buildExerciseInput(cmd *cobra.Command, performedAt time.Time) (service.ExerciseLogInput, error) {
	var duration *int
	if cmd.Flags().Changed("duration-min") {
		v := exerciseDurationMin
		duration = &v
	}

	var distance *float64
	if cmd.Flags().Changed("distance") {
		v := exerciseDistance
		distance = &v
	}

	return service.ExerciseLogInput{
		ExerciseType:   exerciseType,
		CaloriesBurned: exerciseCalories,
		DurationMin:    duration,
		Distance:       distance,
		DistanceUnit:   exerciseDistanceUnit,
		PerformedAt:    performedAt,
		Notes:          exerciseNotes,
		Metadata:       exerciseMetadata,
	}, nil
}

func init() {
	rootCmd.AddCommand(exerciseCmd)
	exerciseCmd.AddCommand(exerciseAddCmd, exerciseListCmd, exerciseUpdateCmd, exerciseDeleteCmd)

	for _, c := range []*cobra.Command{exerciseAddCmd, exerciseUpdateCmd} {
		c.Flags().StringVar(&exerciseType, "type", "", "Exercise type (running, cycling, strength, etc.)")
		c.Flags().IntVar(&exerciseCalories, "calories", 0, "Calories burned")
		c.Flags().IntVar(&exerciseDurationMin, "duration-min", 0, "Duration in minutes (optional)")
		c.Flags().Float64Var(&exerciseDistance, "distance", 0, "Distance (optional)")
		c.Flags().StringVar(&exerciseDistanceUnit, "distance-unit", "", "Distance unit: km or mi (required with --distance)")
		c.Flags().StringVar(&exerciseDate, "date", "", "Date YYYY-MM-DD")
		c.Flags().StringVar(&exerciseTime, "time", "", "Time HH:MM")
		c.Flags().StringVar(&exerciseNotes, "notes", "", "Optional notes")
		c.Flags().StringVar(&exerciseMetadata, "metadata-json", "", "Optional metadata JSON object")
		_ = c.MarkFlagRequired("type")
		_ = c.MarkFlagRequired("calories")
	}
	_ = exerciseUpdateCmd.MarkFlagRequired("date")
	_ = exerciseUpdateCmd.MarkFlagRequired("time")

	exerciseListCmd.Flags().StringVar(&exerciseListDate, "date", "", "Filter by date YYYY-MM-DD")
	exerciseListCmd.Flags().StringVar(&exerciseFromDate, "from", "", "Filter from date YYYY-MM-DD")
	exerciseListCmd.Flags().StringVar(&exerciseToDate, "to", "", "Filter to date YYYY-MM-DD")
	exerciseListCmd.Flags().StringVar(&exerciseListType, "type", "", "Filter by exercise type")
	exerciseListCmd.Flags().IntVar(&exerciseLimit, "limit", 50, "Result limit")
}
