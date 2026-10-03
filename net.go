package main

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Multiplayer: one player hosts and is authoritative for the world, the clock,
// hostiles, animals, drops and lit TNT. Clients simulate their own movement,
// send actions, and render the host's snapshots. Messages are gob-encoded
// over TCP.

type NetRole int

const (
	RoleSingle NetRole = iota
	RoleHost
	RoleClient
)

const (
	snapshotInterval = 50 * time.Millisecond
	stateInterval    = 33 * time.Millisecond
)

type PlayerState struct {
	ID       uint32
	Name     string
	Pos      rl.Vector3
	Yaw      float32
	Pitch    float32
	HP       int
	Held     Item
	Sneak    bool
	Swing    float32
	BobPhase float32
	BobAmt   float32
}

type EnemyState struct {
	ID      uint32
	Kind    EnemyKind
	Pos     rl.Vector3
	Heading rl.Vector3
	HP      int
	MaxHP   int
	Alive   bool
	DeathT  float32
	Fuse    float32
	Burning bool
	Flash   float32
}

type AnimalState struct {
	ID      uint32
	Kind    AnimalKind
	Pos     rl.Vector3
	Heading rl.Vector3
	HP      int
	Alive   bool
	DeathT  float32
	Phase   float32
}

type DropState struct {
	ID    uint32
	Pos   rl.Vector3
	Block Block
	Count int
	Ammo  int
	Spin  float32
}

type Snapshot struct {
	SkyT     float32
	Day      int
	Night    int
	Rain     float32
	Raining  bool
	Hostiles int
	Players  []PlayerState
	Enemies  []EnemyState
	Animals  []AnimalState
	Drops    []DropState
	Primed   []Primed
	Arrows   []Arrow
}

type FxKind int

const (
	FxExplosion FxKind = iota
	FxDig
	FxPlace
	FxShot
	FxMessage
)

type Fx struct {
	Kind  FxKind
	Pos   rl.Vector3
	Color rl.Color
	N     int
	Text  string
	Shake float32
}

// Msg carries exactly one payload; nil fields are omitted by gob.
type Msg struct {
	Hello   *struct{ Name string }
	Welcome *struct {
		ID     uint32
		World  []byte // gzip of the voxel volume
		SkyT   float32
		Day    int
		Night  int
		Spawn  rl.Vector3
		Diffic int
	}
	State *PlayerState
	Block *struct {
		X, Y, Z int
		B       Block
	}
	Break *struct{ X, Y, Z int }
	Place *struct {
		X, Y, Z int
		B       Block
	}
	Prime    *struct{ X, Y, Z int }
	Snapshot *Snapshot
	Hit      *struct {
		Enemy    uint32
		Animal   uint32
		Dmg      int
		Headshot bool
		Knock    rl.Vector3
	}
	Pickup *struct{ ID uint32 }
	Give   *struct {
		Block Block
		Count int
		Ammo  int
	}
	Damage *struct {
		Amount  int
		Cause   string
		Armored bool
	}
	Score *struct {
		Points int
		Kill   bool
	}
	DropAll *struct {
		Pos    rl.Vector3
		Blocks [numBlocks]int
		Ammo   int
	}
	Fx    *Fx
	Leave *struct{ ID uint32 }
	Text  *struct{ Text string }
}

// peer is one connection with a locked encoder.
type peer struct {
	id   uint32
	name string
	conn net.Conn
	enc  *gob.Encoder
	mu   sync.Mutex
	dead bool
}

func (p *peer) send(m *Msg) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.dead {
		return
	}
	if err := p.enc.Encode(m); err != nil {
		p.dead = true
	}
}

type inMsg struct {
	from uint32
	msg  *Msg
}

// Net is the shared networking state of a game.
type Net struct {
	Role     NetRole
	Addr     string
	Name     string
	MyID     uint32
	inbox    chan inMsg
	mu       sync.Mutex
	peers    map[uint32]*peer
	nextID   uint32
	listener net.Listener
	server   *peer // client side: the host connection
	err      error
	lastSnap time.Time
	lastSend time.Time
	Status   string
	Snaps    int // messages received (client stats)
	Blocks   int
}

func init() {
	gob.Register(Msg{})
}

func compressWorld(blocks []Block) []byte {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	b := make([]byte, len(blocks))
	for i, v := range blocks {
		b[i] = byte(v)
	}
	_, _ = zw.Write(b)
	_ = zw.Close()
	return buf.Bytes()
}

