// Package microui is a tiny, portable, immediate-mode UI library.
//
// This is a Go port of rxi's microui C library (https://github.com/rxi/microui).
// The library expects the user to provide input and handle the resultant
// drawing commands; it does not do any drawing itself.
package microui

import (
	"encoding/binary"
	"fmt"
	"sort"
	"unsafe"
)

// Version is the library version string.
const Version = "2.02"

// Size constants — mirror the C #defines. Stacks are pre-allocated slices
// capped at these sizes; pools are fixed-size arrays.
const (
	CommandListSize    = 256 * 1024 // max number of commands
	RootListSize       = 32
	ContainerStackSize = 32
	ClipStackSize      = 32
	IDStackSize        = 32
	LayoutStackSize    = 16
	ContainerPoolSize  = 48
	TreeNodePoolSize   = 48
	MaxWidths          = 16
	MaxFmt             = 127
)

// Real is the floating-point type used by the library.
type Real = float32

// Format strings used for number/textbox rendering.
const (
	RealFmt   = "%.3g"
	SliderFmt = "%.2f"
)

// Clip results.
const (
	ClipPart = 1
	ClipAll  = 2
)

// Command kinds.
const (
	CommandJump = iota + 1
	CommandClip
	CommandRect
	CommandText
	CommandIcon
)

// Color IDs.
const (
	ColorText = iota
	ColorBorder
	ColorWindowBg
	ColorTitleBg
	ColorTitleText
	ColorPanelBg
	ColorButton
	ColorButtonHover
	ColorButtonFocus
	ColorBase
	ColorBaseHover
	ColorBaseFocus
	ColorScrollBase
	ColorScrollThumb
	ColorMax
)

// Icon IDs.
const (
	IconClose = iota + 1
	IconCheck
	IconCollapsed
	IconExpanded
	IconMax
)

// Result flags returned by controls.
const (
	ResActive = 1 << 0
	ResSubmit = 1 << 1
	ResChange = 1 << 2
)

// Option flags.
const (
	OptAlignCenter = 1 << 0
	OptAlignRight  = 1 << 1
	OptNoInteract  = 1 << 2
	OptNoFrame     = 1 << 3
	OptNoResize    = 1 << 4
	OptNoScroll    = 1 << 5
	OptNoClose     = 1 << 6
	OptNoTitle     = 1 << 7
	OptHoldFocus   = 1 << 8
	OptAutoSize    = 1 << 9
	OptPopup       = 1 << 10
	OptClosed      = 1 << 11
	OptExpanded    = 1 << 12
)

// Mouse button flags.
const (
	MouseLeft   = 1 << 0
	MouseRight  = 1 << 1
	MouseMiddle = 1 << 2
)

// Key flags.
const (
	KeyShift     = 1 << 0
	KeyCtrl      = 1 << 1
	KeyAlt       = 1 << 2
	KeyBackspace = 1 << 3
	KeyReturn    = 1 << 4
)

// ID is a 32-bit hashed identifier.
type ID uint32

// Font is an opaque font handle passed to text-width/height callbacks. The
// library does not interpret it; the user supplies and consumes it.
type Font any

// Vec2 is a 2D integer vector.
type Vec2 struct{ X, Y int }

// Rect is an integer rectangle.
type Rect struct{ X, Y, W, H int }

// Color is an 8-bit RGBA color.
type Color struct{ R, G, B, A byte }

// PoolItem is an entry in a retained-state pool.
type PoolItem struct {
	ID         ID
	LastUpdate int
}

// Command is a single drawing command in the command list. The Kind field
// selects which fields are meaningful; a single struct holds all variants
// (the C library uses a union). Variable-length text is stored as a Go
// string — no flexible array member is needed.
type Command struct {
	Kind    int    // CommandJump/Clip/Rect/Text/Icon
	JumpDst int    // index into Context.CommandList (used by CommandJump)
	Rect    Rect   // used by Clip/Rect/Icon
	Color   Color  // used by Rect/Text/Icon
	Font    Font   // used by Text
	Pos     Vec2   // used by Text
	Text    string // used by Text
	IconID  int    // used by Icon
}

