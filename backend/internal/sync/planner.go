package sync

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/systembug/commando/internal/sync/database"
	"github.com/systembug/commando/internal/sync/engine"
	"github.com/systembug/commando/internal/sync/filter"
)

// BuildPlan compares leftRoot and rightRoot and returns a sync plan.
// progress is called once per plan item after it is resolved (may be nil).
func BuildPlan(ctx context.Context, leftRoot, rightRoot string, direction Direction, opts Options, progress ProgressFn) (*Plan, error) {
	leftRoot = filepath.Clean(leftRoot)
	rightRoot = filepath.Clean(rightRoot)

	rules := opts.Filter
	if len(rules.Include) == 0 {
		rules = filter.DefaultRules()
	}
	m := filter.NewMatcher(rules)

	var indexScanned atomic.Int64
	var lastIndexEmit atomic.Int64 // unix nano
	reportIndex := func(side, rel string) {
		if progress == nil {
			return
		}
		n := indexScanned.Add(1)
		now := time.Now().UnixNano()
		if n > 1 && now-lastIndexEmit.Load() < int64(50*time.Millisecond) {
			return
		}
		lastIndexEmit.Store(now)
		progress(side+"/"+rel, ActionIndex, int(n), 0, nil)
	}

	engOpts := engine.IndexOptions{
		SymlinkMode: engine.SymlinkExclude,
		IndexProgress: func(scanned int, rel string) {
			_ = scanned
			reportIndex("left", rel)
		},
	}
	leftEntries, err := engine.IndexRoot(ctx, leftRoot, m, engOpts)
	if err != nil {
		return nil, err
	}
	engOpts.IndexProgress = func(scanned int, rel string) {
		_ = scanned
		reportIndex("right", rel)
	}
	rightEntries, err := engine.IndexRoot(ctx, rightRoot, m, engOpts)
	if err != nil {
		return nil, err
	}
	if progress != nil {
		if final := int(indexScanned.Load()); final > 0 {
			progress("", ActionIndex, final, 0, nil)
		}
	}

	var plan *Plan
	if opts.ChangesMode {
		plan, err = buildChangesPlan(ctx, leftRoot, rightRoot, leftEntries, rightEntries, direction, opts)
	} else {
		plan, err = buildDifferencesPlan(leftRoot, rightRoot, leftEntries, rightEntries, direction, opts)
	}
	if err != nil {
		return nil, err
	}

	if progress != nil {
		total := len(plan.Items)
		for i, item := range plan.Items {
			progress(item.RelativePath, item.Action, i+1, total, nil)
		}
	}

	// ponytail: best-effort like GenerateSessionLog — a read-only left root
	// must still compare/sync; it only loses replay/history for this plan.
	_ = SavePlan(leftRoot, plan)
	return plan, nil
}

func jobIDForRoots(leftRoot, rightRoot string, direction Direction) string {
	return leftRoot + "|" + rightRoot + "|" + string(direction)
}

func buildDifferencesPlan(leftRoot, rightRoot string, leftEntries, rightEntries engine.Index, direction Direction, opts Options) (*Plan, error) {

	relPaths := unionKeys(leftEntries, rightEntries)
	sort.Strings(relPaths)

	// --resume: skip items already done with matching mtime+size
	var done map[string]database.ProgressRow
	if opts.Resume && opts.DBPath != "" {
		db, err := database.Open(opts.DBPath)
		if err == nil {
			done, _ = db.DonePaths(jobIDForRoots(leftRoot, rightRoot, direction))
			db.Close()
		}
	}

	plan := &Plan{
		ID:        uuid.NewString(),
		CreatedAt: time.Now().UTC(),
		LeftRoot:  leftRoot,
		RightRoot: rightRoot,
		Direction: direction,
		Items:     make([]PlanItem, 0, len(relPaths)),
	}

	for _, rel := range relPaths {
		left, hasLeft := leftEntries[rel]
		right, hasRight := rightEntries[rel]
		// resume skip: done + mtime+size match
		if pr, ok := done[rel]; ok && hasLeft && !left.IsDir &&
			left.ModTimeUnix == pr.Mtime && left.Size == pr.Size {
			item := PlanItem{RelativePath: rel, Action: ActionSkip, Reason: "resumed"}
			plan.Items = append(plan.Items, item)
			plan.ToSkip++
			continue
		}
		item := resolveItem(rel, left, right, hasLeft, hasRight, leftRoot, rightRoot, direction, opts)
		plan.Items = append(plan.Items, item)
		switch item.Action {
		case ActionCopy:
			plan.ToCopy++
		case ActionDelete:
			plan.ToDelete++
		case ActionConflict:
			plan.Conflicts++
		default:
			plan.ToSkip++
		}
	}

	return plan, nil
}

