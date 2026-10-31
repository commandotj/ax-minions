// Package sync implements two-pane folder synchronization planning and execution.
package sync

import (
	"time"

	"github.com/systembug/commando/internal/sync/engine"
	"github.com/systembug/commando/internal/sync/filter"
)

// Direction controls how changes flow between the left and right pane roots.
type Direction string

const (
	DirectionLeftToRight  Direction = "left-to-right"
	DirectionRightToLeft  Direction = "right-to-left"
	DirectionBidirectional Direction = "bidirectional"
)

// Action describes what should happen to a single relative path.
type Action string

const (
	ActionCopy     Action = "copy"
	ActionDelete   Action = "delete"
	ActionSkip     Action = "skip"
	ActionConflict Action = "conflict"
	// ActionIndex is progress-only: filesystem walk during plan build.
	ActionIndex Action = "index"
)

// PlanItem is one row in a sync plan shown to the user before execution.
type PlanItem struct {
	RelativePath string `json:"relativePath"`
	Action       Action `json:"action"`
	Source       string `json:"source"`
	Destination  string `json:"destination"`
	Reason       string `json:"reason"`
	IsDir        bool   `json:"isDir,omitempty"`
}

// Plan is the full diff between two pane roots.
type Plan struct {
	ID         string     `json:"id"`
	CreatedAt  time.Time  `json:"createdAt"`
	LeftRoot   string     `json:"leftRoot"`
	RightRoot  string     `json:"rightRoot"`
	Direction  Direction  `json:"direction"`
	Items      []PlanItem `json:"items"`
	Conflicts  int        `json:"conflicts"`
	ToCopy     int        `json:"toCopy"`
	ToDelete   int        `json:"toDelete"`
	ToSkip     int        `json:"toSkip"`
}

// ExecuteResult summarizes a completed sync run.
type ExecuteResult struct {
	PlanID     string         `json:"planId"`
	Copied     int            `json:"copied"`
	Skipped    int            `json:"skipped"`
	Deleted    int            `json:"deleted"`
	Errors     []string       `json:"errors"`
	Restorable []RestoreEntry `json:"restorable,omitempty"`
	// DoneItems lists confirmed successes (not failures), so replay re-runs
	// everything else — failed and never-reached items alike. Cumulative
	// across replays of the same plan.
	DoneItems []string `json:"doneItems,omitempty"`
	// ItemSnapshots holds the source state each attempted copy/delete acted
	// on; replay refuses an item whose source changed since (stale-source).
	ItemSnapshots map[string]ItemSnapshot `json:"itemSnapshots,omitempty"`
}

// ItemSnapshot is the size/mtime fingerprint of a plan item's source.
type ItemSnapshot struct {
	SourcePath  string `json:"sourcePath"`
	Size        int64  `json:"size"`
	ModTimeUnix int64  `json:"modTimeUnix"`
}

type RestoreEntry struct {
	RelativePath string `json:"relativePath"`
	OriginalPath string `json:"originalPath"`
	BackupPath   string `json:"backupPath"`
}

// Options tune planner and executor behavior.
type Options struct {
	DeleteExtraneous bool                          `json:"deleteExtraneous"`
	DryRun           bool                          `json:"dryRun"`
	UseChecksum      bool                          `json:"useChecksum"`
	Filter           filter.FilterRules            `json:"filter"`
	CustomActions    map[engine.Category]Action    `json:"customActions,omitempty"`
	DeleteMethod     string                        `json:"deleteMethod,omitempty"` // "permanent", "trash", "versioning"
	VersionDir       string                        `json:"versionDir,omitempty"`
	ErrorMode        string                        `json:"errorMode,omitempty"`
	VerifyCopies     bool                          `json:"verifyCopies,omitempty"`
	ChangesMode      bool                          `json:"changesMode,omitempty"`  // VAR-03: use DB for change detection
	DBPath           string                        `json:"dbPath,omitempty"`
	Resume           bool                          `json:"resume,omitempty"`
	Reset            bool                          `json:"reset,omitempty"`
}
