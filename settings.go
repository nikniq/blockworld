package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	Creative    bool   // creative mode: fly, build freely, take no damage
	Music       bool
	Antialias   bool
	WorldSize   int // footprint of newly generated worlds
	Mode        int // ModeSurvival, ModeCreative, ModeZombie, ModeBattle
	Quality     int // 0 low, 1 medium, 2 high (-1: not chosen yet)
}

var qualityNames = [...]string{"Low", "Medium", "High"}

// qualityParams are the knobs each quality level turns.
type qualityParams struct {
	FogEnd      float32 // draw distance in blocks
	RenderScale float32 // framebuffer fraction the 3D scene is rendered at
	EntityDist  float32 // creatures beyond this are not drawn
	Builds      int     // chunk meshes built per frame
	Clouds      bool
}

func quality() qualityParams {
	switch settings.Quality {
	case 0:
		return qualityParams{FogEnd: 80, RenderScale: 0.5, EntityDist: 48, Builds: 1, Clouds: false}
	case 1:
		return qualityParams{FogEnd: 120, RenderScale: 0.75, EntityDist: 80, Builds: 2, Clouds: true}
	}
	return qualityParams{FogEnd: 190, RenderScale: 1, EntityDist: 160, Builds: 4, Clouds: true}
}

const (
	ModeSurvival = iota
	ModeCreative
	ModeZombie
	ModeBattle
	numModes
)

var modeNames = [...]string{"Survival", "Creative", "Zombie", "Battle"}
var modeBlurbs = [...]string{
	"mine by day, survive the night, light the beacon",
	"fly and build freely with every block; nothing can hurt you",
	"endless night: hordes of the dead grow without end; last as long as you can",
	"Red and Blue are at war with you from the start; take every flag",
}

// applyMode keeps the derived creative flag in step with the chosen mode.
func (s *Settings) applyMode() {
	if s.Mode < 0 || s.Mode >= numModes {
		s.Mode = ModeSurvival
	}
	s.Creative = s.Mode == ModeCreative
}

var difficultyNames = [...]string{"Peaceful", "Normal", "Hard"}

// hostileScale and damageScale are the difficulty multipliers.
func hostileScale() float32 { return [...]float32{0, 1, 1.5}[settings.Difficulty] }
func damageScale() float32  { return [...]float32{0.5, 1, 1.5}[settings.Difficulty] }

func defaultSettings() Settings {
	return Settings{Sensitivity: 1, Volume: 0.8, Difficulty: 1, Music: true}
}

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
		case "creative":
			s.Creative = v == "true"
		case "mode":
			if n, err := strconv.Atoi(v); err == nil {
				s.Mode = n
			}
		case "quality":
			if n, err := strconv.Atoi(v); err == nil && n >= 0 && n <= 2 {
				s.Quality = n
			}
		case "music":
			s.Music = v == "true"
		case "antialias":
			s.Antialias = v == "true"
		case "world_size":
			if n, err := strconv.Atoi(v); err == nil && n >= 48 && n%48 == 0 {
				s.WorldSize = n
			}
		case "difficulty":
			if n, err := strconv.Atoi(v); err == nil && n >= 0 && n <= 2 {
				s.Difficulty = n
			}
		}
	}
	if s.Creative && s.Mode == ModeSurvival {
		s.Mode = ModeCreative // older settings files
	}
	s.applyMode()
	if s.Quality < 0 {
		// First run: integrated graphics are common on Linux and Windows laptops.
		s.Quality = 2
		if runtime.GOOS != "darwin" {
			s.Quality = 1
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
	text := fmt.Sprintf("sensitivity=%.2f\nvolume=%.2f\ninvert_y=%t\nfullscreen=%t\nswap_buttons=%t\ndifficulty=%d\nlast_join=%s\nname=%s\ncreative=%t\nmusic=%t\nantialias=%t\nworld_size=%d\nmode=%d\nquality=%d\n", s.Sensitivity, s.Volume, s.InvertY, s.Fullscreen, s.SwapButtons, s.Difficulty, s.LastJoin, s.Name, s.Creative, s.Music, s.Antialias, s.WorldSize, s.Mode, s.Quality)
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

var forceAttack bool // test harness: pretend the attack button is held

func attackDown() bool    { return forceAttack || rl.IsMouseButtonDown(attackButton()) }
func attackPressed() bool { return rl.IsMouseButtonPressed(attackButton()) }
func usePressed() bool    { return rl.IsMouseButtonPressed(useButton()) }
