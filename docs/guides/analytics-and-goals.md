---
title: Analytics and Goals
---

# Analytics and Goals

Purpose: Interpret trend reports, adherence, and goal progress from `kcal` analytics commands.

When to use this page:
- You need weekly, monthly, or custom-range performance views.
- You are validating goal adherence and trend direction.

## Goal Setup and Tracking

```bash
kcal goal set --calories 2200 --protein 160 --carbs 240 --fat 70 --effective-date 2026-02-01
kcal goal current
kcal goal history
```

Suggested targets:

```bash
kcal goal suggest --weight 80 --unit kg --maintenance-calories 2500 --pace cut --apply --effective-date 2026-02-20
```

## Body Tracking

```bash
kcal body add --weight 172 --unit lb --body-fat 20 --date 2026-02-20 --time 07:00
kcal body-goal set --target-weight 170 --unit lb --target-body-fat 18 --effective-date 2026-02-20
```

## Analytics Commands

```bash
kcal analytics week
kcal analytics month --month 2026-02
kcal analytics range --from 2026-02-01 --to 2026-02-20
```

Insights mode:

```bash
kcal analytics insights week
kcal analytics insights range --from 2026-02-01 --to 2026-02-20 --granularity auto --out insights.md --out-format markdown
```

## Interpretation Notes

- Standard analytics reports summarize intake, exercise, net calories, category breakdowns, and adherence.
- Insights reports include period-over-period deltas, consistency metrics, streaks, and optional chart output.
- Exercise-adjusted adherence compares against effective targets that account for logged exercise.

## See Also

- [Getting Started](/kcal/getting-started/)
- [Command Reference](/kcal/reference/commands/)

## Failure and Edge Cases

- `analytics range` and `analytics insights range` require valid `--from` and `--to` dates.
- Sparse logging periods reduce usefulness of consistency and streak metrics.
- Missing active goals can lower adherence interpretability.