func buildChangesPlan(ctx context.Context, leftRoot, rightRoot string, leftEntries, rightEntries engine.Index, direction Direction, opts Options) (*Plan, error) {
	dbPath := opts.DBPath
	if dbPath == "" {
		dbPath = filepath.Join(leftRoot, ".commando", "sync.db")
	}
	db, err := database.Open(opts.DBPath)
	if err != nil {
		return nil, fmt.Errorf("changes mode: %w", err)
	}
	defer db.Close()

	prev, err := db.LoadSnapshot()
	if err != nil {
		return nil, err
	}

	plan := &Plan{ID: uuid.NewString(), CreatedAt: time.Now().UTC(), LeftRoot: leftRoot, RightRoot: rightRoot, Direction: direction}

	for rel, entry := range leftEntries {
		if entry.IsDir {
			if _, existsOnRight := rightEntries[rel]; existsOnRight {
				plan.ToSkip++
				plan.Items = append(plan.Items, PlanItem{
					RelativePath: rel, Action: ActionSkip, Reason: "directory already exists on both sides",
				})
				continue
			}
			plan.ToCopy++
			plan.Items = append(plan.Items, PlanItem{
				RelativePath: rel, Action: ActionCopy,
				Source: entry.AbsolutePath, Destination: filepath.Join(rightRoot, rel),
				Reason: "missing on right", IsDir: true,
			})
			continue
		}
		snap, existed := prev[rel]
		if !existed || snap.ModTimeUnix != entry.ModTimeUnix || snap.Size != entry.Size {
			plan.ToCopy++
			plan.Items = append(plan.Items, PlanItem{
				RelativePath: rel, Action: ActionCopy,
				Source: entry.AbsolutePath, Destination: filepath.Join(rightRoot, rel),
				Reason: "changed",
			})
		} else {
			plan.ToSkip++
			plan.Items = append(plan.Items, PlanItem{
				RelativePath: rel, Action: ActionSkip, Reason: "unchanged",
			})
		}
	}
	for rel := range prev {
		if _, exists := leftEntries[rel]; !exists {
			plan.ToSkip++
			plan.Items = append(plan.Items, PlanItem{
				RelativePath: rel, Action: ActionSkip, Reason: "deleted",
			})
		}
	}
	return plan, nil
}

