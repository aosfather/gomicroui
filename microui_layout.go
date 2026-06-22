package microui

// Layout-internal next-type modes.
const (
	layoutRelative = 1
	layoutAbsolute = 2
)

// pushLayout pushes a fresh layout for a container body of the given size,
// offset by scroll. The first row is initialised with a single full-width
// item of zero height (matching the C behaviour).
func (ctx *Context) pushLayout(body Rect, scroll Vec2) {
	var l Layout
	l.Body = NewRect(body.X-scroll.X, body.Y-scroll.Y, body.W, body.H)
	l.Max = NewVec2(-0x1000000, -0x1000000)
	ctx.layoutPush(l)
	width := 0
	ctx.LayoutRow(1, []int{width}, 0)
}

// getLayout returns a pointer to the layout at the top of the stack.
func (ctx *Context) getLayout() *Layout {
	expect(len(ctx.LayoutStack) > 0, "layout stack empty")
	return &ctx.LayoutStack[len(ctx.LayoutStack)-1]
}

// LayoutBeginColumn pushes a new layout rooted at the next layout rect.
func (ctx *Context) LayoutBeginColumn() {
	ctx.pushLayout(ctx.LayoutNext(), NewVec2(0, 0))
}

// LayoutEndColumn pops the current layout, inheriting position/next_row/max
// into the parent layout if greater.
func (ctx *Context) LayoutEndColumn() {
	b := ctx.getLayout()
	ctx.layoutPop()
	a := ctx.getLayout()
	a.Position.X = maxInt(a.Position.X, b.Position.X+b.Body.X-a.Body.X)
	a.NextRow = maxInt(a.NextRow, b.NextRow+b.Body.Y-a.Body.Y)
	a.Max.X = maxInt(a.Max.X, b.Max.X)
	a.Max.Y = maxInt(a.Max.Y, b.Max.Y)
}

// LayoutRow begins a new row of `items` items with the given widths and
// height. If widths is nil, the previous widths are reused.
func (ctx *Context) LayoutRow(items int, widths []int, height int) {
	layout := ctx.getLayout()
	if widths != nil {
		expect(items <= MaxWidths, "LayoutRow: too many items")
		copy(layout.Widths[:items], widths)
	}
	layout.Items = items
	layout.Position = NewVec2(layout.Indent, layout.NextRow)
	layout.Size.Y = height
	layout.ItemIndex = 0
}

// LayoutWidth sets the width of the next laid-out item.
func (ctx *Context) LayoutWidth(width int) {
	ctx.getLayout().Size.X = width
}

// LayoutHeight sets the height of the current row.
func (ctx *Context) LayoutHeight(height int) {
	ctx.getLayout().Size.Y = height
}

// LayoutSetNext overrides the rect used for the next call to LayoutNext.
// If relative is true, the rect is treated as relative to the body; otherwise
// it is absolute.
func (ctx *Context) LayoutSetNext(r Rect, relative bool) {
	layout := ctx.getLayout()
	layout.Next = r
	if relative {
		layout.NextType = layoutRelative
	} else {
		layout.NextType = layoutAbsolute
	}
}

// LayoutNext returns the next rect in the current layout, advancing the
// layout position. Updates ctx.LastRect.
func (ctx *Context) LayoutNext() Rect {
	layout := ctx.getLayout()
	style := ctx.Style
	var res Rect

	if layout.NextType != 0 {
		// Handle rect set by LayoutSetNext.
		typ := layout.NextType
		layout.NextType = 0
		res = layout.Next
		if typ == layoutAbsolute {
			ctx.LastRect = res
			return res
		}
	} else {
		// Handle next row when the current row is full.
		if layout.ItemIndex == layout.Items {
			ctx.LayoutRow(layout.Items, nil, layout.Size.Y)
		}

		// Position.
		res.X = layout.Position.X
		res.Y = layout.Position.Y

		// Size.
		if layout.Items > 0 {
			res.W = layout.Widths[layout.ItemIndex]
		} else {
			res.W = layout.Size.X
		}
		res.H = layout.Size.Y
		if res.W == 0 {
			res.W = style.Size.X + style.Padding*2
		}
		if res.H == 0 {
			res.H = style.Size.Y + style.Padding*2
		}
		if res.W < 0 {
			res.W += layout.Body.W - res.X + 1
		}
		if res.H < 0 {
			res.H += layout.Body.H - res.Y + 1
		}

		layout.ItemIndex++
	}

	// Update position.
	layout.Position.X += res.W + style.Spacing
	layout.NextRow = maxInt(layout.NextRow, res.Y+res.H+style.Spacing)

	// Apply body offset.
	res.X += layout.Body.X
	res.Y += layout.Body.Y

	// Update max position.
	layout.Max.X = maxInt(layout.Max.X, res.X+res.W)
	layout.Max.Y = maxInt(layout.Max.Y, res.Y+res.H)

	ctx.LastRect = res
	return res
}
