package microui

import "testing"

func TestIDHashing(t *testing.T) {
	ctx := &Context{}
	Init(ctx)

	// Same input → same id.
	id1 := ctx.GetIDString("hello")
	id2 := ctx.GetIDString("hello")
	if id1 != id2 {
		t.Errorf("identical inputs produced different ids: %v vs %v", id1, id2)
	}

	// Different input → different id.
	id3 := ctx.GetIDString("world")
	if id1 == id3 {
		t.Errorf("different inputs produced same id: %v", id1)
	}

	// Empty stack uses the initial hash value.
	if id1 == 0 {
		t.Errorf("id should not be zero")
	}
}

func TestPushPopID(t *testing.T) {
	ctx := &Context{}
	Init(ctx)

	// Pushed id affects subsequent ids.
	ctx.PushIDString("parent")
	idWithParent := ctx.GetIDString("child")
	ctx.PopID()

	idWithoutParent := ctx.GetIDString("child")
	if idWithParent == idWithoutParent {
		t.Errorf("pushed id should change child id: %v == %v", idWithParent, idWithoutParent)
	}

	if len(ctx.IDStack) != 0 {
		t.Errorf("id stack should be empty after push/pop: %d", len(ctx.IDStack))
	}
}

func TestRectMath(t *testing.T) {
	r1 := NewRect(0, 0, 10, 10)
	r2 := NewRect(5, 5, 10, 10)
	got := intersectRects(r1, r2)
	want := NewRect(5, 5, 5, 5)
	if got != want {
		t.Errorf("intersect = %v, want %v", got, want)
	}

	// Non-overlapping.
	r3 := NewRect(20, 20, 5, 5)
	got = intersectRects(r1, r3)
	if got.W != 0 || got.H != 0 {
		t.Errorf("non-overlapping intersect = %v, want zero-size", got)
	}

	// Expand.
	got = expandRect(NewRect(10, 10, 5, 5), 2)
	want = NewRect(8, 8, 9, 9)
	if got != want {
		t.Errorf("expand = %v, want %v", got, want)
	}

	// Overlap test.
	if !rectOverlapsVec2(NewRect(0, 0, 10, 10), NewVec2(5, 5)) {
		t.Errorf("rect should overlap vec2")
	}
	if rectOverlapsVec2(NewRect(0, 0, 10, 10), NewVec2(10, 10)) {
		t.Errorf("rect should not overlap vec2 at corner (half-open)")
	}
}

func TestStacks(t *testing.T) {
	ctx := &Context{}
	Init(ctx)

	// Push and pop clip rects.
	ctx.ClipStack = append(ctx.ClipStack, NewRect(0, 0, 100, 100))
	ctx.PushClipRect(NewRect(10, 10, 50, 50))
	got := ctx.GetClipRect()
	if got.X != 10 || got.Y != 10 || got.W != 50 || got.H != 50 {
		t.Errorf("clip rect = %v, want (10,10,50,50)", got)
	}
	ctx.PopClipRect()
	if ctx.GetClipRect().W != 100 {
		t.Errorf("after pop, clip rect should be original")
	}
}

func TestCommandList(t *testing.T) {
	ctx := &Context{}
	Init(ctx)
	// TextWidth/TextHeight required by Begin.
	ctx.TextWidth = func(Font, string) int { return 8 }
	ctx.TextHeight = func(Font) int { return 10 }

	Begin(ctx)
	// Drawing requires a clip rect to be active (normally pushed by a
	// container); push the initial unclipped rect directly via the internal
	// helper (PushClipRect intersects with the existing top, so it requires
	// a non-empty stack).
	ctx.clipPush(unclippedRect)
	ctx.DrawRect(NewRect(0, 0, 10, 10), NewColor(255, 0, 0, 255))
	ctx.DrawRect(NewRect(20, 20, 10, 10), NewColor(0, 255, 0, 255))
	ctx.PopClipRect()
	End(ctx)

	var cmd *Command
	count := 0
	for ctx.NextCommand(&cmd) {
		count++
		if cmd.Kind != CommandRect {
			t.Errorf("command %d kind = %v, want CommandRect", count, cmd.Kind)
		}
	}
	if count != 2 {
		t.Errorf("got %d commands, want 2", count)
	}
}

