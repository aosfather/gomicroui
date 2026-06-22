package microui

import (
	"encoding/binary"
	"strconv"
	"unsafe"
)

// inHoverRoot reports whether the hover root is on the current container
// stack. Stops searching at the first container that has a head command (a
// root container).
func (ctx *Context) inHoverRoot() bool {
	for i := len(ctx.ContainerStack) - 1; i >= 0; i-- {
		if ctx.ContainerStack[i] == ctx.HoverRoot {
			return true
		}
		// Only root containers have their HeadIdx set (>= 0); stop searching
		// once we've reached the current root container.
		if ctx.ContainerStack[i].HeadIdx >= 0 {
			break
		}
	}
	return false
}

// DrawControlFrame draws the background frame for a control, picking the
// color variant based on focus/hover state.
func (ctx *Context) DrawControlFrame(id ID, rect Rect, colorID, opt int) {
	if opt&OptNoFrame != 0 {
		return
	}
	switch {
	case ctx.Focus == id:
		colorID += 2
	case ctx.Hover == id:
		colorID += 1
	}
	ctx.DrawFrame(ctx, rect, colorID)
}

// DrawControlText draws str inside rect, aligned per opt (center/right/left).
func (ctx *Context) DrawControlText(str string, rect Rect, colorID, opt int) {
	font := ctx.Style.Font
	tw := ctx.TextWidth(font, str)
	ctx.PushClipRect(rect)
	pos := Vec2{}
	pos.Y = rect.Y + (rect.H-ctx.TextHeight(font))/2
	switch {
	case opt&OptAlignCenter != 0:
		pos.X = rect.X + (rect.W-tw)/2
	case opt&OptAlignRight != 0:
		pos.X = rect.X + rect.W - tw - ctx.Style.Padding
	default:
		pos.X = rect.X + ctx.Style.Padding
	}
	ctx.DrawText(font, str, pos, ctx.Style.Colors[colorID])
	ctx.PopClipRect()
}

// MouseOver reports whether the mouse is over rect, within the current clip
// rect, and within the hover root.
func (ctx *Context) MouseOver(rect Rect) bool {
	return rectOverlapsVec2(rect, ctx.MousePos) &&
		rectOverlapsVec2(ctx.GetClipRect(), ctx.MousePos) &&
		ctx.inHoverRoot()
}

// UpdateControl updates hover/focus state for a control with the given id
// and rect.
func (ctx *Context) UpdateControl(id ID, rect Rect, opt int) {
	mouseover := ctx.MouseOver(rect)

	if ctx.Focus == id {
		ctx.UpdatedFocus = true
	}
	if opt&OptNoInteract != 0 {
		return
	}
	if mouseover && ctx.MouseDown == 0 {
		ctx.Hover = id
	}

	if ctx.Focus == id {
		if ctx.MousePressed != 0 && !mouseover {
			ctx.SetFocus(0)
		}
		if ctx.MouseDown == 0 && opt&OptHoldFocus == 0 {
			ctx.SetFocus(0)
		}
	}

	if ctx.Hover == id {
		if ctx.MousePressed != 0 {
			ctx.SetFocus(id)
		} else if !mouseover {
			ctx.Hover = 0
		}
	}
}

// Text draws word-wrapped text starting at the next layout rect.
func (ctx *Context) Text(text string) {
	width := -1
	font := ctx.Style.Font
	color := ctx.Style.Colors[ColorText]
	ctx.LayoutBeginColumn()
	ctx.LayoutRow(1, []int{width}, ctx.TextHeight(font))
	p := 0
	for {
		r := ctx.LayoutNext()
		w := 0
		startIdx := p
		endIdx := p
		for {
			wordStart := p
			for p < len(text) && text[p] != ' ' && text[p] != '\n' {
				p++
			}
			w += ctx.TextWidth(font, text[wordStart:p])
			if w > r.W && endIdx != startIdx {
				break
			}
			if p < len(text) {
				w += ctx.TextWidth(font, text[p:p+1])
			}
			endIdx = p
			p++
			if endIdx >= len(text) || text[endIdx] == '\n' {
				break
			}
		}
		ctx.DrawText(font, text[startIdx:endIdx], NewVec2(r.X, r.Y), color)
		p = endIdx + 1
		if endIdx >= len(text) {
			break
		}
	}
	ctx.LayoutEndColumn()
}

