package agent

import (
	"os"
	"path/filepath"
	"testing"
)

// Regression (2026-08-29 co-located-roles incident):
// Both agent services (pat-agent-exec and pat-agent-data) installed on the same
// Windows box previously ran IDENTICAL IPC loops. Both scanned the same
// MetaQuotes Common\Files folders and both consumed PAT_master_data.txt via
// rename — the exec agent could win the rename and steal snapshots from the
// data agent (and both consumed PAT_ticks.txt the client EAs write).
// Symptom: dashboard flapped online/offline; Master Node "not sending data"
// even while connected; snapshot_count only advanced when the data agent
// happened to win the race.
//
// Fix under test: role-gate the consumer loops —
//   masterReadLoop (PAT_master_data.txt) → data role ONLY
//   readLoop       (PAT_ticks.txt)       → exec role ONLY
// Shared files remain shared: PAT_heartbeat.txt (both write identical content),
// PAT_signals.txt (exec-only via role gate in WriteToPipe call sites).

// writeCommonFolder builds a fake MetaQuotes Common\Files tree.
func writeCommonFolder(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestMasterReadLoopRoleGating(t *testing.T) {
	// The fake Common\Files folder is still created to document the intended
	// layout; the gate logic itself is pure.
	_ = writeCommonFolder(t, "PAT_master_data.txt")

	t.Run("data role reads master file", func(t *testing.T) {
		// data role: the loop must be ALLOWED to run (not skipped).
		if !masterReadAllowedFor("data") {
			t.Fatal("data role must consume PAT_master_data.txt")
		}
	})

	t.Run("exec role must not steal master data", func(t *testing.T) {
		if masterReadAllowedFor("exec") {
			t.Fatal("exec role must NOT consume PAT_master_data.txt — it steals from the data agent")
		}
	})
}

func TestClientTickReadLoopRoleGating(t *testing.T) {
	t.Run("exec role reads client ticks", func(t *testing.T) {
		if !clientTickReadAllowedFor("exec") {
			t.Fatal("exec role must consume PAT_ticks.txt written by client EAs")
		}
	})

	t.Run("data role must not steal client ticks", func(t *testing.T) {
		if clientTickReadAllowedFor("data") {
			t.Fatal("data role must NOT consume PAT_ticks.txt — it competes with the exec agent")
		}
	})
}