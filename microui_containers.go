package microui

// scrollbarAxis adds a scrollbar to cnt along one axis. body is the
// container body rect (modified in place to make room for the scrollbar).
// cs is the content size (with padding already added). horizontal selects
// the axis.
//
// This expands the C `scrollbar` macro (microui.c:988-1023) which used
// token-level x/y/w/h substitution to share code between axes.
func (ctx *Context) scrollbarAxis(cnt *Container, body *Rect, cs Vec2, horizontal bool) {
	var maxscroll int
	var scrollVal *int
	var mouseDeltaAxis int
	var bodyAxisLen int
	var csAxisLen int
	var idStr string

	if horizontal {
		maxscroll = cs.X - body.W
		scrollVal = &cnt.Scroll.X
		mouseDeltaAxis = ctx.MouseDelta.X
		bodyAxisLen = body.W
		csAxisLen = cs.X
		idStr = "!scrollbarx"
	} else {
		maxscroll = cs.Y - body.H
		scrollVal = &cnt.Scroll.Y
		mouseDeltaAxis = ctx.MouseDelta.Y
		bodyAxisLen = body.H
		csAxisLen = cs.Y
		idStr = "!scrollbary"
	}

	if maxscroll > 0 && bodyAxisLen > 0 {
		id := ctx.GetIDString(idStr)

		// Base rect: a strip along the edge of body.
		base := *body
		if horizontal {
			base.Y = body.Y + body.H
			base.H = ctx.Style.ScrollbarSize
		} else {
			base.X = body.X + body.W
			base.W = ctx.Style.ScrollbarSize
		}
		baseAxisLen := axisLen(base, horizontal)

		// Handle input.
		ctx.UpdateControl(id, base, 0)
		if ctx.Focus == id && ctx.MouseDown == MouseLeft {
			// Scroll proportionally: dragging the full scrollbar length
			// moves the content by (content_size / body_size) * drag.
			*scrollVal += mouseDeltaAxis * csAxisLen / baseAxisLen
		}
		// Clamp scroll to limits.
		*scrollVal = clampInt(*scrollVal, 0, maxscroll)

		// Draw base and thumb.
		ctx.DrawFrame(ctx, base, ColorScrollBase)
		thumb := base
		thumbLen := maxInt(ctx.Style.ThumbSize, baseAxisLen*bodyAxisLen/csAxisLen)
		setAxisLen(&thumb, thumbLen, horizontal)
		thumbPos := axisPos(thumb, horizontal)
		thumbPos += *scrollVal * (baseAxisLen - thumbLen) / maxscroll
		setAxisPos(&thumb, thumbPos, horizontal)
		ctx.DrawFrame(ctx, thumb, ColorScrollThumb)

		// Set this as the scroll_target if the mouse is over the body.
		if ctx.MouseOver(*body) {
			ctx.ScrollTarget = cnt
		}
	} else {
		*scrollVal = 0
	}
}

// axisLen returns the length of r along the chosen axis.
func axisLen(r Rect, horizontal bool) int {
	if horizontal {
		return r.W
	}
	return r.H
}

// axisPos returns the position of r along the chosen axis.
func axisPos(r Rect, horizontal bool) int {
	if horizontal {
		return r.X
	}
	return r.Y
}

// setAxisLen sets the length of r along the chosen axis.
func setAxisLen(r *Rect, v int, horizontal bool) {
	if horizontal {
		r.W = v
	} else {
		r.H = v
	}
}

// setAxisPos sets the position of r along the chosen axis.
func setAxisPos(r *Rect, v int, horizontal bool) {
	if horizontal {
		r.X = v
	} else {
		r.Y = v
	}
}

// scrollbars adds vertical and horizontal scrollbars to cnt as needed,
// resizing body to make room.
func (ctx *Context) scrollbars(cnt *Container, body *Rect) {
	sz := ctx.Style.ScrollbarSize
	cs := cnt.ContentSize
	cs.X += ctx.Style.Padding * 2
	cs.Y += ctx.Style.Padding * 2
	ctx.PushClipRect(*body)
	// Resize body to make room for scrollbars.
	if cs.Y > cnt.Body.H {
		body.W -= sz
	}
	if cs.X > cnt.Body.W {
		body.H -= sz
	}
	// Vertical then horizontal.
	ctx.scrollbarAxis(cnt, body, cs, false)
	ctx.scrollbarAxis(cnt, body, cs, true)
	ctx.PopClipRect()
}