// Label draws a single line of text at the next layout rect.
func (ctx *Context) Label(text string) {
	ctx.DrawControlText(text, ctx.LayoutNext(), ColorText, 0)
}

// ButtonEx draws a button with an optional icon and option flags. Returns
// ResSubmit if the button was clicked.
func (ctx *Context) ButtonEx(label string, icon, opt int) int {
	var id ID
	if label != "" {
		id = ctx.GetIDString(label)
	} else {
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], uint64(icon))
		id = ctx.GetID(buf[:])
	}
	r := ctx.LayoutNext()
	ctx.UpdateControl(id, r, opt)
	res := 0
	if ctx.MousePressed == MouseLeft && ctx.Focus == id {
		res |= ResSubmit
	}
	ctx.DrawControlFrame(id, r, ColorButton, opt)
	if label != "" {
		ctx.DrawControlText(label, r, ColorText, opt)
	}
	if icon != 0 {
		ctx.DrawIcon(icon, r, ctx.Style.Colors[ColorText])
	}
	return res
}

// Button draws a centered text button. Returns ResSubmit if clicked.
func (ctx *Context) Button(label string) int {
	return ctx.ButtonEx(label, 0, OptAlignCenter)
}

// Checkbox draws a checkbox labelled `label` backed by the bool at state.
// Returns ResChange if the state toggled.
func (ctx *Context) Checkbox(label string, state *bool) int {
	id := ctx.GetID(ptrIDBytes(unsafe.Pointer(state)))
	r := ctx.LayoutNext()
	box := NewRect(r.X, r.Y, r.H, r.H)
	ctx.UpdateControl(id, r, 0)
	res := 0
	if ctx.MousePressed == MouseLeft && ctx.Focus == id {
		res |= ResChange
		*state = !*state
	}
	ctx.DrawControlFrame(id, box, ColorBase, 0)
	if *state {
		ctx.DrawIcon(IconCheck, box, ctx.Style.Colors[ColorText])
	}
	r = NewRect(r.X+box.W, r.Y, r.W-box.W, r.H)
	ctx.DrawControlText(label, r, ColorText, 0)
	return res
}

// TextboxRaw draws a textbox at r with the given id, editing *buf (max
// maxLen bytes). Returns ResChange if text changed, ResSubmit if Return was
// pressed.
func (ctx *Context) TextboxRaw(buf *string, maxLen int, id ID, r Rect, opt int) int {
	res := 0
	ctx.UpdateControl(id, r, opt|OptHoldFocus)

	if ctx.Focus == id {
		// Handle text input.
		s := *buf
		n := minInt(maxLen-len(s), len(ctx.inputText))
		if n > 0 {
			s += ctx.inputText[:n]
			res |= ResChange
		}
		// Handle backspace (skip UTF-8 continuation bytes).
		if ctx.KeyPressed&KeyBackspace != 0 && len(s) > 0 {
			nb := len(s) - 1
			for nb > 0 && (s[nb]&0xc0) == 0x80 {
				nb--
			}
			s = s[:nb]
			res |= ResChange
		}
		*buf = s
		// Handle return.
		if ctx.KeyPressed&KeyReturn != 0 {
			ctx.SetFocus(0)
			res |= ResSubmit
		}
	}

	// Draw.
	ctx.DrawControlFrame(id, r, ColorBase, opt)
	if ctx.Focus == id {
		color := ctx.Style.Colors[ColorText]
		font := ctx.Style.Font
		textw := ctx.TextWidth(font, *buf)
		texth := ctx.TextHeight(font)
		ofx := r.W - ctx.Style.Padding - textw - 1
		textx := r.X + minInt(ofx, ctx.Style.Padding)
		texty := r.Y + (r.H-texth)/2
		ctx.PushClipRect(r)
		ctx.DrawText(font, *buf, NewVec2(textx, texty), color)
		ctx.DrawRect(NewRect(textx+textw, texty, 1, texth), color)
		ctx.PopClipRect()
	} else {
		ctx.DrawControlText(*buf, r, ColorText, opt)
	}

	return res
}