// Layout tracks the layout state for a container body.
type Layout struct {
	Body      Rect
	Next      Rect
	Position  Vec2
	Size      Vec2
	Max       Vec2
	Widths    [MaxWidths]int
	Items     int
	ItemIndex int
	NextRow   int
	NextType  int
	Indent    int
}

// Container is a retained-state window/panel.
type Container struct {
	HeadIdx     int // index of head jump command in CommandList
	TailIdx     int // index of tail jump command in CommandList
	Rect        Rect
	Body        Rect
	ContentSize Vec2
	Scroll      Vec2
	Zindex      int
	Open        bool
}

// Style holds visual metrics and colors.
type Style struct {
	Font          Font
	Size          Vec2
	Padding       int
	Spacing       int
	Indent        int
	TitleHeight   int
	ScrollbarSize int
	ThumbSize     int
	Colors        [ColorMax]Color
}

// Context is the central state of the library. One Context per UI instance.
type Context struct {
	// Callbacks — the user must set TextWidth and TextHeight before calling
	// Begin. DrawFrame is set by Init to defaultDrawFrame and may be replaced.
	TextWidth  func(font Font, text string) int
	TextHeight func(font Font) int
	DrawFrame  func(ctx *Context, rect Rect, colorID int)

	// Core state.
	Style         Style
	Hover         ID
	Focus         ID
	LastID        ID
	LastRect      Rect
	LastZindex    int
	UpdatedFocus  bool
	Frame         int
	HoverRoot     *Container
	NextHoverRoot *Container
	ScrollTarget  *Container
	NumberEditBuf string
	NumberEdit    ID

	// Stacks (slices, capacity-checked on push).
	CommandList    []Command
	RootList       []*Container
	ContainerStack []*Container
	ClipStack      []Rect
	IDStack        []ID
	LayoutStack    []Layout

	// Retained state pools (fixed-size arrays — pointers into Containers are
	// stable because Context is heap-allocated and Go's GC does not move
	// heap objects).
	ContainerPool [ContainerPoolSize]PoolItem
	Containers    [ContainerPoolSize]Container
	TreeNodePool  [TreeNodePoolSize]PoolItem

	// Input state.
	MousePos     Vec2
	LastMousePos Vec2
	MouseDelta   Vec2
	ScrollDelta  Vec2
	MouseDown    int
	MousePressed int
	KeyDown      int
	KeyPressed   int
	inputText    string
}

// Vec2 returns a new Vec2.
func NewVec2(x, y int) Vec2 { return Vec2{X: x, Y: y} }

// Rect returns a new Rect.
func NewRect(x, y, w, h int) Rect { return Rect{X: x, Y: y, W: w, H: h} }

// Color returns a new Color.
func NewColor(r, g, b, a int) Color {
	return Color{R: byte(r), G: byte(g), B: byte(b), A: byte(a)}
}

// unclippedRect is a large rect used to reset clipping.
var unclippedRect = Rect{X: 0, Y: 0, W: 0x1000000, H: 0x1000000}

// defaultStyle mirrors the C default_style.
var defaultStyle = Style{
	Font:          nil,
	Size:          Vec2{X: 68, Y: 10},
	Padding:       5,
	Spacing:       4,
	Indent:        24,
	TitleHeight:   24,
	ScrollbarSize: 12,
	ThumbSize:     8,
	Colors: [ColorMax]Color{
		{230, 230, 230, 255}, // ColorText
		{25, 25, 25, 255},    // ColorBorder
		{50, 50, 50, 255},    // ColorWindowBg
		{25, 25, 25, 255},    // ColorTitleBg
		{240, 240, 240, 255}, // ColorTitleText
		{0, 0, 0, 0},         // ColorPanelBg
		{75, 75, 75, 255},    // ColorButton
		{95, 95, 95, 255},    // ColorButtonHover
		{115, 115, 115, 255}, // ColorButtonFocus
		{30, 30, 30, 255},    // ColorBase
		{35, 35, 35, 255},    // ColorBaseHover
		{40, 40, 40, 255},    // ColorBaseFocus
		{43, 43, 43, 255},    // ColorScrollBase
		{30, 30, 30, 255},    // ColorScrollThumb
	},
}

