package kcal

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func resetCommandFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
	for _, child := range cmd.Commands() {
		resetCommandFlags(child)
	}
}

func TestRootHelp(t *testing.T) {
	resetCommandFlags(rootCmd)
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute root help: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatalf("expected help output")
	}
}

func TestInitCommandIdempotent(t *testing.T) {
	resetCommandFlags(rootCmd)
	path := filepath.Join(t.TempDir(), "kcal.db")
	for i := 0; i < 2; i++ {
		buf := &bytes.Buffer{}
		rootCmd.SetOut(buf)
		rootCmd.SetErr(buf)
		rootCmd.SetArgs([]string{"--db", path, "init"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("init run %d failed: %v", i+1, err)
		}
	}
}

func TestRootVersionFlagAndCommand(t *testing.T) {
	oldVersion, oldCommit, oldDate := version, commit, date
	version = "v1.1.0"
	commit = "abc123"
	date = "2026-02-20T00:00:00Z"
	t.Cleanup(func() {
		version = oldVersion
		commit = oldCommit
		date = oldDate
		showVersion = false
	})

	testCases := [][]string{
		{"--version"},
		{"-v"},
		{"version"},
	}

	for _, args := range testCases {
		resetCommandFlags(rootCmd)
		buf := &bytes.Buffer{}
		rootCmd.SetOut(buf)
		rootCmd.SetErr(buf)
		rootCmd.SetArgs(args)
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("execute %v: %v", args, err)
		}
		out := buf.String()
		if !strings.Contains(out, "kcal v1.1.0") {
			t.Fatalf("expected version in output for %v, got: %s", args, out)
		}
		if !strings.Contains(out, "commit: abc123") {
			t.Fatalf("expected commit in output for %v, got: %s", args, out)
		}
		if !strings.Contains(out, "date: 2026-02-20T00:00:00Z") {
			t.Fatalf("expected date in output for %v, got: %s", args, out)
		}
		showVersion = false
	}
}