// TextboxEx draws a textbox at the next layout rect editing *buf.
func (ctx *Context) TextboxEx(buf *string, maxLen, opt int) int {
	id := ctx.GetID(ptrIDBytes(unsafe.Pointer(buf)))
	r := ctx.LayoutNext()
	return ctx.TextboxRaw(buf, maxLen, id, r, opt)
}

// Textbox draws a textbox with default options.
func (ctx *Context) Textbox(buf *string, maxLen int) int {
	return ctx.TextboxEx(buf, maxLen, 0)
}

// numberTextbox handles the shift-click text-editing mode for slider/number.
// Returns true if the control is in text-edit mode (caller should return
// early).
func (ctx *Context) numberTextbox(value *Real, r Rect, id ID) bool {
	if ctx.MousePressed == MouseLeft && ctx.KeyDown&KeyShift != 0 && ctx.Hover == id {
		ctx.NumberEdit = id
		ctx.NumberEditBuf = strconv.FormatFloat(float64(*value), 'g', 3, 32)
	}
	if ctx.NumberEdit == id {
		tmp := ctx.NumberEditBuf
		res := ctx.TextboxRaw(&tmp, MaxFmt, id, r, 0)
		ctx.NumberEditBuf = tmp
		if res&ResSubmit != 0 || ctx.Focus != id {
			v, _ := strconv.ParseFloat(ctx.NumberEditBuf, 32)
			*value = Real(v)
			ctx.NumberEdit = 0
		} else {
			return true
		}
	}
	return false
}

// SliderEx draws a slider editing *value in [low, high]. Returns ResChange
// if value changed.
func (ctx *Context) SliderEx(value *Real, low, high, step Real, fmt string, opt int) int {
	last := *value
	v := last
	id := ctx.GetID(ptrIDBytes(unsafe.Pointer(value)))
	base := ctx.LayoutNext()
	res := 0

	// Handle text input mode.
	if ctx.numberTextbox(&v, base, id) {
		return res
	}

	// Handle normal mode.
	ctx.UpdateControl(id, base, opt)

	// Handle input.
	if ctx.Focus == id && (ctx.MouseDown|ctx.MousePressed) == MouseLeft {
		v = low + Real(ctx.MousePos.X-base.X)*(high-low)/Real(base.W)
		if step != 0 {
			v = Real(int64((v+step/2)/step)) * step
		}
	}
	// Clamp and store value.
	v = clampReal(v, low, high)
	*value = v
	if last != v {
		res |= ResChange
	}

	// Draw base.
	ctx.DrawControlFrame(id, base, ColorBase, opt)
	// Draw thumb.
	w := ctx.Style.ThumbSize
	x := int((v - low) * Real(base.W-w) / (high - low))
	thumb := NewRect(base.X+x, base.Y, w, base.H)
	ctx.DrawControlFrame(id, thumb, ColorButton, opt)
	// Draw text.
	buf := strconv.FormatFloat(float64(v), 'f', 2, 32)
	if fmt != "" {
		// Honour caller-provided format via simple %f-style parsing.
		// We support the common "%.Nf" / "%.Ng" patterns by falling back to
		// FormatFloat with the parsed precision.
		buf = formatReal(v, fmt)
	}
	ctx.DrawControlText(buf, base, ColorText, opt)

	return res
}

// Slider draws a centered slider with default format.
func (ctx *Context) Slider(value *Real, low, high Real) int {
	return ctx.SliderEx(value, low, high, 0, SliderFmt, OptAlignCenter)
}

// NumberEx draws a number editor editing *value with drag-to-adjust.
func (ctx *Context) NumberEx(value *Real, step Real, fmt string, opt int) int {
	last := *value
	id := ctx.GetID(ptrIDBytes(unsafe.Pointer(value)))
	base := ctx.LayoutNext()
	res := 0

	// Handle text input mode.
	if ctx.numberTextbox(value, base, id) {
		return res
	}

	// Handle normal mode.
	ctx.UpdateControl(id, base, opt)

	// Handle input.
	if ctx.Focus == id && ctx.MouseDown == MouseLeft {
		*value += Real(ctx.MouseDelta.X) * step
	}
	if *value != last {
		res |= ResChange
	}

	// Draw base.
	ctx.DrawControlFrame(id, base, ColorBase, opt)
	// Draw text.
	buf := strconv.FormatFloat(float64(*value), 'f', 2, 32)
	if fmt != "" {
		buf = formatReal(*value, fmt)
	}
	ctx.DrawControlText(buf, base, ColorText, opt)

	return res
}

