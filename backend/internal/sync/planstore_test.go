package sync_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/systembug/commando/internal/sync"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newPlan(left string, created time.Time) *sync.Plan {
	return &sync.Plan{
		ID:        uuid.NewString(),
		CreatedAt: created.UTC(),
		LeftRoot:  left,
		RightRoot: "/right",
		Direction: sync.DirectionLeftToRight,
		Items: []sync.PlanItem{
			{RelativePath: "a.txt", Action: sync.ActionCopy, Source: "/l/a.txt", Destination: "/r/a.txt", Reason: "missing"},
			{RelativePath: "d", Action: sync.ActionCopy, IsDir: true},
		},
		ToCopy: 2,
	}
}

func TestSavePlan_LoadPlan_RoundTrip(t *testing.T) {
	left := t.TempDir()
	plan := newPlan(left, time.Now().Truncate(time.Second))

	if err := sync.SavePlan(left, plan); err != nil {
		t.Fatal(err)
	}
	got, err := sync.LoadPlan(left, plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != plan.ID || !got.CreatedAt.Equal(plan.CreatedAt) || got.LeftRoot != plan.LeftRoot ||
		got.ToCopy != plan.ToCopy || len(got.Items) != 2 || got.Items[0] != plan.Items[0] || got.Items[1] != plan.Items[1] {
		t.Fatalf("round trip mismatch:\nwant %+v\ngot  %+v", plan, got)
	}
}

func TestLoadPlan_RejectsNonUUIDID(t *testing.T) {
	left := t.TempDir()
	for _, id := range []string{"../../etc/passwd", "", "abc"} {
		if _, err := sync.LoadPlan(left, id); err == nil {
			t.Errorf("LoadPlan(%q): expected error", id)
		}
		if _, err := sync.LoadExecuteResult(left, id); err == nil {
			t.Errorf("LoadExecuteResult(%q): expected error", id)
		}
	}
}

func TestBuildPlan_AutoSavesPlan(t *testing.T) {
	left, right := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(left, "a.txt"), "hello")

	plan, err := sync.BuildPlan(context.Background(), left, right, sync.DirectionLeftToRight, sync.Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(left, ".commando", "plans", plan.ID+".json")); err != nil {
		t.Fatalf("plan not persisted: %v", err)
	}
}

func TestExecute_AutoSavesExecuteResult(t *testing.T) {
	left, right := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(left, "a.txt"), "hello")

	plan, err := sync.BuildPlan(context.Background(), left, right, sync.DirectionLeftToRight, sync.Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sync.Execute(context.Background(), plan, sync.Options{}, nil); err != nil {
		t.Fatal(err)
	}
	res, err := sync.LoadExecuteResult(left, plan.ID)
	if err != nil {
		t.Fatalf("result not persisted: %v", err)
	}
	if len(res.DoneItems) != 1 || res.DoneItems[0] != "a.txt" {
		t.Fatalf("expected DoneItems [a.txt], got %v", res.DoneItems)
	}
}

