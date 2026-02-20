package kcal

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/saadjs/kcal-cli/internal/service"
	"github.com/spf13/cobra"
)

var (
	exportFormat string
	exportOut    string
	importFormat string
	importIn     string
	importMode   string
	importDryRun bool
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export local data (json or csv)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(exportOut) == "" {
			return fmt.Errorf("--out is required")
		}
		return withDB(func(sqldb *sql.DB) error {
			switch strings.ToLower(strings.TrimSpace(exportFormat)) {
			case "json":
				data, err := service.ExportDataSnapshot(sqldb)
				if err != nil {
					return err
				}
				b, err := json.MarshalIndent(data, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal export json: %w", err)
				}
				if err := os.WriteFile(exportOut, b, 0o644); err != nil {
					return fmt.Errorf("write export file: %w", err)
				}
			case "csv":
				entries, err := service.ListEntries(sqldb, service.ListEntriesFilter{Limit: 1000000})
				if err != nil {
					return err
				}
				f, err := os.Create(exportOut)
				if err != nil {
					return fmt.Errorf("create export csv: %w", err)
				}
				defer f.Close()
				w := csv.NewWriter(f)
				defer w.Flush()
				if err := w.Write([]string{"name", "calories", "protein_g", "carbs_g", "fat_g", "fiber_g", "sugar_g", "sodium_mg", "micronutrients_json", "category", "consumed_at", "notes", "source_type", "source_id", "metadata_json"}); err != nil {
					return fmt.Errorf("write export csv header: %w", err)
				}
				for _, e := range entries {
					sourceID := ""
					if e.SourceID != nil {
						sourceID = strconv.FormatInt(*e.SourceID, 10)
					}
					record := []string{
						e.Name,
						strconv.Itoa(e.Calories),
						strconv.FormatFloat(e.ProteinG, 'f', -1, 64),
						strconv.FormatFloat(e.CarbsG, 'f', -1, 64),
						strconv.FormatFloat(e.FatG, 'f', -1, 64),
						strconv.FormatFloat(e.FiberG, 'f', -1, 64),
						strconv.FormatFloat(e.SugarG, 'f', -1, 64),
						strconv.FormatFloat(e.SodiumMg, 'f', -1, 64),
						e.Micronutrients,
						e.Category,
						e.ConsumedAt.Format("2006-01-02T15:04:05-07:00"),
						e.Notes,
						e.SourceType,
						sourceID,
						e.Metadata,
					}
					if err := w.Write(record); err != nil {
						return fmt.Errorf("write export csv row: %w", err)
					}
				}
			default:
				return fmt.Errorf("unsupported --format %q (use json or csv)", exportFormat)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Exported data to %s\n", exportOut)
			return nil
		})
	},
}

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import local data (json or csv)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(importIn) == "" {
			return fmt.Errorf("--in is required")
		}
		return withDB(func(sqldb *sql.DB) error {
			switch strings.ToLower(strings.TrimSpace(importFormat)) {
			case "json":
				raw, err := os.ReadFile(importIn)
				if err != nil {
					return fmt.Errorf("read import file: %w", err)
				}
				var payload service.ExportData
				if err := json.Unmarshal(raw, &payload); err != nil {
					return fmt.Errorf("parse import json: %w", err)
				}
				report, err := service.ImportDataSnapshotWithOptions(sqldb, &payload, service.ImportOptions{
					Mode:   service.ImportMode(strings.ToLower(strings.TrimSpace(importMode))),
					DryRun: importDryRun,
				})
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Import report: inserted=%d updated=%d skipped=%d conflicts=%d\n", report.Inserted, report.Updated, report.Skipped, report.Conflicts)
				for _, w := range report.Warnings {
					fmt.Fprintf(cmd.OutOrStdout(), "warning: %s\n", w)
				}
			case "csv":
				f, err := os.Open(importIn)
				if err != nil {
					return fmt.Errorf("open import csv: %w", err)
				}
				defer f.Close()
				r := csv.NewReader(f)
				records, err := r.ReadAll()
				if err != nil {
					return fmt.Errorf("read import csv: %w", err)
				}
				if len(records) <= 1 {
					return fmt.Errorf("import csv contains no data rows")
				}
				for i := 1; i < len(records); i++ {
					row := records[i]
					if len(row) != 11 && len(row) != 15 {
						return fmt.Errorf("csv row %d has %d columns, expected 11 (legacy) or 15", i+1, len(row))
					}
					kcal, _ := strconv.Atoi(row[1])
					protein, _ := strconv.ParseFloat(row[2], 64)
					carbs, _ := strconv.ParseFloat(row[3], 64)
					fat, _ := strconv.ParseFloat(row[4], 64)
					fiber := 0.0
					sugar := 0.0
					sodium := 0.0
					micros := ""
					categoryIdx := 5
					consumedAtIdx := 6
					notesIdx := 7
					sourceTypeIdx := 8
					sourceIDIdx := 9
					metadataIdx := 10
					if len(row) == 15 {
						fiber, _ = strconv.ParseFloat(row[5], 64)
						sugar, _ = strconv.ParseFloat(row[6], 64)
						sodium, _ = strconv.ParseFloat(row[7], 64)
						micros = row[8]
						categoryIdx = 9
						consumedAtIdx = 10
						notesIdx = 11
						sourceTypeIdx = 12
						sourceIDIdx = 13
						metadataIdx = 14
					}
					sourceIDVal, _ := strconv.ParseInt(strings.TrimSpace(row[sourceIDIdx]), 10, 64)
					var sourceID *int64
					if sourceIDVal > 0 {
						sourceID = &sourceIDVal
					}
					consumed, err := parseCSVTime(row[consumedAtIdx])
					if err != nil {
						return fmt.Errorf("csv row %d consumed_at: %w", i+1, err)
					}
					categoryName := strings.ToLower(strings.TrimSpace(row[categoryIdx]))
					if categoryName == "" {
						return fmt.Errorf("csv row %d category is required", i+1)
					}
					if importDryRun {
						continue
					}
					if _, err := sqldb.Exec(`INSERT OR IGNORE INTO categories(name, is_default) VALUES(?, 0)`, categoryName); err != nil {
						return fmt.Errorf("import csv row %d category %q: %w", i+1, categoryName, err)
					}
					if _, err := service.CreateEntry(sqldb, service.CreateEntryInput{
						Name:           row[0],
						Calories:       kcal,
						ProteinG:       protein,
						CarbsG:         carbs,
						FatG:           fat,
						FiberG:         fiber,
						SugarG:         sugar,
						SodiumMg:       sodium,
						Micronutrients: micros,
						Category:       categoryName,
						Consumed:       consumed,
						Notes:          row[notesIdx],
						SourceType:     row[sourceTypeIdx],
						SourceID:       sourceID,
						Metadata:       row[metadataIdx],
					}); err != nil {
						return fmt.Errorf("import csv row %d: %w", i+1, err)
					}
				}
			default:
				return fmt.Errorf("unsupported --format %q (use json or csv)", importFormat)
			}
			if importDryRun {
				fmt.Fprintf(cmd.OutOrStdout(), "Dry-run import validated %s\n", importIn)
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Imported data from %s\n", importIn)
			return nil
		})
	},
}

func parseCSVTime(value string) (t time.Time, err error) {
	layouts := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02 15:04"}
	for _, l := range layouts {
		t, err = time.Parse(l, value)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid timestamp %q", value)
}

func init() {
	rootCmd.AddCommand(exportCmd, importCmd)
	exportCmd.Flags().StringVar(&exportFormat, "format", "json", "Export format: json or csv")
	exportCmd.Flags().StringVar(&exportOut, "out", "", "Output file path")
	importCmd.Flags().StringVar(&importFormat, "format", "json", "Import format: json or csv")
	importCmd.Flags().StringVar(&importIn, "in", "", "Input file path")
	importCmd.Flags().StringVar(&importMode, "mode", "merge", "Import mode for JSON: fail|skip|merge|replace")
	importCmd.Flags().BoolVar(&importDryRun, "dry-run", false, "Validate and report without writing data")
}
