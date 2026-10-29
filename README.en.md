# GoGUI

A pure-Go, **zero-cgo** declarative GUI toolkit that draws its whole interface with OpenGL — no WebView needed for the UI itself (an optional embedded WebView2 component is available). The visual language follows modern web UI: dark surfaces, rounded corners, gradients, shadows, flex layout, and a fully customizable style DSL.

```
┌──────────────────────────────────────────────┐
│  cmd/demo / cmd/gallery     sample apps      │
├──────────────────────────────────────────────┤
│  ui                         declarative tree │
│                             style / layout   │
│                             events / paint   │
├──────────────────────────────────────────────┤
│  internal/gpu               OpenGL 3.3 batch  │
│  internal/text              opentype glyphs  │
│  internal/geom              geometry types   │
│  internal/win32             Win32/WGL bindings│
└──────────────────────────────────────────────┘
```

## Why it looks like the web without a browser

| Web feature | Implementation |
|---|---|
| CSS rounded corners / borders | SDF (signed distance field) in the fragment shader, analytically anti-aliased, any radius |
| Linear gradients | Two-color vertex interpolation (`BgGrad`) |
| `box-shadow` | SDF expansion + distance falloff approximating a Gaussian, one draw call |
| Flex layout | Two-pass measure / arrange constraint solver |
| CSS stylesheet | Go chained DSL — type safe, no parsing cost |
| DOM events | Hit testing + bubble to nearest interactive ancestor |
| Text layout | `x/image/font/opentype` coverage rasterization → GPU grayscale atlas (same approach as gio, crisp CJK) |
| HiDPI | Per-monitor DPI v2, everything laid out in logical pixels |

## Quick start

```go
package main

import "gogui/ui"

func main() {
    ui.Run(ui.Options{
        Title:      "My App",
        Width:      1180,
        Height:     760,
        FontFamily: "Microsoft YaHei UI",
        Background: "#070a12",
    }, view)
}

func view() *ui.Node {
    return ui.Column(
        ui.Text("Console").St(ui.St().Text(26, 700, "#eef2ff")),
        ui.Button("New", func() { /* click handler */ }).
            St(ui.St().Text(13, 600, "#fff").PadXY(18, 9).R(10).
                BgGrad("#6366f1", "#8b5cf6", 90).
                Shadow(18, 6, 0, "rgba(99,102,241,0.45)")),
    ).St(ui.St().Padding(28).GapOf(20))
}
```

```bash
go run ./cmd/demo
# release build without a console window:
go build -ldflags "-H=windowsgui" -o app.exe ./cmd/demo
```

Requirements: Go 1.24+ (developed on Go 1.26, windows/amd64). No CGO toolchain, no external runtime — Win32, WGL and OpenGL are resolved dynamically through `purego`.

## Ultra-sharp rendering (supersampling)

The renderer supports whole-frame **SSAA**: the frame is drawn into an offscreen target at N× the window resolution and then box-downsampled back, while glyphs are rasterized at the same N× factor. Rounded corners, strokes, shadows and text therefore get N² sub-pixel coverage per output pixel.

```go
ui.SetSuperSample(ui.SSAuto) // auto: 2× below 150% DPI, 1× above
ui.SetSuperSample(ui.SS200)  // force 2×
ui.SetSuperSample(ui.SSOff)  // 1:1, lowest GPU cost
// also: ui.SS125, ui.SS150
```

- Logical layout, hit testing and mouse coordinates are unaffected — only the render resolution changes.
- A pixel budget (8192×4320) automatically steps the factor down (`2× → 1.5× → 1.25× → 1×`) on very large windows.
- Text gamma compensation scales back with the factor so glyphs do not get over-bolded.
- Glyph atlases grow up to 2048² when the raster scale reaches 1.75× or more.

## Building

`build.ps1` pins the release flags (`-trimpath -buildvcs=false`, `-s -w -H windowsgui -buildid=`, `CGO_ENABLED=0`):

```powershell
./build.ps1                 # gallery -> dist\gallery.exe
./build.ps1 -Target all     # gallery + demo + tools
./build.ps1 -Upx            # additionally UPX -9 (smallest)
./build.ps1 -Dev            # dev build: symbols + console log
```

Measured (gallery, go1.26 windows/amd64):