// pushContainerBody sets up the layout for a container's body, adding
// scrollbars if not disabled.
func (ctx *Context) pushContainerBody(cnt *Container, body Rect, opt int) {
	if opt&OptNoScroll == 0 {
		ctx.scrollbars(cnt, &body)
	}
	ctx.pushLayout(expandRect(body, -ctx.Style.Padding), cnt.Scroll)
	cnt.Body = body
}

// beginRootContainer pushes cnt onto the container and root stacks, adds a
// head jump command, and sets up clipping.
func (ctx *Context) beginRootContainer(cnt *Container) {
	ctx.containerPush(cnt)
	ctx.rootPush(cnt)
	// Push head jump command; store its index in cnt.HeadIdx.
	headIdx := len(ctx.CommandList)
	ctx.pushJump(-1)
	cnt.HeadIdx = headIdx
	// Set as hover root if mouse overlaps and z-index is higher.
	if rectOverlapsVec2(cnt.Rect, ctx.MousePos) &&
		(ctx.NextHoverRoot == nil || cnt.Zindex > ctx.NextHoverRoot.Zindex) {
		ctx.NextHoverRoot = cnt
	}
	// Reset clipping in case a root container is begun within another's
	// begin/end block.
	ctx.clipPush(unclippedRect)
}

// endRootContainer pushes a tail jump command and patches the head jump to
// skip past the container if it's empty/closed.
func (ctx *Context) endRootContainer() {
	cnt := ctx.GetCurrentContainer()
	tailIdx := len(ctx.CommandList)
	ctx.pushJump(-1)
	cnt.TailIdx = tailIdx
	// Head jump skips to the current end of command list (i.e. past this
	// container). End() may rewire this to chain to the next container.
	ctx.CommandList[cnt.HeadIdx].JumpDst = len(ctx.CommandList)
	ctx.PopClipRect()
	ctx.popContainer()
}

// popContainer pops the current container, layout, and id.
func (ctx *Context) popContainer() {
	cnt := ctx.GetCurrentContainer()
	layout := ctx.getLayout()
	cnt.ContentSize.X = layout.Max.X - layout.Body.X
	cnt.ContentSize.Y = layout.Max.Y - layout.Body.Y
	ctx.containerPop()
	ctx.layoutPop()
	ctx.PopID()
}

