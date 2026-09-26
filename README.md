# Arena Strike

A wave-survival first-person shooter written in Go with [raylib](https://www.raylib.com/).
It runs natively on macOS, Linux and Windows from the same source.

## Gameplay

- A 64x64 block world is generated every run: rolling hills, sand lowlands, trees and brick ruins.
- Survive endless waves of hostiles. Enemies walk the terrain, climb one block at a time and route around obstacles.
- Three enemy types: **Grunts** (baseline), **Runners** (fast, frail, from wave 3) and **Brutes** (slow, tanky, 30 damage, from wave 5).
- Two tools on the hotbar. The **rifle** is a hitscan weapon with 12-round magazines; headshots deal triple damage and 1.5x points.
  The **builder** mines any block by holding left click and places blocks with right click, so you can dig in, wall off or build a tower.
- Mined blocks go to your inventory. You start with 24 planks. Bedrock cannot be broken.
- Green cubes restore health, yellow cubes give ammo. Beacons and the minimap mark them.
- Score points for kills, headshots and cleared waves. Your best score is saved between runs.
- Procedural sound effects: no asset files needed.

| Input | Action |
|---|---|
| Mouse | Look |
| W A S D | Move |
| Left Shift | Sprint |
| Space | Jump (just over one block) |
| 1 / 2 | Rifle / builder |
| Left click | Fire (rifle) or mine (builder, hold) |
| Right click | Place block (builder) |
| Mouse wheel / Tab | Cycle block to place |
| R | Reload |
| Esc | Pause / resume |
| Q (paused or dead) | Back to menu |
| Enter (menu or dead) | Start / restart |

## Requirements

- Go 1.22 or newer
- A C compiler (raylib is compiled from source through cgo)
- Platform libraries listed below

### macOS

```sh
xcode-select --install   # once, for clang
go build -o arena-strike .
./arena-strike
```

### Linux (Debian/Ubuntu)

```sh
sudo apt install build-essential libgl1-mesa-dev libx11-dev libxi-dev \
    libxcursor-dev libxrandr-dev libxinerama-dev libwayland-dev libxkbcommon-dev
go build -o arena-strike .
./arena-strike
```

Fedora: `sudo dnf install gcc mesa-libGL-devel libX11-devel libXi-devel libXcursor-devel libXrandr-devel libXinerama-devel wayland-devel libxkbcommon-devel`

### Windows

Install Go and a GCC toolchain such as [w64devkit](https://github.com/skeeto/w64devkit)
or MSYS2 mingw-w64, make sure `gcc` is on `PATH`, then:

```powershell
go build -ldflags="-H windowsgui" -o arena-strike.exe .
.\arena-strike.exe
```

A `Makefile` with `run`, `build`, `build-macos`, `build-linux` and `build-windows` targets is included.
Because raylib is built through cgo, each binary must be compiled on its own platform
(or with a matching cross toolchain such as `x86_64-w64-mingw32-gcc` and `CGO_ENABLED=1 GOOS=windows CC=x86_64-w64-mingw32-gcc`).

## Code layout

| File | Contents |
|---|---|
| `main.go` | Game loop, states (menu/playing/paused/dead), waves, pickups, shooting, mining and placing, effects, HUD, hotbar, minimap |
| `player.go` | Movement on voxels, mouse look, jumping, head bob, tools, inventory, weapon timers, camera |
| `enemy.go` | Enemy types, AI (nav-field steering with step-ups, separation, melee), damage, drawing |
| `world.go` | Voxel volume, terrain generation, chunk meshing, box-vs-block collision, DDA raycast, spawn points |
| `nav.go` | Flow-field pathfinding over the terrain surface (breadth-first from the player's column, 1-block climbs) |
| `audio.go` | Procedurally synthesised sound effects |
| `score.go` | High score persistence in the user config directory |
# blockworld
# blockworld