func TestCommandListJumps(t *testing.T) {
	// Test that jump commands are followed during iteration.
	ctx := &Context{}
	Init(ctx)
	ctx.TextWidth = func(Font, string) int { return 8 }
	ctx.TextHeight = func(Font) int { return 10 }

	// Manually build a command list with jumps to verify NextCommand
	// follows them.
	Begin(ctx)
	// cmd[0]: jump to cmd[3]
	ctx.CommandList = append(ctx.CommandList, Command{Kind: CommandJump, JumpDst: 3})
	// cmd[1]: skipped (jump target is 3, not 1)
	ctx.CommandList = append(ctx.CommandList, Command{Kind: CommandRect, Rect: NewRect(1, 1, 1, 1)})
	// cmd[2]: skipped
	ctx.CommandList = append(ctx.CommandList, Command{Kind: CommandRect, Rect: NewRect(2, 2, 2, 2)})
	// cmd[3]: jump destination — should be returned
	ctx.CommandList = append(ctx.CommandList, Command{Kind: CommandRect, Rect: NewRect(3, 3, 3, 3)})
	// No root containers, so End() won't touch the jump destinations.
	End(ctx)

	var cmd *Command
	var rects []Rect
	for ctx.NextCommand(&cmd) {
		if cmd.Kind != CommandRect {
			t.Errorf("expected only Rect commands, got kind %d", cmd.Kind)
		}
		rects = append(rects, cmd.Rect)
	}
	if len(rects) != 1 {
		t.Fatalf("got %d rects, want 1 (jump should skip 2)", len(rects))
	}
	if rects[0].X != 3 {
		t.Errorf("rect.X = %d, want 3", rects[0].X)
	}
}

func TestLayoutRow(t *testing.T) {
	ctx := &Context{}
	Init(ctx)
	ctx.TextWidth = func(Font, string) int { return 8 }
	ctx.TextHeight = func(Font) int { return 10 }

	Begin(ctx)
	// Simulate a container body via pushLayout.
	ctx.pushLayout(NewRect(0, 0, 100, 100), NewVec2(0, 0))
	ctx.LayoutRow(2, []int{30, -1}, 20)

	r1 := ctx.LayoutNext()
	r2 := ctx.LayoutNext()

	if r1.W != 30 {
		t.Errorf("r1.W = %d, want 30", r1.W)
	}
	if r1.H != 20 {
		t.Errorf("r1.H = %d, want 20", r1.H)
	}
	// Negative width: -1 means "fill remaining" → 100 - 30 - spacing + 1.
	if r2.W <= 0 {
		t.Errorf("r2.W = %d, should be positive (fill remaining)", r2.W)
	}
	if r2.H != 20 {
		t.Errorf("r2.H = %d, want 20", r2.H)
	}
	ctx.layoutPop()
	End(ctx)
}

func TestCheckbox(t *testing.T) {
	ctx := &Context{}
	Init(ctx)
	ctx.TextWidth = func(Font, string) int { return 8 }
	ctx.TextHeight = func(Font) int { return 10 }

	Begin(ctx)
	ctx.clipPush(unclippedRect)
	ctx.pushLayout(NewRect(0, 0, 100, 100), NewVec2(0, 0))
	state := false
	// Without input, checkbox should not change state.
	ctx.Checkbox("test", &state)
	if state {
		t.Errorf("state should remain false without input")
	}
	ctx.layoutPop()
	ctx.PopClipRect()
	End(ctx)
}

func TestWindowLifecycle(t *testing.T) {
	ctx := &Context{}
	Init(ctx)
	ctx.TextWidth = func(Font, string) int { return 8 }
	ctx.TextHeight = func(Font) int { return 10 }

	// A full frame with a window containing a button and a label.
	Begin(ctx)
	if ctx.BeginWindow("Test", NewRect(10, 10, 200, 100)) != 0 {
		ctx.LayoutRow(2, []int{60, -1}, 0)
		ctx.Label("Hello:")
		ctx.Button("Click")
		ctx.EndWindow()
	}
	End(ctx)

	// Count commands — should have at least the window frame + label + button.
	var cmd *Command
	count := 0
	for ctx.NextCommand(&cmd) {
		count++
	}
	if count == 0 {
		t.Errorf("expected some commands after drawing a window, got 0")
	}

	// Second frame: window should still be open (retained state).
	Begin(ctx)
	open := ctx.BeginWindow("Test", NewRect(10, 10, 200, 100))
	if open == 0 {
		t.Errorf("window should remain open on second frame")
	}
	if open != 0 {
		ctx.EndWindow()
	}
	End(ctx)
}

func TestPopup(t *testing.T) {
	ctx := &Context{}
	Init(ctx)
	ctx.TextWidth = func(Font, string) int { return 8 }
	ctx.TextHeight = func(Font) int { return 10 }

	Begin(ctx)
	// OpenPopup sets up the popup; BeginPopup should then open it.
	ctx.OpenPopup("MyPopup")
	if ctx.BeginPopup("MyPopup") != 0 {
		ctx.Label("Inside popup")
		ctx.EndPopup()
	}
	End(ctx)
}
