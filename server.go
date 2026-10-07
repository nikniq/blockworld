package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// runServer hosts a world without a window: no rendering, no input, no local
// player. It simulates at a fixed rate, auto-saves, and shuts down cleanly on
// Ctrl+C. Players join with -join or the menu exactly as with a windowed host.
func runServer(addr, name string, testSeconds int) {
	settings = loadSettings()
	g := &Game{Audio: &Audio{}, HighScore: loadHighScore(), CraftHover: -1, Remotes: map[uint32]*RemotePlayer{}}
	g.Headless = true
	g.Net = &Net{Name: name}
	if saveExists() && g.load() {
		fmt.Println("blockworld server: loaded the saved world")
	} else {
		g.Reset()
		fmt.Println("blockworld server: generated a new world")
	}
	g.parkHeadlessPlayer()
	if err := g.StartHost(addr); err != nil {
		fmt.Fprintln(os.Stderr, "blockworld server: cannot listen:", err)
		os.Exit(1)
	}
	fmt.Println("blockworld server:", g.Net.Status)
	fmt.Println("blockworld server: Ctrl+C saves and stops")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	const dt = 1.0 / 60
	ticker := time.NewTicker(time.Second / 60)
	defer ticker.Stop()
	start := time.Now()
	lastSave, lastLog := time.Now(), time.Now()
	for {
		select {
		case <-stop:
			g.save()
			fmt.Println("blockworld server: saved, bye")
			return
		case <-ticker.C:
		}
		g.update(dt)
		g.HostTick()
		if g.MsgT > 0 && g.Msg != g.lastLogged {
			fmt.Println("blockworld server:", g.Msg)
			g.lastLogged = g.Msg
		}
		if time.Since(lastSave) > 2*time.Minute {
			lastSave = time.Now()
			g.save()
		}
		if time.Since(lastLog) > 30*time.Second {
			lastLog = time.Now()
			fmt.Printf("blockworld server: day %d %s, %d players, %d hostiles\n", g.Sky.Day, g.Sky.TimeLabel(), g.Net.PlayerCount(), g.aliveEnemies())
		}
		if testSeconds > 0 && time.Since(start) > time.Duration(testSeconds)*time.Second {
			fmt.Printf("NETTEST server: players=%d remotes=%d day=%d\n", g.Net.PlayerCount(), len(g.Remotes), g.Sky.Day)
			return
		}
	}
}

// parkHeadlessPlayer moves the unused local player out of the world so it
// never blocks placement or attracts hostiles.
func (g *Game) parkHeadlessPlayer() {
	g.Player.Pos.Y = -100
	g.Player.HP = 0
}
