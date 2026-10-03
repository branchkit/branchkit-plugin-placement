# BranchKit Placement

Put any window where you want it — a position, a screen, a desktop — without
a pointer, for [BranchKit](https://branchkit.dev). MIT licensed.

Dragging a window by its title bar and edges takes two steady hands and a
pointer to express "left half, other monitor". Placement lets you say it. It
uses each OS's own window model: no tiling engine, no window server hacks.

**Placement places, tiling tiles.** Placement does one-shot moves you ask
for. Keeping a layout going on its own is a tiling plugin's job, and a tiler
can build on Placement's events (below) the way any third-party plugin would.

## What it provides

**Actions** (`placement.*`): `snap` (halves, thirds, two-thirds, quarters,
maximize, almost maximize, center, next or previous display), `to_screen`,
`undo`, `move_to_space`, `desk_switch`, `cycle_window`, `overview`, and window
state: `minimize`, `bring_back`, `fullscreen`, `pin`, `close`.

**Events**: `placement.snapped` and `placement.moved_to_space`, emitted just
before the window moves, so a tiler can release the window from its layout
first.

**Collections**: `plugin.placement.snap_mode` and
`plugin.placement.desk_mode` — both **exclusive tag gates**.

Requires the `windows`, `input` and `display` privileges.

## The interesting part: exclusive gates

`snap_mode` is the smallest complete example of the subtlest thing in the
platform model, which is why it is worth reading even if you never want a
window manager.

An exclusive gate does two things at once. At the matcher, while the tag is
active, every command that does not require an active exclusive tag is
suppressed — so in snap mode only the directional commands are eligible.
At the recognition engine, the same derivation narrows the grammar to that
command set. One declaration, two effects, and they cannot drift apart because
they are projections of the same derivation.

The converse matters too: because `requires_tags` is a conjunction, commands
behind an exclusive gate contribute no structure to the free-context grammar.
This plugin's bare `<number>` in desk mode is why "five" alone does not become
decodable outside desk mode.

Contrast with a non-exclusive gate, which only augments — everything else stays
live. High-churn contexts must be non-exclusive.

## Reading this as an example

This is a **reference implementation, not a tutorial.** It is a real shipped
plugin, carrying real scar tissue: one comment records a panic mid-drag that
left the left mouse button physically held system-wide. Read it to see how the
platform is actually used, not to learn house style.

For idiom, read
[branchkit-plugin-helloworld-go](https://github.com/branchkit/branchkit-plugin-helloworld-go)
or scaffold with `branchkit-cli dev init`.

## Build

```bash
cd src && go build -o ../placement-plugin .
```

Install into a running BranchKit:

```bash
branchkit-cli plugin install . --build
```

## Platform documentation

```bash
branchkit-cli docs sync
grep -rl "exclusive" "$(branchkit-cli docs path)"
```
