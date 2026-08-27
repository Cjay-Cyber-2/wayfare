package snapshot_test

import (
	"testing"

	"github.com/Wayfare-labs/wayfare/snapshot"
)

func TestSnapshotsCoverage(t *testing.T) {
	s, err := snapshot.LoadAll("../testdata/snapshots")
	if err != nil {
		t.Fatalf("failed to load snapshots: %v", err)
	}
	if len(s) == 0 {
		t.Fatal("expected at least one snapshot fixture")
	}
}