// expect panics if cond is false. The C library abort()s on assertion
// failure; panic is the Go equivalent for programming errors.
func expect(cond bool, msg string) {
	if !cond {
		panic("microui: assertion failed: " + msg)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampInt(x, a, b int) int { return minInt(b, maxInt(a, x)) }

func clampReal(x, a, b Real) Real {
	if x < a {
		return a
	}
	if x > b {
		return b
	}
	return x
}

// expandRect grows rect by n in each direction.
func expandRect(r Rect, n int) Rect {
	return NewRect(r.X-n, r.Y-n, r.W+n*2, r.H+n*2)
}

// intersectRects returns the intersection of two rects.
func intersectRects(r1, r2 Rect) Rect {
	x1 := maxInt(r1.X, r2.X)
	y1 := maxInt(r1.Y, r2.Y)
	x2 := minInt(r1.X+r1.W, r2.X+r2.W)
	y2 := minInt(r1.Y+r1.H, r2.Y+r2.H)
	if x2 < x1 {
		x2 = x1
	}
	if y2 < y1 {
		y2 = y1
	}
	return NewRect(x1, y1, x2-x1, y2-y1)
}

// rectOverlapsVec2 reports whether p is inside r (half-open).
func rectOverlapsVec2(r Rect, p Vec2) bool {
	return p.X >= r.X && p.X < r.X+r.W && p.Y >= r.Y && p.Y < r.Y+r.H
}

// defaultDrawFrame is the default frame drawer — a filled rect plus an
// optional 1px border. Set ctx.DrawFrame to override.
func defaultDrawFrame(ctx *Context, rect Rect, colorID int) {
	ctx.DrawRect(rect, ctx.Style.Colors[colorID])
	if colorID == ColorScrollBase ||
		colorID == ColorScrollThumb ||
		colorID == ColorTitleBg {
		return
	}
	if ctx.Style.Colors[ColorBorder].A != 0 {
		ctx.DrawBox(expandRect(rect, 1), ctx.Style.Colors[ColorBorder])
	}
}

// Init prepares a context for use. TextWidth and TextHeight must be set
// afterwards before calling Begin.
func Init(ctx *Context) {
	*ctx = Context{}
	ctx.DrawFrame = defaultDrawFrame
	ctx.Style = defaultStyle
}

// Begin starts a new frame. Resets transient state.
func Begin(ctx *Context) {
	expect(ctx.TextWidth != nil && ctx.TextHeight != nil, "TextWidth and TextHeight must be set")
	ctx.CommandList = ctx.CommandList[:0]
	ctx.RootList = ctx.RootList[:0]
	ctx.ScrollTarget = nil
	ctx.HoverRoot = ctx.NextHoverRoot
	ctx.NextHoverRoot = nil
	ctx.MouseDelta.X = ctx.MousePos.X - ctx.LastMousePos.X
	ctx.MouseDelta.Y = ctx.MousePos.Y - ctx.LastMousePos.Y
	ctx.Frame++
}

// End finalises the frame: handles scroll input, focus, hover-to-front, and
// rewires jump commands so iteration honours z-index ordering.
func End(ctx *Context) {
	// Check stacks are balanced.
	expect(len(ctx.ContainerStack) == 0, "container stack not empty at End")
	expect(len(ctx.ClipStack) == 0, "clip stack not empty at End")
	expect(len(ctx.IDStack) == 0, "id stack not empty at End")
	expect(len(ctx.LayoutStack) == 0, "layout stack not empty at End")

	// Handle scroll input.
	if ctx.ScrollTarget != nil {
		ctx.ScrollTarget.Scroll.X += ctx.ScrollDelta.X
		ctx.ScrollTarget.Scroll.Y += ctx.ScrollDelta.Y
	}

	// Unset focus if focus id was not touched this frame.
	if !ctx.UpdatedFocus {
		ctx.Focus = 0
	}
	ctx.UpdatedFocus = false

	// Bring hover root to front if mouse was pressed.
	if ctx.MousePressed != 0 && ctx.NextHoverRoot != nil &&
		ctx.NextHoverRoot.Zindex < ctx.LastZindex &&
		ctx.NextHoverRoot.Zindex >= 0 {
		ctx.BringToFront(ctx.NextHoverRoot)
	}

	// Reset input state.
	ctx.KeyPressed = 0
	ctx.inputText = ""
	ctx.MousePressed = 0
	ctx.ScrollDelta = NewVec2(0, 0)
	ctx.LastMousePos = ctx.MousePos

	// Sort root containers by zindex.
	n := len(ctx.RootList)
	sort.Slice(ctx.RootList[:n], func(i, j int) bool {
		return ctx.RootList[i].Zindex < ctx.RootList[j].Zindex
	})

	// Rewire jump commands so iteration follows sorted z-order.
	for i := 0; i < n; i++ {
		cnt := ctx.RootList[i]
		if i == 0 {
			// First command in buffer is this container's head jump; make it
			// jump to the command after the head.
			expect(len(ctx.CommandList) > 0, "command list empty at End")
			ctx.CommandList[0].JumpDst = cnt.HeadIdx + 1
		} else {
			prev := ctx.RootList[i-1]
			ctx.CommandList[prev.TailIdx].JumpDst = cnt.HeadIdx + 1
		}
		if i == n-1 {
			ctx.CommandList[cnt.TailIdx].JumpDst = len(ctx.CommandList)
		}
	}
}

// SetFocus marks id as the focused control for this frame.
func (ctx *Context) SetFocus(id ID) {
	ctx.Focus = id
	ctx.UpdatedFocus = true
}

// 32-bit FNV-1a hashing.
const hashInitial uint32 = 2166136261

func hashBytes(h uint32, data []byte) uint32 {
	for _, b := range data {
		h ^= uint32(b)
		h *= 16777619
	}
	return h
}

// GetID hashes data, seeded with the top of the id stack (or the initial
// hash value if the stack is empty).
func (ctx *Context) GetID(data []byte) ID {
	var h uint32
	if n := len(ctx.IDStack); n > 0 {
		h = uint32(ctx.IDStack[n-1])
	} else {
		h = hashInitial
	}
	h = hashBytes(h, data)
	ctx.LastID = ID(h)
	return ID(h)
}

// GetIDString is a convenience wrapper for string data.
func (ctx *Context) GetIDString(s string) ID { return ctx.GetID([]byte(s)) }

// ptrIDBytes encodes a pointer value as 8 little-endian bytes for hashing.
// This is used by controls that take a *state pointer (checkbox, slider,
// number, textbox) so distinct state pointers yield distinct IDs.
func ptrIDBytes(p unsafe.Pointer) []byte {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(uintptr(p)))
	return buf[:]
}

// PushID pushes a hashed id onto the id stack.
func (ctx *Context) PushID(data []byte) {
	ctx.idPush(ctx.GetID(data))
}

// PushIDString is a convenience wrapper for string data.
func (ctx *Context) PushIDString(s string) { ctx.PushID([]byte(s)) }

// PopID pops the top of the id stack.
func (ctx *Context) PopID() { ctx.idPop() }

// PushClipRect intersects rect with the current clip rect and pushes the
// result.
func (ctx *Context) PushClipRect(rect Rect) {
	last := ctx.GetClipRect()
	ctx.clipPush(intersectRects(rect, last))
}

// PopClipRect pops the top of the clip stack.
func (ctx *Context) PopClipRect() { ctx.clipPop() }

// GetClipRect returns the current clip rect.
func (ctx *Context) GetClipRect() Rect {
	expect(len(ctx.ClipStack) > 0, "clip stack empty")
	return ctx.ClipStack[len(ctx.ClipStack)-1]
}

// CheckClip tests how r relates to the current clip rect.
func (ctx *Context) CheckClip(r Rect) int {
	cr := ctx.GetClipRect()
	if r.X > cr.X+cr.W || r.X+r.W < cr.X ||
		r.Y > cr.Y+cr.H || r.Y+r.H < cr.Y {
		return ClipAll
	}
	if r.X >= cr.X && r.X+r.W <= cr.X+cr.W &&
		r.Y >= cr.Y && r.Y+r.H <= cr.Y+cr.H {
		return 0
	}
	return ClipPart
}

// GetCurrentContainer returns the container at the top of the container stack.
func (ctx *Context) GetCurrentContainer() *Container {
	expect(len(ctx.ContainerStack) > 0, "container stack empty")
	return ctx.ContainerStack[len(ctx.ContainerStack)-1]
}

// getContainer looks up a container by id in the pool, creating one if
// necessary. Returns nil if opt has OptClosed and the container does not
// already exist (or is closed).
func (ctx *Context) getContainer(id ID, opt int) *Container {
	idx := ctx.poolGet(ctx.ContainerPool[:], id)
	if idx >= 0 {
		if ctx.Containers[idx].Open || opt&OptClosed == 0 {
			ctx.poolUpdate(ctx.ContainerPool[:], idx)
		}
		return &ctx.Containers[idx]
	}
	if opt&OptClosed != 0 {
		return nil
	}
	idx = ctx.poolInit(ctx.ContainerPool[:], id)
	cnt := &ctx.Containers[idx]
	*cnt = Container{}
	cnt.Open = true
	cnt.HeadIdx = -1
	cnt.TailIdx = -1
	ctx.BringToFront(cnt)
	return cnt
}

// GetContainer looks up a container by name.
func (ctx *Context) GetContainer(name string) *Container {
	id := ctx.GetIDString(name)
	return ctx.getContainer(id, 0)
}

// BringToFront raises cnt above all other containers in z-order.
func (ctx *Context) BringToFront(cnt *Container) {
	ctx.LastZindex++
	cnt.Zindex = ctx.LastZindex
}

// ---- internal stack helpers (capacity-checked) ----

func (ctx *Context) idPush(id ID) {
	expect(len(ctx.IDStack) < IDStackSize, "id stack overflow")
	ctx.IDStack = append(ctx.IDStack, id)
}

func (ctx *Context) idPop() {
	expect(len(ctx.IDStack) > 0, "id stack underflow")
	ctx.IDStack = ctx.IDStack[:len(ctx.IDStack)-1]
}

func (ctx *Context) clipPush(r Rect) {
	expect(len(ctx.ClipStack) < ClipStackSize, "clip stack overflow")
	ctx.ClipStack = append(ctx.ClipStack, r)
}

func (ctx *Context) clipPop() {
	expect(len(ctx.ClipStack) > 0, "clip stack underflow")
	ctx.ClipStack = ctx.ClipStack[:len(ctx.ClipStack)-1]
}

func (ctx *Context) containerPush(cnt *Container) {
	expect(len(ctx.ContainerStack) < ContainerStackSize, "container stack overflow")
	ctx.ContainerStack = append(ctx.ContainerStack, cnt)
}

func (ctx *Context) containerPop() {
	expect(len(ctx.ContainerStack) > 0, "container stack underflow")
	ctx.ContainerStack = ctx.ContainerStack[:len(ctx.ContainerStack)-1]
}

func (ctx *Context) rootPush(cnt *Container) {
	expect(len(ctx.RootList) < RootListSize, "root list overflow")
	ctx.RootList = append(ctx.RootList, cnt)
}

func (ctx *Context) layoutPush(l Layout) {
	expect(len(ctx.LayoutStack) < LayoutStackSize, "layout stack overflow")
	ctx.LayoutStack = append(ctx.LayoutStack, l)
}

func (ctx *Context) layoutPop() {
	expect(len(ctx.LayoutStack) > 0, "layout stack underflow")
	ctx.LayoutStack = ctx.LayoutStack[:len(ctx.LayoutStack)-1]
}

// Stringer for Command (useful for debugging).
func (c Command) String() string {
	switch c.Kind {
	case CommandJump:
		return fmt.Sprintf("Jump(dst=%d)", c.JumpDst)
	case CommandClip:
		return fmt.Sprintf("Clip(%v)", c.Rect)
	case CommandRect:
		return fmt.Sprintf("Rect(%v %v)", c.Rect, c.Color)
	case CommandText:
		return fmt.Sprintf("Text(%q @%v %v)", c.Text, c.Pos, c.Color)
	case CommandIcon:
		return fmt.Sprintf("Icon(%d %v %v)", c.IconID, c.Rect, c.Color)
	}
	return "Command(?)"
}