// BeginWindowEx begins a window. Returns ResActive if the window is open
// (caller should draw contents and call EndWindow).
func (ctx *Context) BeginWindowEx(title string, rect Rect, opt int) int {
	id := ctx.GetIDString(title)
	cnt := ctx.getContainer(id, opt)
	if cnt == nil || !cnt.Open {
		return 0
	}
	ctx.idPush(id)

	if cnt.Rect.W == 0 {
		cnt.Rect = rect
	}
	ctx.beginRootContainer(cnt)
	rect = cnt.Rect
	body := rect

	// Draw frame.
	if opt&OptNoFrame == 0 {
		ctx.DrawFrame(ctx, rect, ColorWindowBg)
	}

	// Title bar.
	if opt&OptNoTitle == 0 {
		tr := rect
		tr.H = ctx.Style.TitleHeight
		ctx.DrawFrame(ctx, tr, ColorTitleBg)

		// Title text + drag.
		{
			titleID := ctx.GetIDString("!title")
			ctx.UpdateControl(titleID, tr, opt)
			ctx.DrawControlText(title, tr, ColorTitleText, opt)
			if titleID == ctx.Focus && ctx.MouseDown == MouseLeft {
				cnt.Rect.X += ctx.MouseDelta.X
				cnt.Rect.Y += ctx.MouseDelta.Y
			}
			body.Y += tr.H
			body.H -= tr.H
		}

		// Close button.
		if opt&OptNoClose == 0 {
			closeID := ctx.GetIDString("!close")
			r := NewRect(tr.X+tr.W-tr.H, tr.Y, tr.H, tr.H)
			tr.W -= r.W
			ctx.DrawIcon(IconClose, r, ctx.Style.Colors[ColorTitleText])
			ctx.UpdateControl(closeID, r, opt)
			if ctx.MousePressed == MouseLeft && closeID == ctx.Focus {
				cnt.Open = false
			}
		}
	}

	ctx.pushContainerBody(cnt, body, opt)

	// Resize handle.
	if opt&OptNoResize == 0 {
		sz := ctx.Style.TitleHeight
		resizeID := ctx.GetIDString("!resize")
		r := NewRect(rect.X+rect.W-sz, rect.Y+rect.H-sz, sz, sz)
		ctx.UpdateControl(resizeID, r, opt)
		if resizeID == ctx.Focus && ctx.MouseDown == MouseLeft {
			cnt.Rect.W = maxInt(96, cnt.Rect.W+ctx.MouseDelta.X)
			cnt.Rect.H = maxInt(64, cnt.Rect.H+ctx.MouseDelta.Y)
		}
	}

	// Resize to content size.
	if opt&OptAutoSize != 0 {
		r := ctx.getLayout().Body
		cnt.Rect.W = cnt.ContentSize.X + (cnt.Rect.W - r.W)
		cnt.Rect.H = cnt.ContentSize.Y + (cnt.Rect.H - r.H)
	}

	// Close popup if elsewhere was clicked.
	if opt&OptPopup != 0 && ctx.MousePressed != 0 && ctx.HoverRoot != cnt {
		cnt.Open = false
	}

	ctx.PushClipRect(cnt.Body)
	return ResActive
}

// BeginWindow begins a window with default options.
func (ctx *Context) BeginWindow(title string, rect Rect) int {
	return ctx.BeginWindowEx(title, rect, 0)
}

// EndWindow ends a window opened by BeginWindow.
func (ctx *Context) EndWindow() {
	ctx.PopClipRect()
	ctx.endRootContainer()
}

// OpenPopup marks a popup named `name` for opening at the mouse cursor.
func (ctx *Context) OpenPopup(name string) {
	cnt := ctx.GetContainer(name)
	// Set as hover root so popup isn't closed in BeginWindowEx.
	ctx.HoverRoot = cnt
	ctx.NextHoverRoot = cnt
	// Position at mouse cursor, open, bring to front.
	cnt.Rect = NewRect(ctx.MousePos.X, ctx.MousePos.Y, 1, 1)
	cnt.Open = true
	ctx.BringToFront(cnt)
}

// BeginPopup begins a popup. Returns ResActive if open.
func (ctx *Context) BeginPopup(name string) int {
	opt := OptPopup | OptAutoSize | OptNoResize | OptNoScroll | OptNoTitle | OptClosed
	return ctx.BeginWindowEx(name, NewRect(0, 0, 0, 0), opt)
}

// EndPopup ends a popup opened by BeginPopup.
func (ctx *Context) EndPopup() {
	ctx.EndWindow()
}

// BeginPanelEx begins a panel at the next layout rect.
func (ctx *Context) BeginPanelEx(name string, opt int) {
	ctx.PushIDString(name)
	cnt := ctx.getContainer(ctx.LastID, opt)
	cnt.Rect = ctx.LayoutNext()
	if opt&OptNoFrame == 0 {
		ctx.DrawFrame(ctx, cnt.Rect, ColorPanelBg)
	}
	ctx.containerPush(cnt)
	ctx.pushContainerBody(cnt, cnt.Rect, opt)
	ctx.PushClipRect(cnt.Body)
}

// BeginPanel begins a panel with default options.
func (ctx *Context) BeginPanel(name string) {
	ctx.BeginPanelEx(name, 0)
}

// EndPanel ends a panel opened by BeginPanel.
func (ctx *Context) EndPanel() {
	ctx.PopClipRect()
	ctx.popContainer()
}