func decompressWorld(data []byte) ([]Block, error) {
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(zr)
	if err != nil {
		return nil, err
	}
	if len(raw) != worldW*worldH*worldD {
		return nil, errors.New("world size mismatch")
	}
	blocks := make([]Block, len(raw))
	for i, v := range raw {
		blocks[i] = Block(v)
	}
	return blocks, nil
}

// lanIP returns this machine's private IPv4 address for others to join, if any.
func lanIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		if ipn, ok := a.(*net.IPNet); ok {
			ip := ipn.IP.To4()
			if ip != nil && !ip.IsLoopback() && ip.IsPrivate() {
				return ip.String()
			}
		}
	}
	return ""
}

const defaultPort = ":7777"

// ---------- host ----------

// StartHost opens the listener; the main loop calls HostTick each frame.
func (g *Game) StartHost(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	n := &Net{Role: RoleHost, Addr: ln.Addr().String(), Name: g.Net.Name, MyID: 1, inbox: make(chan inMsg, 4096), peers: map[uint32]*peer{}, nextID: 2, listener: ln}
	g.Net = n
	g.World.OnSet = func(x, y, z int, b Block) {
		n.broadcast(&Msg{Block: &struct {
			X, Y, Z int
			B       Block
		}{x, y, z, b}}, 0)
	}
	go n.acceptLoop(g)
	n.Status = "Hosting on port " + addr[strings.LastIndex(addr, ":")+1:]
	if ip := lanIP(); ip != "" {
		n.Status += "   others join: " + ip + addr[strings.LastIndex(addr, ":"):]
	}
	return nil
}

func (n *Net) acceptLoop(g *Game) {
	for {
		conn, err := n.listener.Accept()
		if err != nil {
			return
		}
		go n.serveClient(g, conn)
	}
}

func (n *Net) serveClient(g *Game, conn net.Conn) {
	dec := gob.NewDecoder(conn)
	var hello Msg
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	if err := dec.Decode(&hello); err != nil || hello.Hello == nil {
		conn.Close()
		return
	}
	_ = conn.SetReadDeadline(time.Time{})
	n.mu.Lock()
	id := n.nextID
	n.nextID++
	p := &peer{id: id, name: hello.Hello.Name, conn: conn, enc: gob.NewEncoder(conn)}
	n.peers[id] = p
	n.mu.Unlock()
	// The world snapshot is taken on the main thread via the inbox to avoid races.
	n.inbox <- inMsg{from: id, msg: &hello}
	for {
		var m Msg
		if err := dec.Decode(&m); err != nil {
			break
		}
		n.inbox <- inMsg{from: id, msg: &m}
	}
	n.mu.Lock()
	delete(n.peers, id)
	n.mu.Unlock()
	conn.Close()
	n.inbox <- inMsg{from: id, msg: &Msg{Leave: &struct{ ID uint32 }{id}}}
}

func (n *Net) broadcast(m *Msg, except uint32) {
	n.mu.Lock()
	ps := make([]*peer, 0, len(n.peers))
	for _, p := range n.peers {
		if p.id != except {
			ps = append(ps, p)
		}
	}
	n.mu.Unlock()
	for _, p := range ps {
		p.send(m)
	}
}

func (n *Net) sendTo(id uint32, m *Msg) {
	n.mu.Lock()
	p := n.peers[id]
	n.mu.Unlock()
	if p != nil {
		p.send(m)
	}
}

// PlayerCount is everyone in the world including the host.
func (n *Net) PlayerCount() int {
	if n == nil || n.Role != RoleHost {
		return 1
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.peers) + 1
}

// HostTick applies client messages and broadcasts snapshots.
func (g *Game) HostTick() {
	n := g.Net
	for {
		select {
		case im := <-n.inbox:
			g.hostHandle(im.from, im.msg)
			continue
		default:
		}
		break
	}
	if time.Since(n.lastSnap) >= snapshotInterval {
		n.lastSnap = time.Now()
		n.broadcast(&Msg{Snapshot: g.makeSnapshot()}, 0)
	}
}

