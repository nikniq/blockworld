package main

import (
	"compress/gzip"
	"encoding/gob"
	"errors"
	"os"
	"path/filepath"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const saveVersion = 2

// SaveData is the persisted state of a run: the whole voxel volume, the
// player, the clock and the animals. Hostiles and drops are not kept.
type SaveData struct {
	Version   int
	NumBlocks int
	Blocks    []Block
	Pos       rl.Vector3
	Yaw       float32
	Pitch     float32
	HP        int
	Ammo      int
	Reserve   int
	Inv       []int
	SwordTier int
	PickTier  int
	ArmorTier int
	Spawn     rl.Vector3
	Deaths    int
	SkyT      float32
	Day       int
	Night     int
	Score     int
	Kills     int
	Animals   []SavedAnimal
}

type SavedAnimal struct {
	Kind AnimalKind
	Pos  rl.Vector3
	HP   int
}

func savePath() string {
	if p := os.Getenv("BLOCKWORLD_SAVE"); p != "" {
		return p
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "blockworld", "world.sav")
}

func saveExists() bool {
	p := savePath()
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil
}

func deleteSave() {
	if p := savePath(); p != "" {
		_ = os.Remove(p)
	}
}

// save writes the current run to disk; returns false on failure.
func (g *Game) save() bool {
	err := g.saveErr()
	if err != nil {
		rl.TraceLog(rl.LogWarning, "SAVE: %v", err)
	}
	return err == nil
}

func (g *Game) saveErr() error {
	p := savePath()
	if p == "" {
		return errors.New("no save path")
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	pl := g.Player
	d := SaveData{
		Version: saveVersion, NumBlocks: int(numBlocks), Blocks: g.World.Blocks,
		Pos: pl.Pos, Yaw: pl.Yaw, Pitch: pl.Pitch, HP: pl.HP, Ammo: pl.Ammo, Reserve: pl.Reserve,
		Inv: pl.Inv[:], SwordTier: pl.SwordTier, PickTier: pl.PickTier, ArmorTier: pl.ArmorTier,
		Spawn: g.Spawn, Deaths: g.Deaths,
		SkyT: g.Sky.T, Day: g.Sky.Day, Night: g.Night, Score: g.Score, Kills: g.Kills,
	}
	for _, a := range g.Animals {
		if a.Alive {
			d.Animals = append(d.Animals, SavedAnimal{a.Kind, a.Pos, a.HP})
		}
	}
	tmp := p + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	zw := gzip.NewWriter(f)
	err = gob.NewEncoder(zw).Encode(&d)
	if cerr := zw.Close(); err == nil {
		err = cerr
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, p)
}

// load restores a saved run; returns false if there is none or it is unreadable.
func (g *Game) load() bool {
	p := savePath()
	if p == "" {
		return false
	}
	f, err := os.Open(p)
	if err != nil {
		return false
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return false
	}
	var d SaveData
	if err := gob.NewDecoder(zr).Decode(&d); err != nil || d.Version != saveVersion ||
		d.NumBlocks != int(numBlocks) || len(d.Blocks) != worldW*worldH*worldD {
		return false
	}
	g.Reset()
	g.World.Unload()
	g.World = NewWorldFromBlocks(d.Blocks)
	g.Nav = NewNavGrid(g.World)
	pl := g.Player
	pl.Pos, pl.Yaw, pl.Pitch = d.Pos, d.Yaw, d.Pitch
	pl.HP, pl.Ammo, pl.Reserve = d.HP, d.Ammo, d.Reserve
	copy(pl.Inv[:], d.Inv)
	pl.SwordTier, pl.PickTier, pl.ArmorTier = d.SwordTier, d.PickTier, d.ArmorTier
	g.Spawn, g.Deaths = d.Spawn, d.Deaths
	pl.EnsureHeld()
	g.Sky.T, g.Sky.Day = d.SkyT, d.Day
	g.Night, g.Score, g.Kills = d.Night, d.Score, d.Kills
	g.WasNight = g.Sky.IsNight()
	g.Animals = nil
	for _, sa := range d.Animals {
		a := NewAnimal(sa.Pos, sa.Kind)
		a.HP = sa.HP
		g.Animals = append(g.Animals, a)
	}
	g.say("World loaded", 2)
	return true
}
