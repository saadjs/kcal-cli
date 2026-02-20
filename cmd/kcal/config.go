package kcal

import (
	"database/sql"
	"fmt"
	"sort"

	"github.com/saad/kcal-cli/internal/service"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage kcal local configuration",
}

var (
	cfgBarcodeProvider      string
	cfgBarcodeFallbackOrder string
	cfgAPIKeyHint           string
)

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set configuration values",
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			updates := 0
			if cmd.Flags().Changed("barcode-provider") {
				if err := service.SetConfig(sqldb, service.ConfigBarcodeProvider, cfgBarcodeProvider); err != nil {
					return err
				}
				updates++
			}
			if cmd.Flags().Changed("fallback-order") {
				if err := service.SetConfig(sqldb, service.ConfigBarcodeFallbackOrder, cfgBarcodeFallbackOrder); err != nil {
					return err
				}
				updates++
			}
			if cmd.Flags().Changed("api-key-hint") {
				if err := service.SetConfig(sqldb, service.ConfigAPIKeyHint, cfgAPIKeyHint); err != nil {
					return err
				}
				updates++
			}
			if updates == 0 {
				return fmt.Errorf("set at least one flag")
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated %d config value(s)\n", updates)
			return nil
		})
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			cfg, err := service.ListConfig(sqldb)
			if err != nil {
				return err
			}
			keys := make([]string, 0, len(cfg))
			for k := range cfg {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			fmt.Fprintln(cmd.OutOrStdout(), "KEY\tVALUE")
			for _, k := range keys {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", k, cfg[k])
			}
			return nil
		})
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd, configGetCmd)

	configSetCmd.Flags().StringVar(&cfgBarcodeProvider, "barcode-provider", "", "Default barcode provider")
	configSetCmd.Flags().StringVar(&cfgBarcodeFallbackOrder, "fallback-order", "", "Default fallback order (comma-separated)")
	configSetCmd.Flags().StringVar(&cfgAPIKeyHint, "api-key-hint", "", "API key setup hint text (non-secret)")
}