func (g *Game) localState() PlayerState {
	p := g.Player
	var id uint32
	name := playerName
	if g.Net != nil {
		id, name = g.Net.MyID, g.Net.Name
	}
	return PlayerState{ID: id, Name: name, Pos: p.Pos, Yaw: p.Yaw, Pitch: p.Pitch, HP: p.HP, Held: p.Held, Sneak: p.Sneak, Swing: p.Swing, BobPhase: p.BobPhase, BobAmt: p.BobAmount}
}

func (g *Game) makeSnapshot() *Snapshot {
	s := &Snapshot{SkyT: g.Sky.T, Day: g.Sky.Day, Night: g.Night, Rain: g.Sky.Rain, Raining: g.Sky.Raining, Hostiles: g.aliveEnemies()}
	s.Players = append(s.Players, g.localState())
	for _, r := range g.Remotes {
		s.Players = append(s.Players, r.PlayerState)
	}
	for _, e := range g.Enemies {
		s.Enemies = append(s.Enemies, EnemyState{e.ID, e.Kind, e.Pos, e.Heading, e.HP, e.MaxHP, e.Alive, e.DeathT, e.Fuse, e.Burning, e.HitFlash})
	}
	for _, a := range g.Animals {
		s.Animals = append(s.Animals, AnimalState{a.ID, a.Kind, a.Pos, a.Heading, a.HP, a.Alive, a.DeathT, a.Phase})
	}
	for i := range g.Drops {
		d := &g.Drops[i]
		s.Drops = append(s.Drops, DropState{d.ID, d.Pos, d.Block, d.Count, d.Ammo, d.Spin})
	}
	s.Primed = g.Primed
	s.Arrows = g.Arrows
	return s
}

func (g *Game) hostHandle(from uint32, m *Msg) {
	n := g.Net
	w := g.World
	switch {
	case m.Hello != nil:
		g.Remotes[from] = &RemotePlayer{PlayerState: PlayerState{ID: from, Name: m.Hello.Name, Pos: g.Spawn, HP: maxHealth}}
		n.sendTo(from, &Msg{Welcome: &struct {
			ID     uint32
			World  []byte
			SkyT   float32
			Day    int
			Night  int
			Spawn  rl.Vector3
			Diffic int
		}{from, compressWorld(w.Blocks), g.Sky.T, g.Sky.Day, g.Night, g.Spawn, settings.Difficulty}})
		g.say(m.Hello.Name+" joined", 2.5)
		n.broadcast(&Msg{Fx: &Fx{Kind: FxMessage, Text: m.Hello.Name + " joined"}}, from)
	case m.Leave != nil:
		if r, ok := g.Remotes[from]; ok {
			g.say(r.Name+" left", 2.5)
			n.broadcast(&Msg{Fx: &Fx{Kind: FxMessage, Text: r.Name + " left"}}, from)
		}
		delete(g.Remotes, from)
	case m.State != nil:
		if r, ok := g.Remotes[from]; ok {
			st := *m.State
			st.ID, st.Name = from, r.Name
			r.PlayerState = st
		}
	case m.Break != nil:
		b := m.Break
		if r, ok := g.Remotes[from]; ok && w.InBounds(b.X, b.Y, b.Z) && withinReach(r.Pos, b.X, b.Y, b.Z) {
			blk := w.Get(b.X, b.Y, b.Z)
			if blk != Air && blocks[blk].MineTime >= 0 {
				g.breakBlock(b.X, b.Y, b.Z)
			}
		}
	case m.Place != nil:
		pl := m.Place
		if r, ok := g.Remotes[from]; ok && w.InBounds(pl.X, pl.Y, pl.Z) && withinReach(r.Pos, pl.X, pl.Y, pl.Z) && pl.B.Placeable() {
			t := w.Get(pl.X, pl.Y, pl.Z)
			if (t == Air || (t.Liquid() && !blocks[pl.B].Tiny)) && !(blocks[pl.B].Solid && g.blockOccupied(pl.X, pl.Y, pl.Z)) {
				w.Set(pl.X, pl.Y, pl.Z, pl.B)
			}
		}
	case m.Prime != nil:
		g.primeTNT(m.Prime.X, m.Prime.Y, m.Prime.Z, 2.5)
	case m.Hit != nil:
		h := m.Hit
		if h.Enemy != 0 {
			for _, e := range g.Enemies {
				if e.ID == h.Enemy && e.Alive {
					if h.Knock.X != 0 || h.Knock.Z != 0 {
						e.Pos, _ = w.MoveBox(e.Pos, e.Spec.Radius, e.Spec.Height, h.Knock, false)
						e.Fuse = max(0, e.Fuse-0.5)
					}
					if e.Hit(h.Dmg) {
						pts := e.Spec.Points
						if h.Headshot {
							pts += pts / 2
						}
						g.killEnemyFor(e, pts, from)
					}
				}
			}
		}
		if h.Animal != 0 {
			for _, a := range g.Animals {
				if a.ID == h.Animal && a.Alive && a.Hit(h.Dmg) {
					g.killAnimal(a)
					n.sendTo(from, &Msg{Score: &struct {
						Points int
						Kill   bool
					}{10, false}})
				}
			}
		}
	case m.Pickup != nil:
		for i := range g.Drops {
			d := &g.Drops[i]
			if d.ID == m.Pickup.ID {
				n.sendTo(from, &Msg{Give: &struct {
					Block Block
					Count int
					Ammo  int
				}{d.Block, max(1, d.Count), d.Ammo}})
				g.Drops = append(g.Drops[:i], g.Drops[i+1:]...)
				break
			}
		}
	case m.DropAll != nil:
		da := m.DropAll
		for b := Block(1); b < numBlocks; b++ {
			if da.Blocks[b] > 0 {
				g.Drops = append(g.Drops, Drop{Pos: da.Pos, Vel: rl.NewVector3(0, 3, 0), Block: b, Count: da.Blocks[b]})
			}
		}
		if da.Ammo > 0 {
			g.spawnDrop(da.Pos, 0, da.Ammo)
		}
	case m.Text != nil:
		g.say(m.Text.Text, 3)
		n.broadcast(&Msg{Fx: &Fx{Kind: FxMessage, Text: m.Text.Text}}, from)
	}
}

