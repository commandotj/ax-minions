package sync

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	databaseapi "github.com/systembug/commando/internal/sync/database"
	dels "github.com/systembug/commando/internal/sync/delete"
	"github.com/systembug/commando/internal/sync/engine"
)

// ProgressFn is called after each item is processed. relPath is the
// relative path being processed. action describes what happened.
type ProgressFn func(relPath string, action Action, done, total int, err error)

// errStaleSource marks a replayed item whose source changed since an earlier
// attempt: the user reviewed the old content, so it is not copied/deleted.
var errStaleSource = errors.New("stale-source")

// Execute runs a sync plan. ErrorMode="stop" aborts on first failure.
// progress is called after each item (may be nil). The result is saved under
// the plan's left root (RFC-041) unless DryRun, even on abort, so ReplayPlan
// can pick up the unfinished items.
func Execute(ctx context.Context, plan *Plan, opts Options, progress ProgressFn) (*ExecuteResult, error) {
	result, err := execute(ctx, plan, opts, progress, nil)
	if !opts.DryRun && plan.LeftRoot != "" {
		// ponytail: best-effort like GenerateSessionLog; a lost result only
		// means replay re-runs every item.
		_ = SaveExecuteResult(plan.LeftRoot, result)
	}
	return result, err
}

// ReplayPlan re-runs a saved plan, skipping items an earlier run confirmed
// done and refusing items whose source changed since they were last attempted.
// DoneItems/Restorable/ItemSnapshots in the saved result accumulate.
func ReplayPlan(ctx context.Context, leftRoot, planID string, opts Options, progress ProgressFn) (*ExecuteResult, error) {
	plan, err := LoadPlan(leftRoot, planID)
	if err != nil {
		return nil, err
	}
	prev, err := LoadExecuteResult(leftRoot, planID)
	if os.IsNotExist(err) {
		prev = &ExecuteResult{}
	} else if err != nil {
		return nil, err
	}

	done := make(map[string]bool, len(prev.DoneItems))
	for _, rel := range prev.DoneItems {
		done[rel] = true
	}
	replay := *plan
	replay.Items = nil
	for _, item := range plan.Items {
		if !done[item.RelativePath] {
			replay.Items = append(replay.Items, item)
		}
	}

	result, runErr := execute(ctx, &replay, opts, progress, prev.ItemSnapshots)
	if opts.DryRun {
		return result, runErr
	}
	result.DoneItems = append(prev.DoneItems, result.DoneItems...)
	result.Restorable = append(prev.Restorable, result.Restorable...)
	for rel, snap := range prev.ItemSnapshots {
		if _, ok := result.ItemSnapshots[rel]; !ok {
			result.ItemSnapshots[rel] = snap
		}
	}
	if err := SaveExecuteResult(leftRoot, result); err != nil && runErr == nil {
		runErr = err
	}
	return result, runErr
}

// snapshotSource fingerprints the source an item is about to act on and
// rejects it if it differs from an earlier attempt's fingerprint.
func snapshotSource(item PlanItem, prev map[string]ItemSnapshot, result *ExecuteResult) error {
	info, err := os.Stat(item.Source)
	if err != nil {
		return nil // a missing source surfaces as the copy/delete error itself
	}
	snap := ItemSnapshot{SourcePath: item.Source, Size: info.Size(), ModTimeUnix: info.ModTime().Unix()}
	if old, ok := prev[item.RelativePath]; ok && (old.Size != snap.Size || old.ModTimeUnix != snap.ModTimeUnix) {
		return errStaleSource // keep the old snapshot: the new content is still unreviewed
	}
	result.ItemSnapshots[item.RelativePath] = snap
	return nil
}

