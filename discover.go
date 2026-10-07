package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// LAN discovery: hosts broadcast a UDP beacon once a second; the menu and
// join screen listen and list the worlds they hear about.

const (
	beaconPort   = 7778
	beaconPrefix = "BLOCKWORLD1|"
	beaconTTL    = 4 * time.Second
)

type LANGame struct {
	Addr    string // host:port to join
	Name    string
	Players int
	Day     int
	Seen    time.Time
}

// beacon broadcasts this host's presence until the stop channel closes.
func (n *Net) beacon(g *Game, stop <-chan struct{}) {
	conn, err := net.DialUDP("udp4", nil, &net.UDPAddr{IP: net.IPv4bcast, Port: beaconPort})
	if err != nil {
		return
	}
	defer conn.Close()
	port := n.Addr[strings.LastIndex(n.Addr, ":")+1:]
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
		}
		msg := fmt.Sprintf("%s%s|%s|%d|%d", beaconPrefix, port, n.Name, n.PlayerCount(), g.Sky.Day)
		_, _ = conn.Write([]byte(msg))
	}
}

// Discovery listens for beacons and keeps the games heard recently.
type Discovery struct {
	mu    sync.Mutex
	games map[string]LANGame
	conn  net.PacketConn
	Err   string
}

func startDiscovery() *Discovery {
	d := &Discovery{games: map[string]LANGame{}}
	conn, err := net.ListenPacket("udp4", fmt.Sprintf(":%d", beaconPort))
	if err != nil {
		d.Err = "LAN discovery unavailable (port busy): type the address"
		return d
	}
	d.conn = conn
	go func() {
		buf := make([]byte, 512)
		for {
			n, from, err := conn.ReadFrom(buf)
			if err != nil {
				return
			}
			s := string(buf[:n])
			if !strings.HasPrefix(s, beaconPrefix) {
				continue
			}
			f := strings.Split(strings.TrimPrefix(s, beaconPrefix), "|")
			if len(f) < 4 {
				continue
			}
			host := from.(*net.UDPAddr).IP.String()
			players, _ := strconv.Atoi(f[2])
			day, _ := strconv.Atoi(f[3])
			game := LANGame{Addr: host + ":" + f[0], Name: f[1], Players: players, Day: day, Seen: time.Now()}
			d.mu.Lock()
			d.games[game.Addr] = game
			d.mu.Unlock()
		}
	}()
	return d
}

// Games returns the worlds heard from recently, in a stable order.
func (d *Discovery) Games() []LANGame {
	if d == nil {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	var out []LANGame
	for k, g := range d.games {
		if time.Since(g.Seen) > beaconTTL {
			delete(d.games, k)
			continue
		}
		out = append(out, g)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Addr < out[j-1].Addr; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func (d *Discovery) Stop() {
	if d != nil && d.conn != nil {
		d.conn.Close()
	}
}