func withinReach(from rl.Vector3, x, y, z int) bool {
	c := rl.NewVector3(float32(x)+0.5, float32(y)+0.5, float32(z)+0.5)
	return rl.Vector3Distance(rl.Vector3Add(from, rl.NewVector3(0, eyeHeight, 0)), c) <= reachDist+1.5
}

// killEnemyFor scores a kill for a remote player.
func (g *Game) killEnemyFor(e *Enemy, pts int, who uint32) {
	g.killEnemyRaw(e)
	g.Net.sendTo(who, &Msg{Score: &struct {
		Points int
		Kill   bool
	}{pts, true}})
}

// sendFx tells clients about an effect that happened on the host.
func (g *Game) sendFx(fx Fx) {
	if g.Net != nil && g.Net.Role == RoleHost {
		g.Net.broadcast(&Msg{Fx: &fx}, 0)
	}
}

// ---------- client ----------

// Connect joins a host, waits for the world, and sets the game up as a client.
func (g *Game) Connect(addr, name string) error {
	conn, err := net.DialTimeout("tcp", addr, 8*time.Second)
	if err != nil {
		return err
	}
	n := &Net{Role: RoleClient, Addr: addr, Name: name, inbox: make(chan inMsg, 4096)}
	n.server = &peer{conn: conn, enc: gob.NewEncoder(conn)}
	n.server.send(&Msg{Hello: &struct{ Name string }{name}})
	dec := gob.NewDecoder(conn)
	_ = conn.SetReadDeadline(time.Now().Add(20 * time.Second))
	var welcome Msg
	if err := dec.Decode(&welcome); err != nil || welcome.Welcome == nil {
		conn.Close()
		return fmt.Errorf("no welcome from host: %v", err)
	}
	_ = conn.SetReadDeadline(time.Time{})
	blocks, err := decompressWorld(welcome.Welcome.World)
	if err != nil {
		conn.Close()
		return err
	}
	wl := welcome.Welcome
	g.Reset()
	g.World.Unload()
	g.World = NewWorldFromBlocks(blocks)
	g.Nav = NewNavGrid(g.World)
	g.Enemies, g.Animals, g.Drops = nil, nil, nil
	g.Spawn = wl.Spawn
	g.Player.Pos = wl.Spawn
	g.Sky.T, g.Sky.Day, g.Night = wl.SkyT, wl.Day, wl.Night
	g.WasNight = g.Sky.IsNight()
	settings.Difficulty = wl.Diffic
	n.MyID = wl.ID
	g.Net = n
	n.Status = "Connected to " + addr
	go func() {
		for {
			var m Msg
			if err := dec.Decode(&m); err != nil {
				n.inbox <- inMsg{msg: &Msg{Leave: &struct{ ID uint32 }{0}}}
				return
			}
			n.inbox <- inMsg{msg: &m}
		}
	}()
	g.say("Joined "+addr, 3)
	return nil
}

