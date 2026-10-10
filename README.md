# Blockworld

A Minecraft-style survival game with a rifle, written in Go with [raylib](https://www.raylib.com/).
It runs natively on macOS, Linux and Windows from the same source and ships without asset files:
block textures and sound effects are generated at startup.

## Gameplay

- The world has no edges: it wraps around, so walking east you eventually arrive from the west with
  terrain, caves, light and hostiles continuing seamlessly across the seam.
- Worlds come in three sizes, chosen with W on the main menu for the next new world: Normal (192x192),
  Large (384x384, the default, four times the land) and Huge (576x576). All are 96 blocks tall and
  everything scales with the area: villages, dungeons, mineshafts, ravines, herds and ores.
- A world is generated every run: hills, a mountain band, beaches, a sea,
  winding caves and ore veins (coal, iron, gold, diamond), plus stone-brick ruins for cover.
- **Biomes**: plains with tall grass and flowers, oak and birch forests, deserts with cacti (they sting),
  snowy taiga with tall spruces, and red-sand outback with pale eucalyptus and dead bushes.
- **Australian wildlife**: kangaroos hop across the outback and kick when provoked, emus sprint off,
  wombats trundle about, koalas doze up in the trees, platypuses paddle in rivers and lakes, and
  crocodiles lie on warm shores and charge anyone who wanders close. A kookaburra laughs at dawn.
  Koalas and platypuses are protected: harming one costs 100 points.
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
- **Game modes** (O on the main menu for the next world, G on the pause screen to switch):
  **Survival** is the normal game. **Creative** lets you fly and build with every block, unharmed.
  **Zombie** never sees the sun: hordes of the dead come every minute or so and grow without end;
  the score is how long you last. **Battle** starts you in iron gear with a rifle, bow and bread, with
  Redfort and Bluehaven already at war with you; their warbands come fast and often, half of them
  straight for your nearest flag, and the goal is to hold every flag on the map.
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
- **Two dragons** rule the skies: a fire dragon circling the highest peak and a frost dragon over the
  snowy taiga. Come within forty blocks of a roost, or hurt one, and it hunts you: fireballs that scorch
  the ground (frost ones chill and slow you), dives that bite for heavy damage. They have 150 health;
  the sword does double damage and arrows too. A slain dragon crashes down and leaves diamonds, gold and
  ammo, and the compass panel warns when one is near.
- **Dinosaurs** of several sizes. A tyrannosaur roams every world: three blocks tall, it thuds and
  roars, ignores you until hurt, then charges and bites. Brontosaur herds live in the **swamps**, a new
  biome of mud, pools, reeds, marsh lights and drooping willows: they are twice the tyrannosaur's size,
  crane their necks up to browse the willow leaves (and sometimes shake loose an apple), and are docile
  unless attacked. Raptor packs hunt together across the outback, and tiny compys scatter underfoot.
- **The goal**: an ancient beacon tower stands far from spawn (the compass shows its bearing and
  distance, and the map marks it). Climb it and light the beacon with three diamond ore to conquer
  the world: 5000 points, fireworks and an achievement. The world carries on afterwards.
- **Wolves** roam the forests and taiga. Feed one meat or fish twice and it is yours: it follows you,
  teleports to catch up, and attacks hostiles near you. Wild wolves bite back if hurt.
- **Horses** graze the plains in brown, black and white. Feed one wheat, bread or an apple twice and it is
  yours: right click to ride. Horses are fast, gallop with Ctrl, jump a block and a half with Space, and
  Shift dismounts.
- **Boats**: five planks. Hold the boat and right click at water to launch it, right click the boat to
  board, and steer with WASD; it is quick on water and crawls on land. Break it to pick it back up.
- **Cats**: ginger, black and tabby strays live around villages and in forests and shy away from you
  unless you hold a fish. Give one a fish and it is your pet: it follows you, teleports to keep up, and
  creepers will not go near it. Right click it to make it sit and wait, or follow again; a fish heals it.
- **Fishing**: craft a rod from planks and wool, click at water to cast, and click again when the
  float dips. Mostly fish (food), sometimes seeds or an old boot.
- **War of the flags**: two of the villages are rivals, Redfort and Bluehaven, each with a flag in its
  square. By day they send warbands of soldiers marching across the map to capture control-point flags
  on the open land and to raid each other; guards defend, and soldiers fight anyone of another colour.
  Stand beside any flag for eight seconds with no enemy guards near to capture it for yourself (250
  points; a village's flag turns its villagers to your side). Capturing a faction's flag or killing its
  people makes it your enemy. Hold every flag to rule the land.
- **Your own village**: craft a Village Flag (wool, planks, gold ore) and plant it on open ground. A
  settler arrives at once, and every bed you build nearby fills with another villager over time. Right
  click your flag to rally your guards against the nearest enemy flag. The minimap and compass panel
  show every flag and who holds it.
- **Villages**: a few per world on flat grassland, each a ring of plank or cobblestone houses with doors,
  windows, beds and torches around a well. One villager lives in each house: farmers, guards and
  librarians, who wander by day, go home at night and run from the undead (guards draw their swords).
  Zombies hunt villagers as well as you. Right click a villager to talk: pick replies with the
  number keys. They greet you by name, gossip about the village (who is left, how many nights, the
  trader, the tower), tell you about the land according to their trade, and offer quests you accept
  first and hand in later: bring wheat, stone, leather, coal or gold, or slay hostiles, for ammo,
  armour, diamonds or a diamond sword. Their mood shifts with the hour, the weather, their wounds and
  your deeds. Villagers also chat to each other and to you as you pass. Killing a villager costs 200
  points and the whole village holds a grudge for a while.
- **Chests**: eight planks. Right click to open a two-column screen, click a row to move a whole stack
  between your inventory and the chest (Shift moves eight), and break the chest to spill it. When you
  die your blocks and ammo go into a chest at the spot, so nothing is lost to the grass.
- **Doors**: four planks make a door. Right click opens and closes it; closed doors stop hostiles and arrows.
- **Wandering trader**: a robed visitor turns up every couple of days and wanders off again. Right click to
  trade: gold ore for rifle ammo, iron for diamonds, wool for bread, coal for TNT, and more.
- **Abandoned mineshafts**: timbered corridors with plank floors, torches and loot crates run through the
  rock, waiting to be broken into.
- **Asteroids**: every so often a rock falls from the sky. A blinking warning gives the bearing, distance and
  a twenty-second countdown while the fireball streaks in; the impact blasts a crater and scatters
  glowing meteorite, which smelts into iron or rifle ammo. The target shows as a pulsing red ring on the
  minimap and full map (M) during the warning, and each impact site stays marked with an orange X.
- **Living trees**: damaged trees regrow their leaves, canopies slowly fill out, and oak leaves next to the
  trunk ripen into apple-bearing leaves (red dots). Right click them to pick apples.
- **Storms**: heavy rain brings lightning: a flash, a bolt, and thunder that rolls in after a delay.
- **Animals**: pigs, cows and sheep graze on the grass and flee when hurt. They drop raw meat;
  hold it and right click to eat for 30 health.
- **TNT**: craft it from sand and coal, place it, then shoot it or hit it with the sword. It falls,
  flashes for a few seconds and blows a large crater. Nearby TNT chains.
- **The deep**: the world is 96 blocks tall and the surface sits high, so below the hills lies a whole
  underground layer of darker deep stone with vast halls and wide tunnels, underground lakes, lava at the
  very bottom, glowshrooms that light the floors (and can be eaten), amethyst crystals that glow and sell
  to the trader, and the only diamonds in the world. Ravines split the surface open down into it, and
  cave spiders breed in the dark around anyone exploring down there.
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
  pitch black and nights are dim. You start with forty **torches**; craft more (coal ore + planks) and place them anywhere to light
  your mine or base. Hostiles and dropped items are lit by the cell they stand in.
- Textured blocks with ambient occlusion and smooth lighting, fog, drifting clouds, stars and a sun and moon.
  The scene is anti-aliased with an FXAA pass and finished with a soft vignette (A on the pause screen
  toggles it). Plants are crossed quads rather than boxes, faces brighten as they turn toward the sun
  through the day, broken blocks shatter into textured chips, and the held block is a real cube in hand.
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
day/night events, and saves on Ctrl+C. While online, **T** or **Enter** opens chat (Enter sends, Esc cancels; gameplay keys are ignored while
typing) and holding **P** lists players. Joins and departures appear in the chat log.

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

The game runs in high-DPI mode, so on scaled desktops (Retina, Wayland or X11 at 150-200 percent)
the whole window is used and the picture is rendered at the display's native resolution.

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

### If it runs slowly

`BLOCKWORLD_BENCH=1 ./blockworld` flies a lap over a new world and logs the average frame rate (with
`BLOCKWORLD_QUALITY=0|1|2` to test a level). Press Esc and use **V** to switch graphics quality. Low renders the scene at half the window's pixel
size, draws 80 blocks out, skips clouds and keeps distant creatures off screen; Medium is the default
on Linux and Windows; High is the default on macOS. Chunk meshes now stream in a few per frame rather
than all at once. **F3** shows an overlay with the frame rate, chunks and vertices drawn, framebuffer
size and quality, which is useful when reporting a slow machine.

### Packaging as an app with an icon

The quickest way is the packaging script, which checks the toolchain, generates the icon, runs the
tests, builds, packages for the platform it runs on and writes checksums into `dist/`:

```sh
./cx.sh                  # dist/Blockworld.app + zip on macOS, tar.gz with installer on Linux, zip on Windows
./cx.sh --version 1.2    # stamp a version into the package names and the menu
./cx.sh --run            # launch the packaged game afterwards
./cx.sh --deps --install # Linux one-shot: install the distro packages, build, package and add to the app menu
```

On Windows run it from a Git Bash or MSYS2 shell. The pieces it uses are also available as make targets:

The icon is generated from code (`go run ./cmd/mkicon` writes `icon.png`) and embedded in the binary,
so the window shows it on Linux and Windows. To make a proper application:

| Platform | Command | Result |
|---|---|---|
| macOS | `make app-macos` | `dist/Blockworld.app` with an `.icns` icon; drag it to Applications |
| Linux | `make app-linux` | installs to `~/.local/bin` with a desktop launcher and icon in your app menu |
| Windows | `make app-windows` | `blockworld.exe` with the icon embedded (needs `go install github.com/tc-hib/go-winres@latest`) |

The macOS bundle is unsigned, so the first launch needs right click, Open (or
`xattr -dr com.apple.quarantine dist/Blockworld.app`). Signing and notarising with a Developer ID
is the extra step for distributing outside your own machines.
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
