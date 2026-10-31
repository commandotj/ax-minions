package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	gosync "sync"

	"github.com/spf13/cobra"
	"github.com/systembug/commando/internal/sync"
	databaseapi "github.com/systembug/commando/internal/sync/database"
	"github.com/systembug/commando/internal/sync/filter"
)

func init() {
	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Plan or run two-pane folder sync",
	}

	planCmd := &cobra.Command{
		Use:   "plan",
		Short: "Build a sync plan without writing files",
		RunE: func(cmd *cobra.Command, _ []string) error {
			left, _ := cmd.Flags().GetString("left")
			right, _ := cmd.Flags().GetString("right")
			dirFlag, _ := cmd.Flags().GetString("direction")
			include, _ := cmd.Flags().GetString("include")
			exclude, _ := cmd.Flags().GetString("exclude")
			showProgress, _ := cmd.Flags().GetBool("progress")

			dir, err := parseDirection(dirFlag)
			if err != nil {
				return err
			}

			rules := filterRules(include, exclude)
			if err := rules.Validate(); err != nil {
				return err
			}

			var progressFn sync.ProgressFn
			if showProgress {
				progressFn = func(rel string, act sync.Action, done, total int, itemErr error) {
					evt := map[string]any{
						"type": "progress", "file": rel, "action": string(act),
						"done": done, "total": total,
					}
					if itemErr != nil {
						evt["error"] = itemErr.Error()
					}
					writeProgressLine(evt)
				}
			}

			plan, err := sync.BuildPlan(context.Background(), left, right, dir, sync.Options{
				DryRun: true,
				Filter: rules,
			}, progressFn)
			if err != nil {
				if showProgress {
					emitSyncDone("error", nil, err)
					return nil
				}
				return err
			}

			if showProgress {
				emitSyncDone("done", plan, nil)
				return nil
			}

			enc := json.NewEncoder(os.Stdout)
			return enc.Encode(plan)
		},
	}
	planCmd.Flags().String("left", "", "left pane root")
	planCmd.Flags().String("right", "", "right pane root")
	planCmd.Flags().String("direction", "both", "sync direction: l2r, r2l, both")
	planCmd.Flags().String("include", "", "include glob pattern (comma-separated, default: **)")
	planCmd.Flags().String("exclude", "", "exclude glob pattern (comma-separated)")
	planCmd.Flags().Bool("progress", false, "output NDJSON progress lines")
	_ = planCmd.MarkFlagRequired("left")
	_ = planCmd.MarkFlagRequired("right")

	listPlansCmd := &cobra.Command{
		Use:   "list-plans",
		Short: "List saved sync plans",
		RunE: func(cmd *cobra.Command, _ []string) error {
			left, _ := cmd.Flags().GetString("left")
			plans, err := sync.ListPlans(left)
			if err != nil {
				return err
			}
			return json.NewEncoder(os.Stdout).Encode(plans)
		},
	}
	listPlansCmd.Flags().String("left", "", "left pane root")
	listPlansCmd.Flags().Bool("json", false, "output JSON")
	_ = listPlansCmd.MarkFlagRequired("left")

	replayCmd := &cobra.Command{
		Use:   "replay",
		Short: "Replay unfinished items from a saved sync plan",
		RunE: func(cmd *cobra.Command, _ []string) error {
			resume, _ := cmd.Flags().GetBool("resume")
			if resume {
				return fmt.Errorf("--resume cannot be combined with sync replay")
			}

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			left, _ := cmd.Flags().GetString("left")
			planID, _ := cmd.Flags().GetString("plan-id")
			result, err := sync.ReplayPlan(ctx, left, planID, sync.Options{}, nil)
			if err != nil {
				return err
			}
			if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
				return err
			}
			if len(result.Errors) > 0 {
				return fmt.Errorf("%d errors", len(result.Errors))
			}
			return nil
		},
	}
	replayCmd.Flags().String("plan-id", "", "saved sync plan ID")
	replayCmd.Flags().String("left", "", "left pane root")
	replayCmd.Flags().Bool("resume", false, "resume an active sync run (incompatible with replay)")
	_ = replayCmd.MarkFlagRequired("plan-id")
	_ = replayCmd.MarkFlagRequired("left")

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Execute a sync between two folders",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			left, _ := cmd.Flags().GetString("left")
			right, _ := cmd.Flags().GetString("right")
			dirFlag, _ := cmd.Flags().GetString("direction")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			include, _ := cmd.Flags().GetString("include")
			exclude, _ := cmd.Flags().GetString("exclude")
			showProgress, _ := cmd.Flags().GetBool("progress")
			resume, _ := cmd.Flags().GetBool("resume")
			reset, _ := cmd.Flags().GetBool("reset")
			deleteExtra, _ := cmd.Flags().GetBool("delete-extraneous")
			errorMode, _ := cmd.Flags().GetString("error-mode")
			deleteMethod, _ := cmd.Flags().GetString("delete-method")
			versionDir, _ := cmd.Flags().GetString("version-dir")
			verify, _ := cmd.Flags().GetBool("verify")

			dir, err := parseDirection(dirFlag)
			if err != nil {
				return err
			}

			rules := filterRules(include, exclude)
			if err := rules.Validate(); err != nil {
				return err
			}

			opts := sync.Options{
				DryRun:           dryRun,
				Filter:           rules,
				DBPath:           filepath.Join(left, ".commando", "sync.db"),
				Resume:           resume,
				Reset:            reset,
				DeleteExtraneous: deleteExtra,
				ErrorMode:        errorMode,
				DeleteMethod:     deleteMethod,
				VersionDir:       versionDir,
				VerifyCopies:     verify,
			}
			if reset {
				clearResume(opts.DBPath, jobIDForSync(left, right, dir))
			}

			plan, err := sync.BuildPlan(ctx, left, right, dir, opts, nil)
			if err != nil {
				if showProgress {
					emitSyncDone("error", nil, err)
					return nil
				}
				return err
			}

			var progressFn sync.ProgressFn
			if showProgress {
				progressFn = func(rel string, act sync.Action, done, total int, itemErr error) {
					evt := map[string]any{
						"type": "progress", "file": rel, "action": string(act),
						"done": done, "total": total,
					}
					if itemErr != nil {
						evt["error"] = itemErr.Error()
					}
					writeProgressLine(evt)
				}
			}

			result, err := sync.Execute(ctx, plan, opts, progressFn)
			if err != nil {
				if showProgress {
					emitSyncDone("error", nil, err)
					return nil
				}
				fmt.Fprintln(os.Stderr, "error:", err)
				return nil
			}
			if len(result.Errors) == 0 {
				clearResume(opts.DBPath, jobIDForSync(left, right, dir))
			}

			if showProgress {
				emitSyncDone("done", result, nil)
				return nil
			}

			enc := json.NewEncoder(os.Stdout)
			if err := enc.Encode(result); err != nil {
				return err
			}
			if len(result.Errors) > 0 {
				return fmt.Errorf("%d errors", len(result.Errors))
			}
			return nil
		},
	}
	runCmd.Flags().String("left", "", "left pane root")
	runCmd.Flags().String("right", "", "right pane root")
	runCmd.Flags().String("direction", "both", "sync direction: l2r, r2l, both")
	runCmd.Flags().Bool("dry-run", false, "show actions without writing")
	runCmd.Flags().String("include", "", "include glob pattern (comma-separated, default: **)")
	runCmd.Flags().String("exclude", "", "exclude glob pattern (comma-separated)")
	runCmd.Flags().Bool("progress", false, "output NDJSON progress lines")
	runCmd.Flags().Bool("resume", false, "skip already-completed items")
	runCmd.Flags().Bool("reset", false, "clear progress and start fresh")
	runCmd.Flags().Bool("delete-extraneous", false, "delete files not on source")
	runCmd.Flags().String("error-mode", "ignore", "error handling: stop or ignore")
	runCmd.Flags().String("delete-method", "permanent", "delete method: permanent, trash, or versioning")
	runCmd.Flags().String("version-dir", "", "versioning backup directory")
	runCmd.Flags().Bool("verify", false, "verify copied and deleted files")
	_ = runCmd.MarkFlagRequired("left")
	_ = runCmd.MarkFlagRequired("right")

	syncCmd.AddCommand(planCmd, runCmd, listPlansCmd, replayCmd)
	rootCmd.AddCommand(syncCmd)
}

