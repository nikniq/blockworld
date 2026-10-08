package main

import (
	"compress/gzip"
	"encoding/gob"
	"errors"
	"os"
	"path/filepath"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const saveVersion = 12

// SaveData is the persisted state of a run: the whole voxel volume, the
// player, the clock and the animals. Hostiles and drops are not kept.
type SaveData struct {
	Version   int
	NumBlocks int
	Blocks    []Block
	Seed      int
	Width     int
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
	Won       bool
	Lost      int
	Flags     []Flag
	Villages  []Village
	WarRed    bool
	WarBlue   bool
	Mode      int
	Hordes    int
	Chests    map[ChestKey]*Chest
	Hunger    float32
	SkyT      float32
	Day       int
	Night     int
	Score     int
	Kills     int
	Animals   []SavedAnimal
}

type SavedAnimal struct {
	Kind       AnimalKind
	Pos        rl.Vector3
	HP         int
	Home       rl.Vector3
	Prof       Profession
	Name       string
	Quest      int
	QuestDone  bool
	QuestKills int
	Tamed      bool
	Accepted   bool
	Talks      int
	Coat       int
	Sitting    bool
	Faction    int
	Village    int
	Warband    bool
	Goal       int
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
		Version: saveVersion, NumBlocks: int(numBlocks), Blocks: g.World.Blocks, Seed: g.World.Seed, Width: worldW,
		Pos: pl.Pos, Yaw: pl.Yaw, Pitch: pl.Pitch, HP: pl.HP, Ammo: pl.Ammo, Reserve: pl.Reserve,
		Inv: pl.Inv[:], SwordTier: pl.SwordTier, PickTier: pl.PickTier, ArmorTier: pl.ArmorTier,
		Spawn: g.Spawn, Deaths: g.Deaths, Hunger: pl.Hunger, Won: g.Won, Lost: g.VillagersLost,
		Flags: g.Flags, Villages: g.Villages, WarRed: g.WarRed, WarBlue: g.WarBlue, Mode: settings.Mode, Hordes: g.Hordes, Chests: g.Chests,
		SkyT: g.Sky.T, Day: g.Sky.Day, Night: g.Night, Score: g.Score, Kills: g.Kills,
	}
	for _, a := range g.Animals {
		if a.Alive {
			d.Animals = append(d.Animals, SavedAnimal{a.Kind, a.Pos, a.HP, a.Home, a.Prof, a.Name, a.Quest, a.QuestDone, a.QuestKills, a.Tamed, a.QuestAccepted, a.TalkCount, a.Coat, a.Sitting, a.Faction, a.Village, a.Warband, a.Goal})
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
		d.NumBlocks != int(numBlocks) || d.Width <= 0 || len(d.Blocks) != d.Width*worldH*d.Width {
		return false
	}
	setWorldSize(d.Width)
	g.loading = true
	g.Reset()
	g.loading = false
	g.World.Unload()
	g.World = NewWorldFromBlocks(d.Blocks, d.Seed)
	g.Nav = NewNavGrid(g.World)
	pl := g.Player
	pl.Pos, pl.Yaw, pl.Pitch = d.Pos, d.Yaw, d.Pitch
	pl.HP, pl.Ammo, pl.Reserve = d.HP, d.Ammo, d.Reserve
	copy(pl.Inv[:], d.Inv)
	pl.SwordTier, pl.PickTier, pl.ArmorTier = d.SwordTier, d.PickTier, d.ArmorTier
	g.Spawn, g.Deaths, g.Won, g.VillagersLost = d.Spawn, d.Deaths, d.Won, d.Lost
	g.Flags, g.Villages, g.WarRed, g.WarBlue = d.Flags, d.Villages, d.WarRed, d.WarBlue
	settings.Mode = d.Mode
	settings.applyMode()
	g.Hordes = d.Hordes
	if d.Chests != nil {
		g.Chests = d.Chests
	}
	pl.Hunger = d.Hunger
	pl.EnsureHeld()
	g.Sky.T, g.Sky.Day = d.SkyT, d.Day
	g.Night, g.Score, g.Kills = d.Night, d.Score, d.Kills
	g.WasNight = g.Sky.IsNight()
	g.Animals = nil
	for _, sa := range d.Animals {
		a := NewAnimal(sa.Pos, sa.Kind)
		a.HP = sa.HP
		a.Home, a.Prof, a.Name = sa.Home, sa.Prof, sa.Name
		a.Quest, a.QuestDone, a.QuestKills = sa.Quest%len(quests), sa.QuestDone, sa.QuestKills
		a.Tamed = sa.Tamed
		a.QuestAccepted, a.TalkCount = sa.Accepted, sa.Talks
		a.Coat, a.Sitting = sa.Coat, sa.Sitting
		a.Faction, a.Village, a.Warband, a.Goal = sa.Faction, sa.Village, sa.Warband, sa.Goal
		if a.Kind == AnimalVillager || a.Kind == AnimalWolf {
			a.Walking = true
		}
		g.Animals = append(g.Animals, a)
	}
	g.say("World loaded", 2)
	return true
}
