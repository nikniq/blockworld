# Blockworld

A Minecraft-style survival game with a rifle, written in Go with [raylib](https://www.raylib.com/).
It runs natively on macOS, Linux and Windows from the same source and ships without asset files:
block textures and sound effects are generated at startup.

## Gameplay

- A 192x192x64 block world is generated every run: hills, a mountain band, beaches, a sea,
  winding caves and ore veins (coal, iron, gold, diamond), plus stone-brick ruins for cover.
- **Biomes**: plains with tall grass and flowers, oak and birch forests, deserts with cacti (they sting),
  and snowy taiga with tall spruces.
- **Day/night cycle** (six minutes). Mine, craft and build by day. At nightfall zombies, spiders,
  creepers, skeleton archers and zombie brutes rise in the dark; more arrive through the night.
  Hostiles only rise where there is no torchlight, so a well-lit base is a safe base. Undead burn at sunrise.
  Each night survived is worth a bonus; nights get bigger.
- **Dungeons**: cobblestone rooms deep underground hold a monster spawner that breeds zombies and
  skeletons while you are near, and loot crates with ammo, ore, TNT or diamonds. Smash the spawner
  for a diamond and 500 points.
- **Giants** walk on every fifth night: huge, slow, 60 health, and they smash through anything but
  bedrock to reach you. Worth 1500 points and three diamonds.
- **Hunger**: ten drumsticks next to the hearts drain as you move, sprint, jump and swim. Below a third you
  cannot sprint; empty, you starve. Eat raw meat, cooked meat (craft from meat and coal) or apples that
  drop from leaves. A full stomach heals you faster.
- **Farming**: tall grass drops wheat seeds. Plant them on grass or dirt in the light; they sprout, grow
  and ripen into wheat. Three wheat bake into bread. Ripe wheat drops seeds to replant.
- **Bow and arrows**: craft a bow from planks and wool and arrows from planks and gravel. Arrows arc
  with gravity, do double damage on headshots, and work online.
- Snow falls instead of rain over snowy ground, and hits knock you back.
- **Achievements**: fourteen goals from breaking your first block to slaying a giant, with a toast when
  earned and a list on K. They persist across worlds.
- Blocks crack progressively as you mine them, and a gentle generated music loop plays in the world
  (N on the pause screen turns it off).
- **Creative mode** (G on the pause screen): double-tap Space to fly (Space up, Shift down, Ctrl fast),
  every block is on the hotbar in unlimited supply, blocks break instantly, and nothing can hurt you.
  Hostiles ignore creative players.
- **Characters** are textured, jointed models: swinging limbs, tilting heads, attack swings, and they
  topple over when killed. Zombies shamble with their arms out, skeletons carry bows, creepers have
  their face, spiders have eight legs and eight eyes, and the animals have faces and patches.
- **Difficulty** (D on the pause screen): Peaceful has no hostiles and half damage from the world,
  Hard brings half again as many hostiles doing half again as much damage.
- **Creepers** hiss when they reach you and explode, hurting everything nearby and blasting a crater.
- **Mining and building**: hold left click to break a block, right click to place the held block.
  Broken blocks drop as items you walk over. Sand and gravel fall when unsupported.
  Stone needs a pickaxe tier; iron ore needs stone, gold and diamond ore need iron.
- **Crafting (E)**: logs to planks, torches, TNT, ladders, beds, cobblestone to stone bricks, sand to glass, ore to rifle ammo,
  and stone/iron/diamond swords and pickaxes. Better pickaxes mine faster; better swords hit harder.
- **A dinosaur** roams every world: a striped, two-legged beast three blocks tall that thuds as it walks
  and roars across the valley. Leave it alone and it ignores you. Hurt it and it charges and bites for
  serious damage until it loses interest. Slaying it is worth 800 points, eight meat and leather.
- **Animals**: pigs, cows and sheep graze on the grass and flee when hurt. They drop raw meat;
  hold it and right click to eat for 30 health.
- **TNT**: craft it from sand and coal, place it, then shoot it or hit it with the sword. It falls,
  flashes for a few seconds and blows a large crater. Nearby TNT chains.
- **Lava** pools at the bottom of the deepest caves. It glows, burns anything that touches it and destroys
  dropped items. Place blocks into it to cross.
- **Beds**: craft one from planks and wool (sheep drop wool), place it and right click it at night to sleep
  through to sunrise, unless hostiles are nearby.
- **Saplings** drop from leaves. Plant one on grass or dirt in the open and it grows into a new oak.
- **Ladders**: craft from planks and stack them up a wall; hold forward or jump to climb, sneak to hang.
- **Death and respawn**: dying shows what got you. Respawn at the world spawn, or at the last bed you
  used, with full health; your blocks and ammo drop where you fell, tools and armour are kept.
- **Armour**: leather (cows drop leather), iron and diamond armour absorb 25, 45 and 65 percent of
  melee, arrow and blast damage.
- **Weather**: rain rolls in now and then, greying the sky, thickening the fog and dimming the light.
- **Save and continue**: S on the pause screen saves; quitting to the menu or closing the window
  saves too. C on the menu continues the saved world. N on the death screen starts a new world.
- **Hotbar**: rifle, sword and pickaxe, followed by every block stack you own. Pick with 1-9 or the wheel.
- The rifle is hitscan with 12-round magazines; headshots deal triple damage. Ammo comes from crafting and
  from hostiles. Health regenerates slowly out of combat. Falls hurt; water breaks the fall and you can swim.
- Sneak to move slowly without falling off edges. Sprint for speed.
- **Lighting**: sunlight and block light propagate through the world, so caves and sealed rooms are
  pitch black and nights are dim. Craft **torches** (coal ore + planks) and place them anywhere to light
  your mine or base. Hostiles and dropped items are lit by the cell they stand in.
- Textured blocks with ambient occlusion and smooth lighting, fog, drifting clouds, stars and a sun and moon.
  Grass and leaves take on the colour of their biome (lush forest, dry desert, cool taiga), textures are
  mipmapped so distant terrain does not shimmer, and the view fades into fog about 190 blocks out.
  Torchlight is warm and flickers, sunlight turns orange at dawn and dusk and blue at night, water ripples
  and reflects more at grazing angles, leaves let light through, a gradient sky dome glows around the sun,
  creatures cast soft shadows and embers drift up from lava.
- Compass and coordinates under the minimap. F5 switches to a third-person view of your blocky self.
- Score points for kills, headshots and nights survived. Your best score is saved between runs.

| Input | Action |
|---|---|
| Mouse or arrow keys | Look |
| W A S D | Move |
| Left Ctrl | Sprint |
| Left Shift | Sneak |
| Space | Jump / swim up |
| 1-9, mouse wheel, Tab | Select hotbar item |
| Left click | Shoot (rifle), swing (sword), mine (pickaxe or block, hold) |
| Right click | Place held block |
| E | Crafting |
| R | Reload |
| Esc | Pause / resume (settings live here: sensitivity, volume, invert Y, swap mouse buttons) |
| F11 | Fullscreen |
| F5 | Third-person view |
| H | Controls help |
| M | Full map |
| T (online) | Chat |
| K | Achievements |
| P (online, hold) | Player list |
| S (paused) | Save world |
| C (menu) | Continue saved world |
| Q (paused or dead) | Back to menu |
| Enter (menu or dead) | Start / restart |

## Multiplayer

One player hosts; the host's machine owns the world, the clock, hostiles, animals and drops, and
streams them to everyone else over TCP. Joined players move, mine, build, fight, craft and eat as
usual; their actions are sent to the host, which applies them and broadcasts the result.

From the main menu press **H** to host (the screen then shows the address friends should type) or
**J** to join. Hosts announce themselves on the local network, so the join screen lists the worlds it
finds: press F1 to F9 to pick one, or type an address for a host elsewhere. Names are remembered between runs; every
window shows "You are ..." while online. The same works from the command line:

```sh
# Host (continues your saved world if there is one, otherwise a new one):
./blockworld -host :7777 -name Alice

# Join from another machine on the same network:
./blockworld -join 192.168.1.10:7777 -name Bob
```

A **dedicated server** runs without a window, for example on a Linux box or a spare laptop:

```sh
./blockworld -serve :7777
```

It loads the saved world (or generates one), auto-saves every two minutes, prints joins, chat and
day/night events, and saves on Ctrl+C. While online, **T** opens chat and holding **P** lists players.

Open port 7777 (TCP) on the host's firewall for LAN play, and UDP 7778 if you want it discovered. Over the internet, forward the port on the
host's router or use a tunnel such as Tailscale. Only the host can save and sleep through the night;
beds still set each player's own spawn point. Hostiles chase whichever player is nearest.
Other players are drawn as blocky figures with name tags.

## Requirements

- Go 1.22 or newer
- A C compiler (raylib is compiled from source through cgo)
- OpenGL 3.3 (the world shader falls back to unlit rendering without it)
- Platform libraries listed below

### macOS

```sh
xcode-select --install   # once, for clang
go build -o blockworld .
./blockworld
```

### Linux (Debian/Ubuntu)

```sh
sudo apt install build-essential libgl1-mesa-dev libx11-dev libxi-dev \
    libxcursor-dev libxrandr-dev libxinerama-dev libwayland-dev libxkbcommon-dev
go build -o blockworld .
./blockworld
```

Fedora: `sudo dnf install gcc mesa-libGL-devel libX11-devel libXi-devel libXcursor-devel libXrandr-devel libXinerama-devel wayland-devel libxkbcommon-devel`

### Windows

Install Go and a GCC toolchain such as [w64devkit](https://github.com/skeeto/w64devkit)
or MSYS2 mingw-w64, make sure `gcc` is on `PATH`, then:

```powershell
go build -ldflags="-H windowsgui" -o blockworld.exe .
.\blockworld.exe
```

A `Makefile` with `run`, `build`, `build-macos`, `build-linux` and `build-windows` targets is included.
Because raylib is built through cgo, each binary must be compiled on its own platform
(or with a matching cross toolchain such as `x86_64-w64-mingw32-gcc` and `CGO_ENABLED=1 GOOS=windows CC=x86_64-w64-mingw32-gcc`).

`go test ./...` covers generation, collision, pathfinding, meshing, tools and crafting without a window.
Setting `BLOCKWORLD_SHOTS=1` runs a short scripted session that writes sky, day, night, crafting and cave screenshots
to the working directory and exits. `BLOCKWORLD_SOAK=1` runs a 4000-frame stress session (rapid mining, building,
day/night cycling, hostiles, explosions, TNT, animals, crafting, save/load) and exits; it is meant to shake out crashes.
`BLOCKWORLD_NETTEST=host` and `BLOCKWORLD_NETTEST=client`, run as two processes, perform a scripted
multiplayer session on port 7799 and log whether the client received the world, snapshots and block updates.

## Code layout

| File | Contents |
|---|---|
| `main.go` | Game loop, states (menu/playing/paused/crafting/dead), day-night events, shooting, sword, explosions, mining and placing, falling sand, HUD, hotbar, minimap |
| `player.go` | Movement on voxels, swimming, sneaking, fall damage, regeneration, mouse look, hotbar model, tool tiers |
| `enemy.go` | Zombie, spider, creeper and brute AI (nav-field steering, step-ups, wading, fuses), damage, drawing |
| `drops.go` | Item drops: physics, pickup, textured cube rendering |
| `craft.go` | Recipes and the crafting screen |
| `animals.go` | Pigs, cows and sheep: wandering, fleeing, meat drops |
| `save.go` | Saving and loading a run (gzip + gob in the user config directory) |
| `settings.go` | Mouse sensitivity, volume, invert Y and fullscreen preferences |
| `arrows.go` | Skeleton arrows |
| `net.go` | Multiplayer: host listener, client connection, gob messages, snapshots |
| `sky.go` | Day/night cycle, sky and fog colours, sun, moon, stars and clouds |
| `world.go` | Voxel volume, terrain/cave/ore generation, sunlight and torch light propagation, chunk meshing with ambient occlusion and smooth lighting, terrain and entity shaders (fog, daylight), collision, DDA raycast, spawn points |
| `textures.go` | Procedural 16x16 block texture atlas |
| `nav.go` | Flow-field pathfinding over the ground (breadth-first from the player's column, 1-block climbs) |
| `audio.go` | Procedurally synthesised sound effects |
| `score.go` | High score persistence in the user config directory |
