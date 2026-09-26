package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// The high score lives in the per-user config directory
// (~/Library/Application Support on macOS, ~/.config on Linux, %AppData% on Windows).
func highScorePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "arena-strike", "highscore.txt")
}

func loadHighScore() int {
	p := highScorePath()
	if p == "" {
		return 0
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func saveHighScore(n int) {
	p := highScorePath()
	if p == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	_ = os.WriteFile(p, []byte(strconv.Itoa(n)+"\n"), 0o644)
}