func execute(ctx context.Context, plan *Plan, opts Options, progress ProgressFn, prevSnaps map[string]ItemSnapshot) (*ExecuteResult, error) {
	result := &ExecuteResult{PlanID: plan.ID, ItemSnapshots: map[string]ItemSnapshot{}}
	total := len(plan.Items)

	for i, item := range plan.Items {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}
		switch item.Action {
		case ActionSkip, ActionConflict:
			result.Skipped++
			if progress != nil {
				progress(item.RelativePath, item.Action, i+1, total, nil)
			}
			continue
		case ActionCopy:
			if opts.DryRun {
				result.Copied++
				if progress != nil {
					progress(item.RelativePath, item.Action, i+1, total, nil)
				}
				continue
			}
			if item.IsDir {
				mkErr := os.MkdirAll(item.Destination, 0o755)
				if progress != nil {
					progress(item.RelativePath, item.Action, i+1, total, mkErr)
				}
				if mkErr != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", item.RelativePath, mkErr))
					if opts.ErrorMode == "stop" {
						return result, fmt.Errorf("mkdir %s: %w", item.RelativePath, mkErr)
					}
					continue
				}
				result.Copied++
				result.DoneItems = append(result.DoneItems, item.RelativePath)
				continue
			}
			if staleErr := snapshotSource(item, prevSnaps, result); staleErr != nil {
				if progress != nil {
					progress(item.RelativePath, item.Action, i+1, total, staleErr)
				}
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", item.RelativePath, staleErr))
				if opts.ErrorMode == "stop" {
					return result, fmt.Errorf("copy %s: %w", item.RelativePath, staleErr)
				}
				continue
			}
			cpErr := copyFileAtomic(item.Source, item.Destination)
			if progress != nil {
				progress(item.RelativePath, item.Action, i+1, total, cpErr)
			}
			if cpErr != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", item.RelativePath, cpErr))
				if opts.ErrorMode == "stop" {
					return result, fmt.Errorf("copy %s: %w", item.RelativePath, cpErr)
				}
				continue
			}
			result.Copied++
			if opts.DBPath != "" {
				info, _ := os.Stat(item.Source)
				if info != nil {
					saveProgress(opts.DBPath, jobID(plan), item.RelativePath, "copy", info.ModTime().Unix(), info.Size())
				}
			}
			if opts.VerifyCopies {
				if !filesEqual(item.Source, item.Destination) {
					result.Errors = append(result.Errors, fmt.Sprintf("%s: verify failed", item.RelativePath))
					if opts.ErrorMode == "stop" {
						return result, fmt.Errorf("verify %s: mismatch", item.RelativePath)
					}
					continue
				}
			}
			result.DoneItems = append(result.DoneItems, item.RelativePath)
		case ActionDelete:
			if opts.DryRun {
				result.Deleted++
				continue
			}
			if staleErr := snapshotSource(item, prevSnaps, result); staleErr != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", item.RelativePath, staleErr))
				if opts.ErrorMode == "stop" {
					return result, fmt.Errorf("delete %s: %w", item.RelativePath, staleErr)
				}
				continue
			}
			method := deleteMethod(opts.DeleteMethod)
			restorePath, err := dels.Delete(ctx, item.Source, method, opts.VersionDir, "")
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", item.RelativePath, err))
				if opts.ErrorMode == "stop" {
					return result, fmt.Errorf("delete %s: %w", item.RelativePath, err)
				}
				continue
			}
			if opts.VerifyCopies && (method == dels.Versioning || method == dels.VersionReplace) {
				if _, statErr := os.Stat(restorePath); statErr != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("%s: backup missing after delete", item.RelativePath))
					if opts.ErrorMode == "stop" {
						return result, fmt.Errorf("verify delete %s: backup missing", item.RelativePath)
					}
					continue
				}
			}
			result.Deleted++
			result.DoneItems = append(result.DoneItems, item.RelativePath)
			if restorePath != "" {
				result.Restorable = append(result.Restorable, RestoreEntry{
					RelativePath: item.RelativePath, OriginalPath: item.Source, BackupPath: restorePath,
				})
			}
		}
	}

	if len(result.Errors) == 0 {
		_ = GenerateSessionLog(plan, result)
	}
	return result, nil
}

// WriteSnapshot writes the current index to the DB for changes-mode use.
func WriteSnapshot(dbPath string, idx engine.Index) error {
	db, err := databaseapi.Open(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	db.ClearSnapshot()
	for rel, entry := range idx {
		if entry.IsDir {
			continue
		}
		info, err := os.Stat(entry.AbsolutePath)
		if err != nil {
			continue
		}
		db.UpsertSnapshot(databaseapi.SnapshotRow{
			RelativePath: rel, ModTimeUnix: entry.ModTimeUnix,
			Size: entry.Size, FileID: databaseapi.FileID(info),
		})
	}
	return nil
}

// copyFileAtomic copies source to a temporary file next to destination, then
// renames it atomically. Destination mode and mtime are preserved from source.
func copyFileAtomic(source, destination string) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}

	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()

	srcInfo, err := src.Stat()
	if err != nil {
		return err
	}

	tmpPath := destination + ".commando-tmp"
	dst, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode().Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(dst, src)
	dst.Close()
	if copyErr != nil {
		os.Remove(tmpPath)
		return copyErr
	}

	if err := os.Chtimes(tmpPath, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return os.Rename(tmpPath, destination)
}

func filesEqual(a, b string) bool {
	fa, err := os.Open(a)
	if err != nil {
		return false
	}
	defer fa.Close()
	fb, err := os.Open(b)
	if err != nil {
		return false
	}
	defer fb.Close()

	bufA := make([]byte, 64*1024)
	bufB := make([]byte, 64*1024)
	for {
		na, _ := fa.Read(bufA)
		nb, _ := fb.Read(bufB)
		if na != nb || !bytes.Equal(bufA[:na], bufB[:nb]) {
			return false
		}
		if na == 0 {
			return true
		}
	}
}

func deleteMethod(s string) dels.Method {
	switch s {
	case "trash":
		return dels.Trash
	case "versioning":
		return dels.Versioning
	case "versionreplace":
		return dels.VersionReplace
	default:
		return dels.Permanent
	}
}

func saveProgress(dbPath, jobID, relPath, action string, mtime, size int64) {
	db, err := databaseapi.Open(dbPath)
	if err != nil {
		return
	}
	defer db.Close()
	db.MarkDone(jobID, relPath, action, mtime, size)
}

func jobID(plan *Plan) string {
	return plan.LeftRoot + "|" + plan.RightRoot + "|" + string(plan.Direction)
}