func filterRules(include, exclude string) filter.FilterRules {
	rules := filter.DefaultRules()
	if include != "" {
		rules.Include = splitTrim(include)
	}
	if exclude != "" {
		rules.Exclude = splitTrim(exclude)
	}
	return rules
}

func splitTrim(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func emitSyncDone(status string, result any, err error) {
	evt := map[string]any{"type": "done", "status": status}
	if result != nil {
		evt["result"] = result
	}
	if err != nil {
		evt["error"] = err.Error()
	}
	writeProgressLine(evt)
}

var progressStdoutMu gosync.Mutex

func writeProgressLine(evt map[string]any) {
	b, err := json.Marshal(evt)
	if err != nil {
		return
	}
	progressStdoutMu.Lock()
	defer progressStdoutMu.Unlock()
	_, _ = os.Stdout.Write(append(b, '\n'))
}

func parseDirection(value string) (sync.Direction, error) {
	switch value {
	case "l2r", "left-to-right":
		return sync.DirectionLeftToRight, nil
	case "r2l", "right-to-left":
		return sync.DirectionRightToLeft, nil
	case "both", "bidirectional":
		return sync.DirectionBidirectional, nil
	default:
		return "", fmt.Errorf("unknown direction: %q (expected l2r, r2l, or both)", value)
	}
}

func jobIDForSync(left, right string, dir sync.Direction) string {
	return left + "|" + right + "|" + string(dir)
}

func clearResume(dbPath, jobID string) {
	db, err := databaseapi.Open(dbPath)
	if err != nil {
		return
	}
	defer db.Close()
	db.ClearProgress(jobID)
}
