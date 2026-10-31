package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RFC-041: plans and their results live next to sync.db and reports under
// {leftRoot}/.commando (RFC-012 §3.7), hidden from compare by the default filter.
const (
	planExt   = ".json"
	resultExt = ".result.json"
)

// PlanSummary is one row of ListPlans: enough to pick a plan without loading Items.
type PlanSummary struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	LeftRoot  string    `json:"leftRoot"`
	RightRoot string    `json:"rightRoot"`
	ToCopy    int       `json:"toCopy"`
	ToDelete  int       `json:"toDelete"`
	Conflicts int       `json:"conflicts"`
	Executed  bool      `json:"executed"`
}

func plansDir(leftRoot string) string {
	return filepath.Join(leftRoot, ".commando", "plans")
}

// planFile maps an id to its file. The id arrives from the CLI, so it must be
// a UUID — anything else could escape the plans directory.
func planFile(leftRoot, id, ext string) (string, error) {
	if _, err := uuid.Parse(id); err != nil {
		return "", fmt.Errorf("invalid plan id %q", id)
	}
	return filepath.Join(plansDir(leftRoot), id+ext), nil
}

func SavePlan(leftRoot string, plan *Plan) error {
	return saveJSON(leftRoot, plan.ID, planExt, plan)
}

func LoadPlan(leftRoot, id string) (*Plan, error) {
	var plan Plan
	return &plan, loadJSON(leftRoot, id, planExt, &plan)
}

func SaveExecuteResult(leftRoot string, result *ExecuteResult) error {
	return saveJSON(leftRoot, result.PlanID, resultExt, result)
}

// LoadExecuteResult returns an os.IsNotExist error for a plan never executed.
func LoadExecuteResult(leftRoot, planID string) (*ExecuteResult, error) {
	var result ExecuteResult
	return &result, loadJSON(leftRoot, planID, resultExt, &result)
}

// ListPlans returns the plans saved under leftRoot, newest first.
func ListPlans(leftRoot string) ([]PlanSummary, error) {
	entries, err := os.ReadDir(plansDir(leftRoot))
	if os.IsNotExist(err) {
		return []PlanSummary{}, nil
	}
	if err != nil {
		return nil, err
	}
	list := []PlanSummary{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, planExt) || strings.HasSuffix(name, resultExt) {
			continue
		}
		id := strings.TrimSuffix(name, planExt)
		plan, err := LoadPlan(leftRoot, id)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		_, statErr := os.Stat(filepath.Join(plansDir(leftRoot), id+resultExt))
		list = append(list, PlanSummary{
			ID: plan.ID, CreatedAt: plan.CreatedAt, LeftRoot: plan.LeftRoot, RightRoot: plan.RightRoot,
			ToCopy: plan.ToCopy, ToDelete: plan.ToDelete, Conflicts: plan.Conflicts,
			Executed: statErr == nil,
		})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt.After(list[j].CreatedAt) })
	return list, nil
}

// saveJSON writes via temp file + rename so a crash never leaves a truncated
// plan/result that would break ListPlans or replay.
func saveJSON(leftRoot, id, ext string, v any) error {
	path, err := planFile(leftRoot, id, ext)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	tmp := path + ".commando-tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func loadJSON(leftRoot, id, ext string, v any) error {
	path, err := planFile(leftRoot, id, ext)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
