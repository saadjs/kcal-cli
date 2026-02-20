package kcal

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestRootHelp(t *testing.T) {
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