func unionKeys(left, right engine.Index) []string {
	seen := make(map[string]struct{}, len(left)+len(right))
	for key := range left {
		seen[key] = struct{}{}
	}
	for key := range right {
		seen[key] = struct{}{}
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	return keys
}

func resolveItem(
	rel string,
	left, right engine.Entry,
	hasLeft, hasRight bool,
	leftRoot, rightRoot string,
	direction Direction,
	opts Options,
) PlanItem {
	// VAR-05: custom action overrides take priority
	cat, _, _ := engine.Categorize(
		entryPtr(left, hasLeft), entryPtr(right, hasRight),
		engine.TimeAndSize, engine.CompareSettings{},
	)
	if act, ok := opts.CustomActions[cat]; ok {
		return itemForAction(act, rel, left, right, leftRoot, rightRoot, hasLeft, hasRight)
	}

	switch {
	case hasLeft && !hasRight:
		return planMissingSide(rel, left, leftRoot, rightRoot, direction, opts, true)
	case !hasLeft && hasRight:
		return planMissingSide(rel, right, rightRoot, leftRoot, direction, opts, false)
	default:
		return planBothSides(rel, left, right, leftRoot, rightRoot, direction, opts)
	}
}

func entryPtr(e engine.Entry, has bool) *engine.Entry {
	if !has {
		return nil
	}
	return &e
}

func itemForAction(act Action, rel string, left, right engine.Entry, leftRoot, rightRoot string, hasLeft, hasRight bool) PlanItem {
	switch act {
	case ActionCopy:
		src, dest := left, filepath.Join(rightRoot, rel)
		if !hasLeft {
			src, dest = right, filepath.Join(leftRoot, rel)
		}
		return PlanItem{RelativePath: rel, Action: ActionCopy, Source: src.AbsolutePath, Destination: dest, Reason: "custom action"}
	case ActionDelete:
		if hasLeft {
			return PlanItem{RelativePath: rel, Action: ActionDelete, Source: left.AbsolutePath, Reason: "custom action"}
		}
		return PlanItem{RelativePath: rel, Action: ActionDelete, Source: right.AbsolutePath, Reason: "custom action"}
	case ActionConflict:
		return PlanItem{RelativePath: rel, Action: ActionConflict, Source: left.AbsolutePath, Reason: "custom action"}
	default:
		return PlanItem{RelativePath: rel, Action: ActionSkip, Reason: "custom action"}
	}
}

func planMissingSide(
	rel string,
	source engine.Entry,
	sourceRoot, destRoot string,
	direction Direction,
	opts Options,
	sourceIsLeft bool,
) PlanItem {
	if direction == DirectionLeftToRight {
		if sourceIsLeft {
			return PlanItem{
				RelativePath: rel,
				Action:       ActionCopy,
				Source:       source.AbsolutePath,
				Destination:  filepath.Join(destRoot, rel),
				Reason:       "missing on right",
				IsDir:        source.IsDir,
			}
		}
		if opts.DeleteExtraneous {
			return PlanItem{
				RelativePath: rel,
				Action:       ActionDelete,
				Source:       source.AbsolutePath,
				Reason:       "extraneous on right",
				IsDir:        source.IsDir,
			}
		}
		return PlanItem{RelativePath: rel, Action: ActionSkip, Reason: "only on right; update strategy skips"}
	}

	if direction == DirectionRightToLeft {
		if !sourceIsLeft {
			return PlanItem{
				RelativePath: rel,
				Action:       ActionCopy,
				Source:       source.AbsolutePath,
				Destination:  filepath.Join(destRoot, rel),
				Reason:       "missing on left",
				IsDir:        source.IsDir,
			}
		}
		if opts.DeleteExtraneous {
			return PlanItem{
				RelativePath: rel,
				Action:       ActionDelete,
				Source:       source.AbsolutePath,
				Reason:       "extraneous on left",
				IsDir:        source.IsDir,
			}
		}
		return PlanItem{RelativePath: rel, Action: ActionSkip, Reason: "only on left; update strategy skips"}
	}

	return PlanItem{
		RelativePath: rel,
		Action:       ActionCopy,
		Source:       source.AbsolutePath,
		Destination:  filepath.Join(destRoot, rel),
		Reason:       "missing on other pane",
		IsDir:        source.IsDir,
	}
}

func planBothSides(
	rel string,
	left, right engine.Entry,
	leftRoot, rightRoot string,
	direction Direction,
	opts Options,
) PlanItem {
	if left.IsDir || right.IsDir {
		return PlanItem{RelativePath: rel, Action: ActionSkip, Reason: "directory already exists on both sides"}
	}

	mode := engine.TimeAndSize
	if opts.UseChecksum {
		mode = engine.Content
	}
	if eq, _ := engine.IsEqual(left, right, mode, engine.CompareSettings{}); eq {
		return PlanItem{RelativePath: rel, Action: ActionSkip, Reason: "already in sync"}
	}

	switch direction {
	case DirectionLeftToRight:
		return PlanItem{
			RelativePath: rel,
			Action:       ActionCopy,
			Source:       left.AbsolutePath,
			Destination:  filepath.Join(rightRoot, rel),
			Reason:       "left newer or different",
		}
	case DirectionRightToLeft:
		return PlanItem{
			RelativePath: rel,
			Action:       ActionCopy,
			Source:       right.AbsolutePath,
			Destination:  filepath.Join(leftRoot, rel),
			Reason:       "right newer or different",
		}
	default:
		return planBidirectional(rel, left, right, leftRoot, rightRoot)
	}
}

func planBidirectional(rel string, left, right engine.Entry, leftRoot, rightRoot string) PlanItem {
	switch {
	case left.ModTimeUnix > right.ModTimeUnix:
		return PlanItem{
			RelativePath: rel,
			Action:       ActionCopy,
			Source:       left.AbsolutePath,
			Destination:  filepath.Join(rightRoot, rel),
			Reason:       "left newer",
		}
	case right.ModTimeUnix > left.ModTimeUnix:
		return PlanItem{
			RelativePath: rel,
			Action:       ActionCopy,
			Source:       right.AbsolutePath,
			Destination:  filepath.Join(leftRoot, rel),
			Reason:       "right newer",
		}
	default:
		return PlanItem{
			RelativePath: rel,
			Action:       ActionConflict,
			Source:       left.AbsolutePath,
			Destination:  right.AbsolutePath,
			Reason:       "same mtime but different size",
		}
	}
}
