package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	_ "embed"
)

//go:embed hooks/claude_posttool.sh
// global-ok: required by //go:embed (compiler restriction — must be a package-level var).
var claudePostToolScript []byte

// claudeHookCommand is the exact "command" string written into the matcher
// entry. We use it both when registering and when detecting an existing
// install (idempotency).
const claudeHookCommand = "$HOME/.claude/gox-hook.sh"

// claudeHookMatcher is the matcher pattern this hook is registered under.
const claudeHookMatcher = "Edit|Write|MultiEdit"

// claudeHookTimeout is the per-call timeout (seconds).
const claudeHookTimeout = 30

func runInstall(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "gox install: target required (currently supported: claude)")
		return 2
	}
	switch args[0] {
	case "claude":
		return installClaude()
	default:
		fmt.Fprintf(os.Stderr, "gox install: unknown target %q (supported: claude)\n", args[0])
		return 2
	}
}

func installClaude() int {
	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		fmt.Fprintln(os.Stderr, "gox install: cannot resolve home dir:", homeErr)
		return 2
	}
	claudeDir := filepath.Join(home, ".claude")
	if mkErr := os.MkdirAll(claudeDir, 0o755); mkErr != nil {
		fmt.Fprintln(os.Stderr, "gox install: cannot create ~/.claude:", mkErr)
		return 2
	}

	scriptPath := filepath.Join(claudeDir, "gox-hook.sh")
	if writeErr := os.WriteFile(scriptPath, claudePostToolScript, 0o755); writeErr != nil {
		fmt.Fprintln(os.Stderr, "gox install: cannot write hook script:", writeErr)
		return 2
	}
	fmt.Printf("✓ wrote %s\n", scriptPath)

	settingsPath := filepath.Join(claudeDir, "settings.json")
	added, settingsErr := registerClaudeHook(settingsPath)
	if settingsErr != nil {
		fmt.Fprintln(os.Stderr, "gox install: cannot update settings.json:", settingsErr)
		return 2
	}
	if added {
		fmt.Printf("✓ registered PostToolUse hook in %s\n", settingsPath)
	} else {
		fmt.Printf("✓ hook already registered in %s (script refreshed)\n", settingsPath)
	}
	fmt.Println()
	fmt.Println("note: Claude Code only re-reads settings.json when /hooks is opened or the")
	fmt.Println("      app is restarted, so the hook activates in a new session (or right after")
	fmt.Println("      opening /hooks once in the current one).")
	return 0
}

// registerClaudeHook reads settings.json (if it exists), inserts or refreshes
// the gox PostToolUse hook entry, and writes it back. Returns true if a new
// entry was added, false if an equivalent entry was already present.
func registerClaudeHook(path string) (bool, error) {
	settings := map[string]any{}
	data, readErr := os.ReadFile(path)
	switch {
	case readErr == nil:
		if jsonErr := json.Unmarshal(data, &settings); jsonErr != nil {
			return false, fmt.Errorf("parse %s: %w", path, jsonErr)
		}
	case os.IsNotExist(readErr):
		// fresh install
	default:
		return false, fmt.Errorf("read %s: %w", path, readErr)
	}

	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
		settings["hooks"] = hooks
	}
	postArr, _ := hooks["PostToolUse"].([]any)

	// Look for an existing entry with our exact matcher AND command.
	for _, raw := range postArr {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if entry["matcher"] != claudeHookMatcher {
			continue
		}
		inner, _ := entry["hooks"].([]any)
		for _, rh := range inner {
			h, ok := rh.(map[string]any)
			if !ok {
				continue
			}
			if h["type"] == "command" && h["command"] == claudeHookCommand {
				return false, nil // already installed
			}
		}
	}

	// Not present — append a new entry.
	newEntry := map[string]any{
		"matcher": claudeHookMatcher,
		"hooks": []any{
			map[string]any{
				"type":    "command",
				"command": claudeHookCommand,
				"timeout": claudeHookTimeout,
			},
		},
	}
	hooks["PostToolUse"] = append(postArr, newEntry)

	out, marshalErr := json.MarshalIndent(settings, "", "  ")
	if marshalErr != nil {
		return false, fmt.Errorf("marshal settings: %w", marshalErr)
	}
	out = append(out, '\n')
	tmp := path + ".tmp"
	if wErr := os.WriteFile(tmp, out, 0o644); wErr != nil {
		return false, fmt.Errorf("write tmp: %w", wErr)
	}
	if rnErr := os.Rename(tmp, path); rnErr != nil {
		return false, fmt.Errorf("rename: %w", rnErr)
	}
	return true, nil
}
