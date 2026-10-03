package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Settings are player preferences, kept in the user config directory.
type Settings struct {
	Sensitivity float32 // mouse look multiplier
	Volume      float32 // 0..1
	InvertY     bool
	Fullscreen  bool
	SwapButtons bool // right click mines and shoots, left click places and uses
	Difficulty  int  // 0 peaceful, 1 normal, 2 hard
	LastJoin    string
	Name        string // player name shown to others
}

var difficultyNames = [...]string{"Peaceful", "Normal", "Hard"}

// hostileScale and damageScale are the difficulty multipliers.
func hostileScale() float32 { return [...]float32{0, 1, 1.5}[settings.Difficulty] }
func damageScale() float32  { return [...]float32{0.5, 1, 1.5}[settings.Difficulty] }

func defaultSettings() Settings { return Settings{Sensitivity: 1, Volume: 0.8, Difficulty: 1} }

func settingsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "blockworld", "settings.txt")
}

func loadSettings() Settings {
	s := defaultSettings()
	p := settingsPath()
	if p == "" {
		return s
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return s
	}
	for _, line := range strings.Split(string(b), "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		switch k {
		case "sensitivity":
			if f, err := strconv.ParseFloat(v, 32); err == nil {
				s.Sensitivity = clamp(float32(f), 0.2, 4)
			}
		case "volume":
			if f, err := strconv.ParseFloat(v, 32); err == nil {
				s.Volume = clamp(float32(f), 0, 1)
			}
		case "invert_y":
			s.InvertY = v == "true"
		case "fullscreen":
			s.Fullscreen = v == "true"
		case "swap_buttons":
			s.SwapButtons = v == "true"
		case "last_join":
			s.LastJoin = v
		case "name":
			s.Name = v
		case "difficulty":
			if n, err := strconv.Atoi(v); err == nil && n >= 0 && n <= 2 {
				s.Difficulty = n
			}
		}
	}
	return s
}

func (s Settings) save() {
	p := settingsPath()
	if p == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	text := fmt.Sprintf("sensitivity=%.2f\nvolume=%.2f\ninvert_y=%t\nfullscreen=%t\nswap_buttons=%t\ndifficulty=%d\nlast_join=%s\nname=%s\n", s.Sensitivity, s.Volume, s.InvertY, s.Fullscreen, s.SwapButtons, s.Difficulty, s.LastJoin, s.Name)
	_ = os.WriteFile(p, []byte(text), 0o644)
}

// attackButton and useButton are the in-game mouse buttons, honouring the swap setting.
func attackButton() rl.MouseButton {
	if settings.SwapButtons {
		return rl.MouseButtonRight
	}
	return rl.MouseButtonLeft
}

func useButton() rl.MouseButton {
	if settings.SwapButtons {
		return rl.MouseButtonLeft
	}
	return rl.MouseButtonRight
}

func attackDown() bool    { return rl.IsMouseButtonDown(attackButton()) }
func attackPressed() bool { return rl.IsMouseButtonPressed(attackButton()) }
func usePressed() bool    { return rl.IsMouseButtonPressed(useButton()) }
