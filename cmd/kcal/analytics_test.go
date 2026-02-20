package kcal

import "testing"

func TestResolveWeekRangeRejectsMalformedWeek(t *testing.T) {
	t.Parallel()
	_, _, err := resolveWeekRange("2026-W1")
	if err == nil {
		t.Fatalf("expected malformed week format to fail")
	}
}

func TestResolveWeekRangeRejectsWeekZero(t *testing.T) {
	t.Parallel()
	_, _, err := resolveWeekRange("2026-W00")
	if err == nil {
		t.Fatalf("expected week zero to fail")
	}
}

func TestResolveWeekRangeRejectsOutOfRangeWeek(t *testing.T) {
	t.Parallel()
	_, _, err := resolveWeekRange("2021-W53")
	if err == nil {
		t.Fatalf("expected out-of-range week to fail")
	}
}

func TestResolveWeekRangeAcceptsValidISOWeek(t *testing.T) {
	t.Parallel()
	start, end, err := resolveWeekRange("2020-W53")
	if err != nil {
		t.Fatalf("expected valid ISO week, got error: %v", err)
	}
	if start.Format("2006-01-02") != "2020-12-28" {
		t.Fatalf("expected start 2020-12-28, got %s", start.Format("2006-01-02"))
	}
	if end.Format("2006-01-02") != "2021-01-03" {
		t.Fatalf("expected end 2021-01-03, got %s", end.Format("2006-01-02"))
	}
}
