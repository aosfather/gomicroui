package microui

import "unsafe"

// pushCommand appends a new command of the given kind and returns a pointer
// to it. The caller fills in the kind-specific fields.
func (ctx *Context) pushCommand(kind int) *Command {
	expect(len(ctx.CommandList) < CommandListSize, "command list overflow")
	ctx.CommandList = append(ctx.CommandList, Command{Kind: kind})
	return &ctx.CommandList[len(ctx.CommandList)-1]
}

// pushJump appends a JUMP command pointing at dst and returns a pointer to it.
// dst may be -1 (placeholder) and patched later.
func (ctx *Context) pushJump(dst int) *Command {
	cmd := ctx.pushCommand(CommandJump)
	cmd.JumpDst = dst
	return cmd
}

// NextCommand iterates the command list in z-index order, following jump
// commands. Call with cmd == nil to start; pass the same &cmd each call.
// Returns false when iteration is complete (cmd is set to nil).
//
// Usage:
//
//	var cmd *microui.Command
//	for ctx.NextCommand(&cmd) {
//	    switch cmd.Kind { ... }
//	}
func (ctx *Context) NextCommand(cmd **Command) bool {
	var idx int
	if *cmd != nil {
		// Advance past the last returned command.
		idx = commandIndexOf(ctx, *cmd) + 1
	}
	for idx < len(ctx.CommandList) {
		c := &ctx.CommandList[idx]
		if c.Kind != CommandJump {
			*cmd = c
			return true
		}
		idx = c.JumpDst
	}
	*cmd = nil
	return false
}

// commandIndexOf returns the slice index of cmd within ctx.CommandList,
// computed by pointer arithmetic. This is safe because the CommandList slice
// backing array is stable for the duration of a frame (Begin resets length
// to 0 but does not reallocate once the slice has reached steady-state cap).
func commandIndexOf(ctx *Context, cmd *Command) int {
	if len(ctx.CommandList) == 0 {
		return -1
	}
	base := &ctx.CommandList[0]
	diff := uintptr(unsafe.Pointer(cmd)) - uintptr(unsafe.Pointer(base))
	size := unsafe.Sizeof(Command{})
	return int(diff / uintptr(size))
}

// SetClip pushes a CLIP command.
func (ctx *Context) SetClip(rect Rect) {
	cmd := ctx.pushCommand(CommandClip)
	cmd.Rect = rect
}

// DrawRect pushes a RECT command, clipped to the current clip rect. No-op if
// the result is empty.
func (ctx *Context) DrawRect(rect Rect, color Color) {
	rect = intersectRects(rect, ctx.GetClipRect())
	if rect.W > 0 && rect.H > 0 {
		cmd := ctx.pushCommand(CommandRect)
		cmd.Rect = rect
		cmd.Color = color
	}
}

// DrawBox draws a 1-pixel border around rect using four filled rects.
func (ctx *Context) DrawBox(rect Rect, color Color) {
	ctx.DrawRect(NewRect(rect.X+1, rect.Y, rect.W-2, 1), color)
	ctx.DrawRect(NewRect(rect.X+1, rect.Y+rect.H-1, rect.W-2, 1), color)
	ctx.DrawRect(NewRect(rect.X, rect.Y, 1, rect.H), color)
	ctx.DrawRect(NewRect(rect.X+rect.W-1, rect.Y, 1, rect.H), color)
}

// DrawText pushes a TEXT command. str is the text to draw; pos is the
// baseline position. The text is clipped to the current clip rect.
func (ctx *Context) DrawText(font Font, str string, pos Vec2, color Color) {
	rect := NewRect(pos.X, pos.Y, ctx.TextWidth(font, str), ctx.TextHeight(font))
	clipped := ctx.CheckClip(rect)
	if clipped == ClipAll {
		return
	}
	if clipped == ClipPart {
		ctx.SetClip(ctx.GetClipRect())
	}
	cmd := ctx.pushCommand(CommandText)
	cmd.Font = font
	cmd.Pos = pos
	cmd.Color = color
	cmd.Text = str
	if clipped != 0 {
		ctx.SetClip(unclippedRect)
	}
}

// DrawIcon pushes an ICON command, clipped to the current clip rect.
func (ctx *Context) DrawIcon(id int, rect Rect, color Color) {
	clipped := ctx.CheckClip(rect)
	if clipped == ClipAll {
		return
	}
	if clipped == ClipPart {
		ctx.SetClip(ctx.GetClipRect())
	}
	cmd := ctx.pushCommand(CommandIcon)
	cmd.IconID = id
	cmd.Rect = rect
	cmd.Color = color
	if clipped != 0 {
		ctx.SetClip(unclippedRect)
	}
}