| Build | Size | Notes |
| --- | --- | --- |
| `go build` (default) | 6.45 MB | symbols, debug info, local paths |
| `build.ps1` | **4.54 MB** | −30% |
| `build.ps1 -Upx` | **1.52 MB** | −76%, self-extracting at startup |

## State and interaction

The component tree is a **pure function rebuilt every frame**. Interaction state lives in your own package-level variables; call `ui.RequestRedraw()` after mutating them (thread-safe):

```go
var current = "dashboard"

ui.Button("Components", func() {
    current = "components"
    ui.RequestRedraw() // next frame: rebuild -> layout -> paint
}).St(ui.St().Text(13, 500, "#8b93ad").PadXY(14, 10).R(10))
```

- Click handlers fire on the **next** frame against the latest tree, so stale node pointers are impossible and `Key` is optional for clicks.
- `hover` / `pressed` visuals are applied automatically for any node with `OnClick != nil`.
- Scroll offsets are stored per node `Key`; give scroll containers a unique `.K("name")`.

## Components

Beyond primitives (`Box/Row/Column/Center/Spacer/Text/Button/Scroll/Stack`), `ui/widgets.go` ships ready-made controls (state held by the caller, call `ui.RequestRedraw()` in callbacks):

| Group | Components |
|---|---|
| Buttons | `Btn` (5 variants), `IconButton`, `Button`, `Link` |
| Forms | `Input`, `PasswordInput`, `Checkbox`, `Radio`, `Switch`, `Slider`, `Progress`, `Ring`, `SelectBox`, `Textarea`, `NumberInput`, `SearchInput`, `Chip`, `DateBox` |
| Display | `Card`, `Title`, `Divider`, `Tag`, `Dot`, `Avatar`, `Badge`, `KeyValue`, `Empty`, `Skeleton`, `Spinner` |
| Navigation | `Tabs`, `TabsBar`, `Breadcrumb`, `Accordion`, `MenuBar` |
| Data | `Table` (fixed/flexible columns, zebra stripes, row clicks), `ListItem` |
| Media | `Video`, `VideoPlayer`, `Player`, `VideoClip`, `MediaSource` (ffmpeg), `MusicPlayer` |
| Browser | `WebView` (system WebView2) + `WebViewNode` / `WebViewBar` / `WebViewStatus` |
| Overlays | `ModalPanel` / `AlertDlg` / `ConfirmDlg` / `PromptDlg` / `Drawer`, `MenuPanel`, `TooltipOf` + `TooltipLayer`, `Toast`, `Banner` |
| Markdown | `Markdown` with theme-aware syntax highlighting |

Engine capabilities behind them: `Stack` layering, absolute positioning (`AbsFill/AbsTL/AbsTR/AbsBL/AbsBR/AbsCenter/AbsBottom`), focus & keyboard (`FocusOn`, `OnCharEvt/OnKeyEvt`, `ui.SetFocus`), per-frame animation (`AnimOn` + `ui.AnimTime()`, `Tween`/`Spring`/`Keyframes`), coordinate-aware taps (`TapAt`), dragging (`DragEvt`), hover dedup (`OnHoverEvt`), `DisabledOn`, built-in scrollbars, text measuring (`ui.TextWidth`), tooltips (`ui.SetTooltip`).

## Layout

`Row` / `Column` are flex containers, defaulting to `AlignStretch`.

| Method | Meaning |
|---|---|
| `.W(v)` `.H(v)` | fixed size |
| `.Wp(p)` `.Hp(p)` | percentage of parent |
| `.MinWidth` `.MaxWidth` `.MinHeight` `.MaxHeight` | size constraints |
| `.GrowTo(n)` | distribute remaining main-axis space |
| `.Padding(v)` `.PadXY(h,v)` `.PadOnly(t,r,b,l)` | inner spacing |
| `.GapOf(v)` | gap between children |
| `.AlignTo(a)` `.JustifyTo(a)` | cross-axis alignment / main-axis distribution |
| `.Self(a)` | `align-self` override |
| `.Clipped()` | clip overflow |
| `.CenterY` | vertically center text inside its box |

Additional containers: `Grid`, `Wrap`, `Masonry`, `EqualRow`, `EqualCol`, plus responsive helpers `Pick(xs, sm, md, lg)`, `BP(...)`, `ViewportW()` / `ViewportH()`.

