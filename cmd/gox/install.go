package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	_ "embed"
)

//go:embed hooks/claude_stop.sh
// global-ok: required by //go:embed (compiler restriction — must be a package-level var).
var claudeStopScript []byte

// claudeHookCommand is the exact "command" string written into the hook
// entry. We use it both when registering and when detecting an existing
// install (idempotency + migration).
const claudeHookCommand = "$HOME/.claude/gox-hook.sh"

// claudeHookEvent is the hook event this hook is registered under. It was
// previously "PostToolUse" (ran on every Edit/Write — too frequent); it is
// now "Stop", which fires once per turn.
const claudeHookEvent = "Stop"

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
	if writeErr := os.WriteFile(scriptPath, claudeStopScript, 0o755); writeErr != nil {
		fmt.Fprintln(os.Stderr, "gox install: cannot write hook script:", writeErr)
		return 2
	}
	fmt.Printf("✓ wrote %s\n", scriptPath)

	settingsPath := filepath.Join(claudeDir, "settings.json")
	added, migrated, settingsErr := registerClaudeHook(settingsPath)
	if settingsErr != nil {
		fmt.Fprintln(os.Stderr, "gox install: cannot update settings.json:", settingsErr)
		return 2
	}
	switch {
	case migrated:
		fmt.Printf("✓ migrated gox hook from PostToolUse to %s in %s\n", claudeHookEvent, settingsPath)
	case added:
		fmt.Printf("✓ registered %s hook in %s\n", claudeHookEvent, settingsPath)
	default:
		fmt.Printf("✓ hook already registered in %s (script refreshed)\n", settingsPath)
	}
	fmt.Println()
	fmt.Println("note: Claude Code only re-reads settings.json when /hooks is opened or the")
	fmt.Println("      app is restarted, so the hook activates in a new session (or right after")
	fmt.Println("      opening /hooks once in the current one).")
	return 0
}

// registerClaudeHook reads settings.json (if it exists), removes any stale
// gox PostToolUse entry from an older install, inserts or refreshes the gox
// Stop hook entry, and writes it back. It returns:
//   - added:    true if a new Stop entry was appended
//   - migrated: true if a stale PostToolUse entry was removed in the process
//
// If the Stop entry was already present and no stale entry existed, both are
// false. Every other key in settings.json is preserved untouched.
func registerClaudeHook(path string) (added, migrated bool, err error) {
	settings := map[string]any{}
	data, readErr := os.ReadFile(path)
	switch {
	case readErr == nil:
		if jsonErr := json.Unmarshal(data, &settings); jsonErr != nil {
			return false, false, fmt.Errorf("parse %s: %w", path, jsonErr)
		}
	case os.IsNotExist(readErr):
		// fresh install
	default:
		return false, false, fmt.Errorf("read %s: %w", path, readErr)
	}

	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
		settings["hooks"] = hooks
	}

	// Migration: drop any PostToolUse entry that ran our hook command (from
	// gox <= 0.1.0, when this was a PostToolUse hook).
	if postArr, ok := hooks["PostToolUse"].([]any); ok {
		kept := make([]any, 0, len(postArr))
		for _, raw := range postArr {
			if entryHasGoxCommand(raw) {
				migrated = true
				continue
			}
			kept = append(kept, raw)
		}
		switch len(kept) {
		case 0:
			delete(hooks, "PostToolUse")
		default:
			hooks["PostToolUse"] = kept
		}
	}

	stopArr, _ := hooks[claudeHookEvent].([]any)

	// Look for an existing Stop entry with our exact command.
	alreadyInstalled := slices.ContainsFunc(stopArr, entryHasGoxCommand)

	if !alreadyInstalled {
		newEntry := map[string]any{
			"hooks": []any{
				map[string]any{
					"type":    "command",
					"command": claudeHookCommand,
					"timeout": claudeHookTimeout,
				},
			},
		}
		hooks[claudeHookEvent] = append(stopArr, newEntry)
		added = true
	}

	if !added && !migrated {
		// Nothing structural changed — leave the file byte-identical.
		return false, false, nil
	}

	out, marshalErr := json.MarshalIndent(settings, "", "  ")
	if marshalErr != nil {
		return false, false, fmt.Errorf("marshal settings: %w", marshalErr)
	}
	out = append(out, '\n')
	tmp := path + ".tmp"
	if wErr := os.WriteFile(tmp, out, 0o644); wErr != nil {
		return false, false, fmt.Errorf("write tmp: %w", wErr)
	}
	if rnErr := os.Rename(tmp, path); rnErr != nil {
		return false, false, fmt.Errorf("rename: %w", rnErr)
	}
	return added, migrated, nil
}

// entryHasGoxCommand reports whether a hooks-array entry (the object that
// holds an inner "hooks" list) contains a command hook running our gox
// hook command.
func entryHasGoxCommand(raw any) bool { // any-ok: settings.json hook entries are decoded as untyped JSON.
	entry, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	inner, _ := entry["hooks"].([]any)
	for _, rh := range inner {
		h, ok := rh.(map[string]any)
		if !ok {
			continue
		}
		if h["type"] == "command" && h["command"] == claudeHookCommand {
			return true
		}
	}
	return false
}