// Number draws a centered number editor with default format.
func (ctx *Context) Number(value *Real, step Real) int {
	return ctx.NumberEx(value, step, SliderFmt, OptAlignCenter)
}

// formatReal renders v according to a printf-style format string like
// "%.2f" or "%.0f". Supports the subset used by the library.
func formatReal(v Real, fmt string) string {
	// Parse "%[.N]f" / "%[.N]g".
	if len(fmt) < 2 || fmt[0] != '%' {
		return strconv.FormatFloat(float64(v), 'f', 2, 32)
	}
	prec := -1
	verb := byte('f')
	i := 1
	if i < len(fmt) && fmt[i] == '.' {
		i++
		start := i
		for i < len(fmt) && fmt[i] >= '0' && fmt[i] <= '9' {
			i++
		}
		if i > start {
			prec = 0
			for j := start; j < i; j++ {
				prec = prec*10 + int(fmt[j]-'0')
			}
		}
	}
	if i < len(fmt) {
		verb = fmt[i]
	}
	if prec < 0 {
		prec = -1
	}
	return strconv.FormatFloat(float64(v), verb, prec, 32)
}

// header is the shared implementation of HeaderEx and BeginTreeNodeEx.
// istreenode selects the tree-node styling (hover-only frame).
func (ctx *Context) header(label string, isTreenode bool, opt int) int {
	id := ctx.GetIDString(label)
	idx := ctx.poolGet(ctx.TreeNodePool[:], id)
	width := -1
	ctx.LayoutRow(1, []int{width}, 0)

	active := idx >= 0
	expanded := active
	if opt&OptExpanded != 0 {
		expanded = !active
	}
	r := ctx.LayoutNext()
	ctx.UpdateControl(id, r, 0)

	// Handle click — toggle active.
	if ctx.MousePressed == MouseLeft && ctx.Focus == id {
		active = !active
	}

	// Update pool ref.
	if idx >= 0 {
		if active {
			ctx.poolUpdate(ctx.TreeNodePool[:], idx)
		} else {
			ctx.TreeNodePool[idx] = PoolItem{}
		}
	} else if active {
		ctx.poolInit(ctx.TreeNodePool[:], id)
	}

	// Draw.
	if isTreenode {
		if ctx.Hover == id {
			ctx.DrawFrame(ctx, r, ColorButtonHover)
		}
	} else {
		ctx.DrawControlFrame(id, r, ColorButton, 0)
	}
	icon := IconCollapsed
	if expanded {
		icon = IconExpanded
	}
	ctx.DrawIcon(icon, NewRect(r.X, r.Y, r.H, r.H), ctx.Style.Colors[ColorText])
	r.X += r.H - ctx.Style.Padding
	r.W -= r.H - ctx.Style.Padding
	ctx.DrawControlText(label, r, ColorText, 0)

	if expanded {
		return ResActive
	}
	return 0
}

// HeaderEx draws a collapsible header. Returns ResActive if expanded.
func (ctx *Context) HeaderEx(label string, opt int) int {
	return ctx.header(label, false, opt)
}

// Header draws a header with default options.
func (ctx *Context) Header(label string) int {
	return ctx.HeaderEx(label, 0)
}

// BeginTreeNodeEx draws a tree node. Returns ResActive if expanded (caller
// should draw children and call EndTreeNode).
func (ctx *Context) BeginTreeNodeEx(label string, opt int) int {
	res := ctx.header(label, true, opt)
	if res&ResActive != 0 {
		ctx.getLayout().Indent += ctx.Style.Indent
		ctx.idPush(ctx.LastID)
	}
	return res
}

// BeginTreeNode draws a tree node with default options.
func (ctx *Context) BeginTreeNode(label string) int {
	return ctx.BeginTreeNodeEx(label, 0)
}

// EndTreeNode closes a tree node opened by BeginTreeNode.
func (ctx *Context) EndTreeNode() {
	ctx.getLayout().Indent -= ctx.Style.Indent
	ctx.PopID()
}