## Styling

```go
ui.St().
    Text(15, 600, "#e8ecff").            // size / weight / color
    BgGrad("#1e293b", "#0f172a", 90).    // linear gradient (angle)
    R(14).                               // radius (per-corner via Rx)
    Stroke(1, "#2c3550").                // border
    Shadow(24, 8, 0, "rgba(0,0,0,0.4)")  // blur / offsetY / spread / color
```

Colors accept `#rgb`, `#rrggbb`, `#rrggbbaa`, `rgb()` / `rgba()`, `0xRRGGBB` and `gpu.Paint`. Hover/pressed states are composed by the framework.

Themes: `ui.UseTheme(ui.ThemeDark | ui.ThemeLight | ui.ThemeSystem)` — every token (`ColBg`, `ColText`, `ColStroke`, `ColGlass`, …) follows the active theme, including the Markdown palette and scrollbar color.

## Rendering pipeline

1. `measure` — bottom-up sizing (text measured through opentype)
2. `arrange` — top-down placement and grow distribution
3. paint — shape/text/image vertices are batched; a batch is flushed only on kind or texture change
4. clip — `glScissor` with nested intersection; scroll containers clip automatically
5. optional resolve pass — downsample the supersampled target into the present target

## Project layout

```
gogui/
├── ui/             tree, style, layout, events, theme, widgets, dialogs, media
│   ├── app.go      window, event loop, painting
│   ├── layout.go   measure + arrange, font cache
│   ├── scale.go    UI zoom + supersampling control
│   ├── node.go     declarative nodes and constructors
│   └── style.go    style DSL
├── internal/
│   ├── gpu/        GLSL shaders, batching, textures, clip, blur, resolve
│   ├── text/       opentype rasterization, atlas, metrics, emoji
│   ├── win32/      Win32 + WGL bindings (purego), WebView2, Media Foundation
│   └── geom/       geometry primitives
├── cmd/demo/       minimal sample
├── cmd/gallery/    full component gallery (8 cases)
└── tools/          fonttest / atlasdump / layouttest
```

## Debugging

```bash
go run ./tools/layouttest      # dump the layout tree (no window)
go run ./tools/fonttest        # print glyph metrics
go run ./tools/atlasdump       # ASCII-dump the glyph atlas
```

Set `ui.Options.DumpLayout = "layout.txt"` to export the first frame's layout tree and exit. `GOGUI_MEMSTATS=1` prints heap/atlas/image statistics every 120 frames; `GOGUI_TEXT_GAMMA` overrides the glyph coverage gamma.

## Performance notes

- Glyph atlases grow on demand (256² → 1024², 2048² at ≥1.75× raster scale), max 24 font instances with LRU eviction.
- Only dirty atlas regions are re-uploaded (`glTexSubImage2D`).
- Font files are memory-mapped, so they never sit on the Go heap.
- Vertex buffers are orphaned + `glBufferSubData` instead of reallocated.
- Repaint on demand: pointer moves only trigger hit testing; minimized windows skip rendering; off-viewport scroll children are culled.
- Images decode asynchronously (4 workers); media frame ring is capped at 48 MB / 2 s.

## Known limitations

- No rich text (inline mixing, image wrapping) or vertical text yet
- No virtualized list widget (paginate manually)
- Video decoding relies on external `ffmpeg` / `ffprobe` (subprocess; high-res software decode is CPU heavy)
- The embedded WebView is HWND-hosted in default mode (A), so web pixels cannot be covered by menus; the experimental DirectComposition mode (B, `GOGUI_COMPOSE=1`) composites them in the same tree

## Platform roadmap

The platform-specific part is `internal/gpu.Backend` plus the window layer; porting does not touch application code.

| Platform | Window | Renderer |
|---|---|---|
| Windows | Win32 (done) | OpenGL 3.3 (done) |
| macOS | Cocoa `NSWindow` + `CAMetalLayer` | Metal / OpenGL 4.1 |
| Linux | X11 / Wayland | OpenGL / Vulkan |
| Android | `ANativeWindow` + EGL | GLES 3 |
| Web | `canvas` (mouse / keyboard / wheel events) | WebGL 2 + WebAssembly (done) |

## License

MIT — see [LICENSE](LICENSE).
