package codex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNormalizeRolloutPath(t *testing.T) {
	t.Parallel()

	got := normalizeRolloutPath(`\\?\C:\Users\me\.codex\sessions\rollout.jsonl`)
	want := `C:\Users\me\.codex\sessions\rollout.jsonl`
	if got != want {
		t.Fatalf("normalizeRolloutPath() = %q, want %q", got, want)
	}
}

func TestUpdateSessionIndexCreatesEntry(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "session_index.jsonl")
	workspace := &Workspace{SessionIndexPath: path}
	originalUpdatedAt := time.Date(2026, 5, 7, 7, 31, 10, 123000000, time.UTC)

	changed, err := workspace.updateSessionIndex([]Thread{{
		ID:          "thread-1",
		Title:       "Recovered Session",
		UpdatedAt:   originalUpdatedAt.Unix(),
		UpdatedAtMS: originalUpdatedAt.UnixMilli(),
	}}, false)
	if err != nil {
		t.Fatalf("updateSessionIndex() error = %v", err)
	}
	if !changed {
		t.Fatalf("updateSessionIndex() changed = false, want true")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(data)
	if !strings.Contains(text, `"id":"thread-1"`) {
		t.Fatalf("session index missing thread id: %s", text)
	}
	if !strings.Contains(text, `"thread_name":"Recovered Session"`) {
		t.Fatalf("session index missing thread title: %s", text)
	}
	if !strings.Contains(text, `"updated_at":"2026-05-07T07:31:10.123Z"`) {
		t.Fatalf("session index did not preserve original updated_at: %s", text)
	}
}

func TestParseModelProvider(t *testing.T) {
	t.Parallel()

	config := `
# comment
model_provider = "custom" # current provider

[model_providers.custom]
name = "Custom"
`
	if got := parseModelProvider(config); got != "custom" {
		t.Fatalf("parseModelProvider() = %q, want custom", got)
	}
}

func TestParseModelProviderFallsBackToSingleProfile(t *testing.T) {
	t.Parallel()

	config := `
service_tier = "priority"

[profiles.m21]
model = "codex-MiniMax-M2.7"
model_provider = "minimax"
`
	if got := parseModelProvider(config); got != "minimax" {
		t.Fatalf("parseModelProvider() = %q, want minimax", got)
	}
}

func TestPatchRolloutMetadataUpdatesAllProviderFields(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	original := strings.Join([]string{
		`{"type":"session_meta","payload":{"id":"thread-1","cwd":"/old","model_provider":"old_provider"}}`,
		`{"type":"event","payload":{"items":[{"model_provider":"old_provider"},{"model_provider":"current"}]}}`,
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	originalModTime := time.Date(2026, 4, 2, 3, 4, 5, 0, time.UTC)
	if err := os.Chtimes(path, originalModTime, originalModTime); err != nil {
		t.Fatalf("Chtimes() error = %v", err)
	}

	changed, err := patchRolloutMetadata(path, "thread-1", "current", "/old", false)
	if err != nil {
		t.Fatalf("patchRolloutMetadata() error = %v", err)
	}
	if !changed {
		t.Fatalf("patchRolloutMetadata() changed = false, want true")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(data)
	if strings.Contains(text, "old_provider") {
		t.Fatalf("rollout still contains old provider: %s", text)
	}
	if got := strings.Count(text, `"model_provider":"current"`); got != 3 {
		t.Fatalf("current provider count = %d, want 3 in %s", got, text)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if !info.ModTime().Equal(originalModTime) {
		t.Fatalf("rollout mod time = %s, want %s", info.ModTime(), originalModTime)
	}
}
