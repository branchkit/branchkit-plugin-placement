# BranchKit Placement

Put any window where you want it — a position, a screen, a desktop — without
a pointer. A plugin for [BranchKit](https://github.com/branchkit), an
accessibility plugin platform for the desktop. MIT licensed.

Dragging a window by its title bar and edges takes two steady hands and a
pointer to express "left half, other monitor". Placement lets you say it, or
press a key for it. It uses each OS's own window model: no tiling engine, no
window server hacks.

**Placement places, tiling tiles.** Placement does one-shot moves you ask
for. Keeping a layout going on its own is a tiling plugin's job, and a tiler
can build on Placement's events (below) the way any third-party plugin would.

BranchKit is pre-launch: the app is not publicly released yet.

## What you can say

The phrases come from `commands.json`. Any of them can also be bound to a key
in Settings → Keybinds, so nothing here needs a voice. `<apps>` is an
application name (the `apps` collection, provided by the System plugin);
`<number>` is a spoken number.

**Windows**

| Say | Does |
|---|---|
| `mission` / `show windows` / `overview` | Show every window: Mission Control on macOS, Task View on Windows, the overview on Linux |
| `next window` | Bring the current app's next window to the front |
| `put it back` | Undo the last placement of the window; say it again to go further back (up to 10) |
| `blank window desk <number>` | Move the focused window to that desktop and follow it |
| `blank <apps> desk <number>` | The same, for the named app's window |
| `stash window desk <number>` | Move the focused window to that desktop and stay here |
| `stash <apps> desk <number>` | The same, for the named app's window |

**Window state**

| Say | Does |
|---|---|
| `minimize window` / `minimize <apps>` | Minimize the focused window, or the named app's front window |
| `bring back window` / `bring back <apps>` | Restore the window you last minimized, or the named app's |
| `full screen window` / `full screen <apps>` | Toggle full screen |
| `keep in front` / `stop keeping in front` | Pin the focused window above all others, or unpin it |
| `close window` | Close the focused window |

**Snap mode.** Say `snap`, then any of these. Each also takes an app name
first (`<apps> left`, `<apps> screen 2`) to place that app's window and bring
it forward. The mode stays on, so you can keep going (`left` … `third
right`), and "snap left" works in one breath.

| Say | Window goes to |
|---|---|
| `left` / `right` | Left or right half |
| `half up` / `half down` | Top or bottom half |
| `third left` / `third middle` / `third right` | A third |
| `two thirds left` / `two thirds right` | Two thirds |
| `corner top left` / `corner top right` / `corner bottom left` / `corner bottom right` | A quarter |
| `full` / `maximize` | Fill the screen |
| `almost maximize` | Almost the whole screen, with a margin |
| `center` | Centered |
| `next` / `previous` | The next or previous display |
| `screen <number>` | A numbered screen, counted left to right |
| `screen left` / `screen right` / `screen up` / `screen down` | The neighbouring screen in that direction |

**Desk mode.** Say `desk` or `desktop`, then a number to switch to that
desktop.

## What it provides to other plugins

**Actions** (`placement.*`), callable from any plugin's commands or
dispatch: `snap` (`position`: `left`, `right`, `top`, `bottom`, `maximize`,
`almost_maximize`, `center`, `left_third`, `center_third`, `right_third`,
`left_two_thirds`, `right_two_thirds`, `top_left`, `top_right`,
`bottom_left`, `bottom_right`, `next`, `prev`), `to_screen`, `undo`,
`move_to_space` (also takes an explicit `window_id`, for a plugin moving a
window it just created), `desk_switch`, `cycle_window`, `overview`,
`minimize`, `bring_back`, `fullscreen`, `pin`, `close`. Most take an optional
`app`.

**Events**: `placement.snapped` (`window_id`, `position`, `frame`) and
`placement.moved_to_space` (`window_id`, `space`, `stay`), emitted just
*before* the window moves, so a plugin managing that window can let go of it
first.

**Collections**: `plugin.placement.snap_mode` and
`plugin.placement.desk_mode`, both exclusive tag gates (below).

## Permissions

| Privilege | Why |
|---|---|
| `windows` | Read where windows are, and move, minimize, pin and close them |
| `input` | Press the OS's own shortcuts (overview, switch desktop) and, on macOS, hold a window's title bar to carry it to another desktop |
| `display` | Read the cursor position, to put the cursor back after that carry |

No network: the manifest declares no hosts, so the sandbox gives it none.

## Platform support

The plugin is portable Go; what each OS can do comes from the platform.

- **macOS** — everything. Desktops are switched with the "Switch to Desktop N"
  shortcuts (desktops 1–16). A window is carried to another desktop by holding
  its title bar while switching, which is visible; `stash` makes a visible
  round trip back.
- **Linux** — on X11 (any EWMH window manager) and sway, everything, with the
  window moved between desktops directly. On GNOME under Wayland, snaps go
  through GNOME's own tiling shortcuts, so left half, right half and maximize
  of the focused window; moving a window between desktops is refused there.
  Other Wayland compositors report window placement as unsupported.
- **Windows** — snapping, screens, window state, desktop switching and Task
  View. The platform has no way to move another app's window between
  desktops on Windows, so the `blank`/`stash … desk` commands are not offered
  there.

That last point is a pattern worth copying: `move_to_space` lists the native
operations it needs in its action's `uses`, so on an OS where one is missing
the platform withholds the command instead of offering one that fails.

## The interesting part: exclusive gates

`snap_mode` is the smallest complete example of the subtlest thing in the
platform model, which is why it is worth reading even if you never want a
window manager.

An exclusive gate does two things at once. At the matcher, while the tag is
active, every command that does not require an active exclusive tag is
suppressed, so in snap mode only the placement words are eligible. At the
speech recogniser, the same derivation narrows the grammar to that command
set. One declaration, two effects, and they cannot drift apart because they
are projections of the same derivation.

The converse matters too: because `requires_tags` is a conjunction, commands
behind an exclusive gate contribute no structure to the free-context grammar.
This plugin's bare `<number>` in desk mode is why "five" alone does not become
recognisable outside desk mode.

The rest of the mode's behaviour is declared, not coded: each in-mode command
sets the tag again (`sets_tags`) to stay in the mode, and the collection's
`lifecycle` and `clear_on_unrelated_command` end it at the input-session
boundary or when you say something else.

Contrast with a non-exclusive gate, which only augments: everything else
stays live. High-churn contexts must be non-exclusive.

## Reading this as an example

This is a **reference implementation, not a tutorial.** It is a real shipped
plugin, carrying real scar tissue: one comment in `src/spaces.go` records how
a panic mid-drag left the left mouse button held system-wide. Read it to see
how the platform is actually used, not to learn house style.

Worth copying: action handlers are typed and registered through `Handle<Action>`
functions generated from `plugin.json` into `src/actions_gen.go` by
[branchkit-gen](https://github.com/branchkit/branchkit-gen), so no action
string is spelled in the code and a handler's params cannot drift from the
manifest. Every platform call goes through the SDK's typed wrappers
(`plugin.NativeWorldModel(...)`, `plugin.NativeRaiseWindow(...)`).

For idiom, read
[branchkit-plugin-helloworld-go](https://github.com/branchkit/branchkit-plugin-helloworld-go)
or scaffold with `branchkit-cli dev init`. The teaching plugin is
[snippets](https://github.com/branchkit/branchkit-plugin-snippets).

## Build

Go 1.24, [plugin-sdk-go](https://github.com/branchkit/plugin-sdk-go).

```bash
cd src && go build -o ../placement-plugin . && go test ./...
```

Install into a running BranchKit:

```bash
branchkit-cli plugin install . --build
```

## License

MIT. See [LICENSE](LICENSE).
