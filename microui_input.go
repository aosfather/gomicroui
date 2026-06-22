package microui

// InputMouseMove updates the mouse position.
func (ctx *Context) InputMouseMove(x, y int) {
	ctx.MousePos = NewVec2(x, y)
}

// InputMouseDown presses mouse button btn at (x, y).
func (ctx *Context) InputMouseDown(x, y, btn int) {
	ctx.InputMouseMove(x, y)
	ctx.MouseDown |= btn
	ctx.MousePressed |= btn
}

// InputMouseUp releases mouse button btn at (x, y).
func (ctx *Context) InputMouseUp(x, y, btn int) {
	ctx.InputMouseMove(x, y)
	ctx.MouseDown &^= btn
}

// InputScroll accumulates scroll delta.
func (ctx *Context) InputScroll(x, y int) {
	ctx.ScrollDelta.X += x
	ctx.ScrollDelta.Y += y
}

// InputKeyDown presses key.
func (ctx *Context) InputKeyDown(key int) {
	ctx.KeyPressed |= key
	ctx.KeyDown |= key
}

// InputKeyUp releases key.
func (ctx *Context) InputKeyUp(key int) {
	ctx.KeyDown &^= key
}

// InputText appends text input for the frame.
func (ctx *Context) InputText(text string) {
	ctx.inputText += text
}