func (g *Game) sendToHost(m *Msg) {
	if g.Net != nil && g.Net.Role == RoleClient {
		g.Net.server.send(m)
	}
}

// ClientTick applies host messages and sends the local player state.
func (g *Game) ClientTick() {
	n := g.Net
	for {
		select {
		case im := <-n.inbox:
			g.clientHandle(im.msg)
			continue
		default:
		}
		break
	}
	if time.Since(n.lastSend) >= stateInterval {
		n.lastSend = time.Now()
		st := g.localState()
		g.sendToHost(&Msg{State: &st})
	}
}

func (g *Game) clientHandle(m *Msg) {
	w := g.World
	p := g.Player
	switch {
	case m.Leave != nil:
		g.Net.err = errors.New("disconnected from host")
		g.say("Disconnected from host", 4)
		g.State = StateMenu
		g.Net = nil
		if rl.IsWindowReady() {
			rl.EnableCursor()
		}
	case m.Block != nil:
		b := m.Block
		g.Net.Blocks++
		w.Set(b.X, b.Y, b.Z, b.B)
	case m.Snapshot != nil:
		g.Net.Snaps++
		g.applySnapshot(m.Snapshot)
	case m.Give != nil:
		if m.Give.Ammo > 0 {
			p.Reserve += m.Give.Ammo
		} else {
			p.Inv[m.Give.Block] += m.Give.Count
		}
		g.Audio.Play(g.Audio.Pickup, 0.5)
	case m.Damage != nil:
		p.Hurt(m.Damage.Amount, m.Damage.Cause, m.Damage.Armored)
		g.Audio.Play(g.Audio.Hurt, 0.9)
	case m.Score != nil:
		g.Score += m.Score.Points
		if m.Score.Kill {
			g.Kills++
			g.Audio.Play(g.Audio.Die, 0.7)
		}
	case m.Fx != nil:
		fx := m.Fx
		switch fx.Kind {
		case FxExplosion:
			g.burst(fx.Pos, rl.NewColor(255, 160, 40, 255), fx.N)
			g.burst(fx.Pos, rl.NewColor(90, 90, 90, 255), fx.N*3/4)
			if rl.Vector3Distance(fx.Pos, p.Pos) < 30 {
				g.Shake = max(g.Shake, fx.Shake)
				g.Audio.Play(g.Audio.Explode, 1)
			}
		case FxDig:
			g.burst(fx.Pos, fx.Color, fx.N)
			if rl.Vector3Distance(fx.Pos, p.Pos) < 20 {
				g.Audio.Play(g.Audio.Dig, 0.5)
			}
		case FxPlace:
			if rl.Vector3Distance(fx.Pos, p.Pos) < 20 {
				g.Audio.Play(g.Audio.Place, 0.5)
			}
		case FxShot:
			if rl.Vector3Distance(fx.Pos, p.Pos) < 40 {
				g.Audio.Play(g.Audio.Shoot, 0.5)
			}
		case FxMessage:
			g.say(fx.Text, 3)
		}
	}
}

