package commands_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildCLI compiles the commando binary once per test run and returns its path.
func buildCLI(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "commando")
	cmd := exec.Command("go", "build", "-o", bin, "github.com/systembug/commando/cmd/commando")
	cmd.Dir = repoRoot(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build CLI failed: %v\n%s", err, out)
	}
	return bin
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// cmd/commando/commands -> backend
	return filepath.Join(dir, "..", "..", "..")
}

func TestPlanCmd_ProgressFlag_EmitsNDJSONLines(t *testing.T) {
	bin := buildCLI(t)
	left := t.TempDir()
	right := t.TempDir()
	if err := os.WriteFile(filepath.Join(left, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "sync", "plan", "--left", left, "--right", right, "--direction", "l2r", "--progress")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("plan --progress failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 NDJSON lines (progress + done), got %d: %q", len(lines), stdout.String())
	}

	for i, line := range lines {
		var payload map[string]any
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			t.Fatalf("line %d not valid JSON: %v\nline: %s", i, err, line)
		}
	}

	var last map[string]any
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &last); err != nil {
		t.Fatal(err)
	}
	if last["type"] != "done" {
		t.Errorf("expected last line type=done, got %v", last["type"])
	}
}

func TestPlanCmd_NoProgressFlag_UnchangedOutput(t *testing.T) {
	bin := buildCLI(t)
	left := t.TempDir()
	right := t.TempDir()
	if err := os.WriteFile(filepath.Join(left, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "sync", "plan", "--left", left, "--right", right, "--direction", "l2r")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("plan failed: %v", err)
	}

	var plan map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatalf("expected single JSON object without --progress, got: %v\noutput: %s", err, stdout.String())
	}
	if _, ok := plan["items"]; !ok {
		t.Errorf("expected plan JSON to have items field, got: %v", plan)
	}
}

func runCLI(t *testing.T, bin string, args ...string) (string, error) {
	t.Helper()
	var stdout bytes.Buffer
	cmd := exec.Command(bin, args...)
	cmd.Stdout = &stdout
	err := cmd.Run()
	return stdout.String(), err
}

func TestCLI_ListPlansAndReplay(t *testing.T) {
	bin := buildCLI(t)
	left, right := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(left, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	if out, err := runCLI(t, bin, "sync", "list-plans", "--left", left, "--json"); err != nil || strings.TrimSpace(out) != "[]" {
		t.Fatalf("empty list-plans: err=%v out=%q", err, out)
	}

	planOut, err := runCLI(t, bin, "sync", "plan", "--left", left, "--right", right, "--direction", "l2r")
	if err != nil {
		t.Fatal(err)
	}
	var plan struct{ ID string }
	if err := json.Unmarshal([]byte(planOut), &plan); err != nil || plan.ID == "" {
		t.Fatalf("plan output: %v %q", err, planOut)
	}

	listOut, err := runCLI(t, bin, "sync", "list-plans", "--left", left, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var list []struct {
		ID       string `json:"id"`
		Executed bool   `json:"executed"`
	}
	if err := json.Unmarshal([]byte(listOut), &list); err != nil || len(list) != 1 || list[0].ID != plan.ID || list[0].Executed {
		t.Fatalf("list-plans after plan: err=%v out=%q", err, listOut)
	}

	replayOut, err := runCLI(t, bin, "sync", "replay", "--plan-id", plan.ID, "--left", left)
	if err != nil {
		t.Fatalf("replay failed: %v %q", err, replayOut)
	}
	var result struct{ Copied int }
	if err := json.Unmarshal([]byte(replayOut), &result); err != nil || result.Copied != 1 {
		t.Fatalf("replay output: %v %q", err, replayOut)
	}
	if _, err := os.Stat(filepath.Join(right, "a.txt")); err != nil {
		t.Fatalf("replay did not copy: %v", err)
	}
}

func TestCLI_ReplayUnknownPlan_Fails(t *testing.T) {
	bin := buildCLI(t)
	if _, err := runCLI(t, bin, "sync", "replay", "--plan-id", "00000000-0000-0000-0000-000000000000", "--left", t.TempDir()); err == nil {
		t.Fatal("expected non-zero exit for unknown plan")
	}
}

// RFC-041 §11: replay (cross-session, Plan.ID) and --resume (same-session,
// sync.db) are different mechanisms and must not be combined.
func TestCLI_ReplayResume_MutuallyExclusive(t *testing.T) {
	bin := buildCLI(t)
	_, err := runCLI(t, bin, "sync", "replay", "--plan-id", "00000000-0000-0000-0000-000000000000", "--left", t.TempDir(), "--resume")
	if err == nil {
		t.Fatal("expected replay --resume to be rejected")
	}
}
