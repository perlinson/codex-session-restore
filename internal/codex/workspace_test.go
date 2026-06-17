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
	now := time.Date(2026, 6, 17, 8, 30, 0, 0, time.UTC)

	changed, err := workspace.updateSessionIndex([]Thread{{
		ID:    "thread-1",
		Title: "Recovered Session",
	}}, now, false)
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
}