// applySnapshot replaces the client's view of everything the host owns.
func (g *Game) applySnapshot(s *Snapshot) {
	g.Sky.T, g.Sky.Day, g.Night = s.SkyT, s.Day, s.Night
	g.Sky.Rain, g.Sky.Raining = s.Rain, s.Raining
	g.RemoteHostiles = s.Hostiles
	seen := map[uint32]bool{}
	for _, ps := range s.Players {
		if ps.ID == g.Net.MyID {
			continue
		}
		seen[ps.ID] = true
		r, ok := g.Remotes[ps.ID]
		if !ok {
			r = &RemotePlayer{}
			g.Remotes[ps.ID] = r
		}
		r.PlayerState = ps
	}
	for id := range g.Remotes {
		if !seen[id] {
			delete(g.Remotes, id)
		}
	}
	// Enemies: keep objects by ID so positions can be smoothed.
	byID := map[uint32]*Enemy{}
	for _, e := range g.Enemies {
		byID[e.ID] = e
	}
	g.Enemies = g.Enemies[:0]
	for _, es := range s.Enemies {
		e, ok := byID[es.ID]
		if !ok {
			e = NewEnemy(es.Pos, es.Kind, 1)
			e.ID = es.ID
		}
		e.Kind, e.Spec = es.Kind, &kinds[es.Kind]
		e.Pos = rl.Vector3Lerp(e.Pos, es.Pos, 0.6)
		if rl.Vector3Distance(e.Pos, es.Pos) > 2 {
			e.Pos = es.Pos
		}
		e.Heading, e.HP, e.MaxHP = es.Heading, es.HP, es.MaxHP
		e.Alive, e.DeathT, e.Fuse, e.Burning, e.HitFlash = es.Alive, es.DeathT, es.Fuse, es.Burning, es.Flash
		g.Enemies = append(g.Enemies, e)
	}
	aByID := map[uint32]*Animal{}
	for _, a := range g.Animals {
		aByID[a.ID] = a
	}
	g.Animals = g.Animals[:0]
	for _, as := range s.Animals {
		a, ok := aByID[as.ID]
		if !ok {
			a = NewAnimal(as.Pos, as.Kind)
			a.ID = as.ID
		}
		a.Pos = rl.Vector3Lerp(a.Pos, as.Pos, 0.6)
		if rl.Vector3Distance(a.Pos, as.Pos) > 2 {
			a.Pos = as.Pos
		}
		a.Heading, a.HP, a.Alive, a.DeathT, a.Phase = as.Heading, as.HP, as.Alive, as.DeathT, as.Phase
		g.Animals = append(g.Animals, a)
	}
	g.Drops = g.Drops[:0]
	for _, ds := range s.Drops {
		g.Drops = append(g.Drops, Drop{ID: ds.ID, Pos: ds.Pos, Block: ds.Block, Count: ds.Count, Ammo: ds.Ammo, Spin: ds.Spin, Age: 1})
	}
	g.Primed = s.Primed
	g.Arrows = s.Arrows
}

// ---------- shared ----------

// RemotePlayer is another player's last known state.
type RemotePlayer struct {
	PlayerState
}

func (r *RemotePlayer) Target() Target {
	eye := rl.Vector3Add(r.Pos, rl.NewVector3(0, eyeHeight, 0))
	box := rl.NewBoundingBox(rl.NewVector3(r.Pos.X-playerHalfW, r.Pos.Y, r.Pos.Z-playerHalfW), rl.NewVector3(r.Pos.X+playerHalfW, r.Pos.Y+playerHeight, r.Pos.Z+playerHalfW))
	return Target{ID: r.ID, Pos: r.Pos, Eye: eye, Box: box}
}

func (g *Game) localTarget() Target {
	p := g.Player
	return Target{ID: 0, Pos: p.Pos, Eye: p.Eye(), Box: p.Box()}
}

// targets lists everyone hostiles may chase.
func (g *Game) targets() []Target {
	ts := []Target{}
	if g.Player.HP > 0 {
		ts = append(ts, g.localTarget())
	}
	for _, r := range g.Remotes {
		if r.HP > 0 {
			ts = append(ts, r.Target())
		}
	}
	if len(ts) == 0 {
		ts = append(ts, g.localTarget())
	}
	return ts
}

func (g *Game) nearestTarget(pos rl.Vector3) Target {
	best := g.localTarget()
	bd := rl.Vector3Distance(pos, best.Pos)
	for _, t := range g.targets() {
		if d := rl.Vector3Distance(pos, t.Pos); d < bd {
			best, bd = t, d
		}
	}
	return best
}

func (g *Game) isClient() bool { return g.Net != nil && g.Net.Role == RoleClient }
func (g *Game) isHost() bool   { return g.Net != nil && g.Net.Role == RoleHost }

// hurtTarget applies damage to whichever player an enemy hit.
func (g *Game) hurtTarget(id uint32, amount int, cause string, armored bool) {
	if id == 0 {
		g.Player.Hurt(amount, cause, armored)
		g.Audio.Play(g.Audio.Hurt, 0.9)
		return
	}
	if g.isHost() {
		g.Net.sendTo(id, &Msg{Damage: &struct {
			Amount  int
			Cause   string
			Armored bool
		}{amount, cause, armored}})
	}
}