func TestExecute_DryRun_DoesNotSaveResult(t *testing.T) {
	left, right := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(left, "a.txt"), "hello")

	plan, err := sync.BuildPlan(context.Background(), left, right, sync.DirectionLeftToRight, sync.Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sync.Execute(context.Background(), plan, sync.Options{DryRun: true}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := sync.LoadExecuteResult(left, plan.ID); !os.IsNotExist(err) {
		t.Fatalf("dry run must not persist a result (it would mark items done), got err=%v", err)
	}
}

// Review note §13.2: DoneItems must count only real successes, so a partial
// failure cannot be double-counted or mistaken for done on replay.
func TestExecute_ResultCarriesDoneItemsAndSnapshots(t *testing.T) {
	left, right := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(left, "ok.txt"), "ok")
	writeFile(t, filepath.Join(left, "bad.txt"), "bad")

	plan, err := sync.BuildPlan(context.Background(), left, right, sync.DirectionLeftToRight, sync.Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Make bad.txt fail: its source vanishes after planning.
	if err := os.Remove(filepath.Join(left, "bad.txt")); err != nil {
		t.Fatal(err)
	}

	res, err := sync.Execute(context.Background(), plan, sync.Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Errors) != 1 {
		t.Fatalf("expected 1 error, got %v", res.Errors)
	}
	if len(res.DoneItems) != res.Copied || len(res.DoneItems) != 1 || res.DoneItems[0] != "ok.txt" {
		t.Fatalf("DoneItems must equal real successes: copied=%d done=%v", res.Copied, res.DoneItems)
	}
	snap, ok := res.ItemSnapshots["ok.txt"]
	if !ok || snap.Size != 2 || snap.ModTimeUnix == 0 {
		t.Fatalf("expected snapshot for ok.txt, got %+v", res.ItemSnapshots)
	}
}

func TestReplayPlan_FirstRun_ExecutesAllItems(t *testing.T) {
	left, right := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(left, "a.txt"), "a")
	writeFile(t, filepath.Join(left, "b.txt"), "b")

	plan, err := sync.BuildPlan(context.Background(), left, right, sync.DirectionLeftToRight, sync.Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := sync.ReplayPlan(context.Background(), left, plan.ID, sync.Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Copied != 2 {
		t.Fatalf("expected 2 copied, got %d (%v)", res.Copied, res.Errors)
	}
}

func TestReplayPlan_SecondRun_SkipsDoneItems(t *testing.T) {
	left, right := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(left, "a.txt"), "a")
	writeFile(t, filepath.Join(left, "b.txt"), "b")

	plan, err := sync.BuildPlan(context.Background(), left, right, sync.DirectionLeftToRight, sync.Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// First run: b.txt fails because its destination parent is a file.
	writeFile(t, filepath.Join(right, "b.txt.blocker"), "")
	blockB := plan.Items[1]
	if blockB.RelativePath != "b.txt" {
		t.Fatalf("unexpected plan order: %+v", plan.Items)
	}
	plan.Items[1].Destination = filepath.Join(right, "b.txt.blocker", "b.txt")
	if err := sync.SavePlan(left, plan); err != nil {
		t.Fatal(err)
	}
	first, err := sync.ReplayPlan(context.Background(), left, plan.ID, sync.Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.Copied != 1 || len(first.Errors) != 1 {
		t.Fatalf("first run: copied=%d errors=%v", first.Copied, first.Errors)
	}

	// Fix b.txt's destination, replay again: only b.txt runs.
	plan.Items[1] = blockB
	if err := sync.SavePlan(left, plan); err != nil {
		t.Fatal(err)
	}
	var seen []string
	second, err := sync.ReplayPlan(context.Background(), left, plan.ID, sync.Options{}, func(rel string, _ sync.Action, _, _ int, _ error) {
		seen = append(seen, rel)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(seen) != 1 || seen[0] != "b.txt" || second.Copied != 1 {
		t.Fatalf("second run should only touch b.txt: seen=%v copied=%d", seen, second.Copied)
	}
	// DoneItems accumulate across replays so a third replay is a no-op.
	saved, err := sync.LoadExecuteResult(left, plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.DoneItems) != 2 {
		t.Fatalf("expected cumulative DoneItems [a.txt b.txt], got %v", saved.DoneItems)
	}
}

func TestReplayPlan_SkipsAlreadyGoneSource(t *testing.T) {
	left, right := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(left, "a.txt"), "a")
	writeFile(t, filepath.Join(left, "b.txt"), "b")

	plan, err := sync.BuildPlan(context.Background(), left, right, sync.DirectionLeftToRight, sync.Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(left, "a.txt")); err != nil {
		t.Fatal(err)
	}
	res, err := sync.ReplayPlan(context.Background(), left, plan.ID, sync.Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Errors) != 1 || res.Copied != 1 {
		t.Fatalf("gone source should fail alone: copied=%d errors=%v", res.Copied, res.Errors)
	}
}

func TestReplayPlan_UnknownPlanID_ReturnsError(t *testing.T) {
	if _, err := sync.ReplayPlan(context.Background(), t.TempDir(), uuid.NewString(), sync.Options{}, nil); err == nil {
		t.Fatal("expected error for unknown plan id")
	}
}

func TestReplayPlan_TargetModifiedSinceLastRun_SkipsOrErrors(t *testing.T) {
	left, right := t.TempDir(), t.TempDir()
	src := filepath.Join(left, "a.txt")
	writeFile(t, src, "v1")

	plan, err := sync.BuildPlan(context.Background(), left, right, sync.DirectionLeftToRight, sync.Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Attempt 1 fails (destination blocked) but snapshots the source.
	dest := plan.Items[0].Destination
	if err := os.MkdirAll(filepath.Join(dest, "blocker"), 0o755); err != nil {
		t.Fatal(err)
	}
	first, err := sync.ReplayPlan(context.Background(), left, plan.ID, sync.Options{}, nil)
	if err != nil || len(first.Errors) != 1 {
		t.Fatalf("first run should fail once: err=%v errors=%v", err, first.Errors)
	}
	if err := os.RemoveAll(dest); err != nil {
		t.Fatal(err)
	}

	// Source changes between runs: replay must not copy the unreviewed content.
	writeFile(t, src, "v2-changed")
	res, err := sync.ReplayPlan(context.Background(), left, plan.ID, sync.Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Copied != 0 || len(res.Errors) != 1 {
		t.Fatalf("expected stale-source error, got copied=%d errors=%v", res.Copied, res.Errors)
	}
	if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
		t.Fatalf("stale source must not be copied to %s", dest)
	}

	// ErrorMode=stop aborts on stale source.
	if _, err := sync.ReplayPlan(context.Background(), left, plan.ID, sync.Options{ErrorMode: "stop"}, nil); err == nil {
		t.Fatal("expected error with ErrorMode=stop")
	}
}

// Review note §13.3: a missing snapshot (never attempted, or hand-edited
// result) downgrades to "run the item", never blocks.
func TestReplayPlan_MissingSnapshot_RunsItemNotBlocks(t *testing.T) {
	left, right := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(left, "a.txt"), "a")

	plan, err := sync.BuildPlan(context.Background(), left, right, sync.DirectionLeftToRight, sync.Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := sync.SaveExecuteResult(left, &sync.ExecuteResult{PlanID: plan.ID}); err != nil {
		t.Fatal(err)
	}
	res, err := sync.ReplayPlan(context.Background(), left, plan.ID, sync.Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Copied != 1 || len(res.Errors) != 0 {
		t.Fatalf("missing snapshot must not block: copied=%d errors=%v", res.Copied, res.Errors)
	}
}

func TestListPlans_OrderedByCreatedAtDesc(t *testing.T) {
	left := t.TempDir()
	base := time.Now()
	older, newer := newPlan(left, base.Add(-time.Hour)), newPlan(left, base)
	for _, p := range []*sync.Plan{older, newer} {
		if err := sync.SavePlan(left, p); err != nil {
			t.Fatal(err)
		}
	}
	list, err := sync.ListPlans(left)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != newer.ID || list[1].ID != older.ID {
		t.Fatalf("expected newest first, got %+v", list)
	}
	if list[0].ToCopy != 2 || list[0].RightRoot != "/right" {
		t.Fatalf("summary fields missing: %+v", list[0])
	}
}

func TestListPlans_ExecutedFlagReflectsResultFile(t *testing.T) {
	left := t.TempDir()
	ran, idle := newPlan(left, time.Now()), newPlan(left, time.Now().Add(-time.Minute))
	for _, p := range []*sync.Plan{ran, idle} {
		if err := sync.SavePlan(left, p); err != nil {
			t.Fatal(err)
		}
	}
	if err := sync.SaveExecuteResult(left, &sync.ExecuteResult{PlanID: ran.ID}); err != nil {
		t.Fatal(err)
	}
	list, err := sync.ListPlans(left)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || !list[0].Executed || list[1].Executed {
		t.Fatalf("executed flags wrong: %+v", list)
	}
}

func TestListPlans_EmptyDir_ReturnsEmptySlice(t *testing.T) {
	list, err := sync.ListPlans(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if list == nil || len(list) != 0 {
		t.Fatalf("expected empty non-nil slice, got %#v", list)
	}
}
