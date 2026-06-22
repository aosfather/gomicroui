// microui demo: GLFW event loop + demo windows. Ported from demo/main.c.
package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/go-gl/glfw/v3.3/glfw"

	"microui"
)

// Globals ported from main.c.
var (
	logbuf         strings.Builder
	logbufUpdated  bool
	bg             = [3]float32{90, 95, 100}
	logTextboxBuf  string
	styleChecks    = [3]bool{true, false, true}
	uint8SliderTmp float32
	ctx            *microui.Context
)

func writeLog(text string) {
	if logbuf.Len() > 0 {
		logbuf.WriteByte('\n')
	}
	logbuf.WriteString(text)
	logbufUpdated = true
}

func testWindow(ctx *microui.Context) {
	if ctx.BeginWindow("Demo Window", microui.NewRect(40, 40, 300, 450)) != 0 {
		win := ctx.GetCurrentContainer()
		win.Rect.W = maxInt(win.Rect.W, 240)
		win.Rect.H = maxInt(win.Rect.H, 300)

		// Window info.
		if ctx.Header("Window Info") != 0 {
			win := ctx.GetCurrentContainer()
			ctx.LayoutRow(2, []int{54, -1}, 0)
			ctx.Label("Position:")
			ctx.Label(fmt.Sprintf("%d, %d", win.Rect.X, win.Rect.Y))
			ctx.Label("Size:")
			ctx.Label(fmt.Sprintf("%d, %d", win.Rect.W, win.Rect.H))
		}

		// Test buttons.
		if ctx.HeaderEx("Test Buttons", microui.OptExpanded) != 0 {
			ctx.LayoutRow(3, []int{86, -110, -1}, 0)
			ctx.Label("Test buttons 1:")
			if ctx.Button("Button 1") != 0 {
				writeLog("Pressed button 1")
			}
			if ctx.Button("Button 2") != 0 {
				writeLog("Pressed button 2")
			}
			ctx.Label("Test buttons 2:")
			if ctx.Button("Button 3") != 0 {
				writeLog("Pressed button 3")
			}
			if ctx.Button("Popup") != 0 {
				ctx.OpenPopup("Test Popup")
			}
			if ctx.BeginPopup("Test Popup") != 0 {
				ctx.Button("Hello")
				ctx.Button("World")
				ctx.EndPopup()
			}
		}

		// Tree and text.
		if ctx.HeaderEx("Tree and Text", microui.OptExpanded) != 0 {
			ctx.LayoutRow(2, []int{140, -1}, 0)
			ctx.LayoutBeginColumn()
			if ctx.BeginTreeNode("Test 1") != 0 {
				if ctx.BeginTreeNode("Test 1a") != 0 {
					ctx.Label("Hello")
					ctx.Label("world")
					ctx.EndTreeNode()
				}
				if ctx.BeginTreeNode("Test 1b") != 0 {
					if ctx.Button("Button 1") != 0 {
						writeLog("Pressed button 1")
					}
					if ctx.Button("Button 2") != 0 {
						writeLog("Pressed button 2")
					}
					ctx.EndTreeNode()
				}
				ctx.EndTreeNode()
			}
			if ctx.BeginTreeNode("Test 2") != 0 {
				ctx.LayoutRow(2, []int{54, 54}, 0)
				if ctx.Button("Button 3") != 0 {
					writeLog("Pressed button 3")
				}
				if ctx.Button("Button 4") != 0 {
					writeLog("Pressed button 4")
				}
				if ctx.Button("Button 5") != 0 {
					writeLog("Pressed button 5")
				}
				if ctx.Button("Button 6") != 0 {
					writeLog("Pressed button 6")
				}
				ctx.EndTreeNode()
			}
			if ctx.BeginTreeNode("Test 3") != 0 {
				ctx.Checkbox("Checkbox 1", &styleChecks[0])
				ctx.Checkbox("Checkbox 2", &styleChecks[1])
				ctx.Checkbox("Checkbox 3", &styleChecks[2])
				ctx.EndTreeNode()
			}
			ctx.LayoutEndColumn()

			ctx.LayoutBeginColumn()
			ctx.LayoutRow(1, []int{-1}, 0)
			ctx.Text("Lorem ipsum dolor sit amet, consectetur adipiscing " +
				"elit. Maecenas lacinia, sem eu lacinia molestie, mi risus faucibus " +
				"ipsum, eu varius magna felis a nulla.")
			ctx.LayoutEndColumn()
		}

		// Background color sliders.
		if ctx.HeaderEx("Background Color", microui.OptExpanded) != 0 {
			ctx.LayoutRow(2, []int{-78, -1}, 74)
			ctx.LayoutBeginColumn()
			ctx.LayoutRow(2, []int{46, -1}, 0)
			ctx.Label("Red:")
			ctx.Slider(&bg[0], 0, 255)
			ctx.Label("Green:")
			ctx.Slider(&bg[1], 0, 255)
			ctx.Label("Blue:")
			ctx.Slider(&bg[2], 0, 255)
			ctx.LayoutEndColumn()
			r := ctx.LayoutNext()
			ctx.DrawRect(r, microui.NewColor(int(bg[0]), int(bg[1]), int(bg[2]), 255))
			buf := fmt.Sprintf("#%02X%02X%02X", int(bg[0]), int(bg[1]), int(bg[2]))
			ctx.DrawControlText(buf, r, microui.ColorText, microui.OptAlignCenter)
		}

		ctx.EndWindow()
	}
}

func logWindow(ctx *microui.Context) {
	if ctx.BeginWindow("Log Window", microui.NewRect(350, 40, 300, 200)) != 0 {
		ctx.LayoutRow(1, []int{-1}, -25)
		ctx.BeginPanel("Log Output")
		panel := ctx.GetCurrentContainer()
		ctx.LayoutRow(1, []int{-1}, -1)
		ctx.Text(logbuf.String())
		ctx.EndPanel()
		if logbufUpdated {
			panel.Scroll.Y = panel.ContentSize.Y
			logbufUpdated = false
		}

		submitted := false
		ctx.LayoutRow(2, []int{-70, -1}, 0)
		if ctx.Textbox(&logTextboxBuf, 128)&microui.ResSubmit != 0 {
			ctx.SetFocus(ctx.LastID)
			submitted = true
		}
		if ctx.Button("Submit") != 0 {
			submitted = true
		}
		if submitted {
			writeLog(logTextboxBuf)
			logTextboxBuf = ""
		}

		ctx.EndWindow()
	}
}

func uint8Slider(ctx *microui.Context, value *byte, low, high int) int {
	uint8SliderTmp = float32(*value)
	res := ctx.SliderEx(&uint8SliderTmp, float32(low), float32(high), 0, "%.0f", microui.OptAlignCenter)
	*value = byte(uint8SliderTmp)
	return res
}

var styleColors = []struct {
	label string
	idx   int
}{
	{"text:", microui.ColorText},
	{"border:", microui.ColorBorder},
	{"windowbg:", microui.ColorWindowBg},
	{"titlebg:", microui.ColorTitleBg},
	{"titletext:", microui.ColorTitleText},
	{"panelbg:", microui.ColorPanelBg},
	{"button:", microui.ColorButton},
	{"buttonhover:", microui.ColorButtonHover},
	{"buttonfocus:", microui.ColorButtonFocus},
	{"base:", microui.ColorBase},
	{"basehover:", microui.ColorBaseHover},
	{"basefocus:", microui.ColorBaseFocus},
	{"scrollbase:", microui.ColorScrollBase},
	{"scrollthumb:", microui.ColorScrollThumb},
}

func styleWindow(ctx *microui.Context) {
	if ctx.BeginWindow("Style Editor", microui.NewRect(350, 250, 300, 240)) != 0 {
		sw := int(float32(ctx.GetCurrentContainer().Body.W) * 0.14)
		ctx.LayoutRow(6, []int{80, sw, sw, sw, sw, -1}, 0)
		for _, c := range styleColors {
			ctx.Label(c.label)
			uint8Slider(ctx, &ctx.Style.Colors[c.idx].R, 0, 255)
			uint8Slider(ctx, &ctx.Style.Colors[c.idx].G, 0, 255)
			uint8Slider(ctx, &ctx.Style.Colors[c.idx].B, 0, 255)
			uint8Slider(ctx, &ctx.Style.Colors[c.idx].A, 0, 255)
			ctx.DrawRect(ctx.LayoutNext(), ctx.Style.Colors[c.idx])
		}
		ctx.EndWindow()
	}
}

func processFrame(ctx *microui.Context) {
	microui.Begin(ctx)
	styleWindow(ctx)
	logWindow(ctx)
	testWindow(ctx)
	microui.End(ctx)
}

// maxInt is a local helper (the microui package's maxInt is unexported).
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// buttonMap maps GLFW mouse button codes to microui mouse button flags.
func buttonMap(b glfw.MouseButton) int {
	switch b {
	case glfw.MouseButtonLeft:
		return microui.MouseLeft
	case glfw.MouseButtonRight:
		return microui.MouseRight
	case glfw.MouseButtonMiddle:
		return microui.MouseMiddle
	}
	return 0
}

// keyMap maps GLFW key codes to microui key flags.
func keyMap(key glfw.Key) int {
	switch key {
	case glfw.KeyLeftShift, glfw.KeyRightShift:
		return microui.KeyShift
	case glfw.KeyLeftControl, glfw.KeyRightControl:
		return microui.KeyCtrl
	case glfw.KeyLeftAlt, glfw.KeyRightAlt:
		return microui.KeyAlt
	case glfw.KeyEnter:
		return microui.KeyReturn
	case glfw.KeyBackspace:
		return microui.KeyBackspace
	}
	return 0
}

func textWidth(font microui.Font, text string) int {
	return renderer.GetTextWidth(text)
}

func textHeight(font microui.Font) int {
	return renderer.GetTextHeight()
}

var renderer *Renderer

func main() {
	runtime.LockOSThread()

	var err error
	renderer, err = NewRenderer(800, 600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "renderer: %v\n", err)
		return
	}
	defer glfw.Terminate()

	ctx = &microui.Context{}
	microui.Init(ctx)
	ctx.TextWidth = textWidth
	ctx.TextHeight = textHeight

	window := renderer.window
	sx, sy := window.GetContentScale()
	scaleX, scaleY := float64(sx), float64(sy)

	window.SetCursorPosCallback(func(w *glfw.Window, x, y float64) {
		ctx.InputMouseMove(int(x*scaleX), int(y*scaleY))
	})

	window.SetMouseButtonCallback(func(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
		x, y := w.GetCursorPos()
		b := buttonMap(button)
		if b != 0 {
			if action == glfw.Press {
				ctx.InputMouseDown(int(x*scaleX), int(y*scaleY), b)
			} else {
				ctx.InputMouseUp(int(x*scaleX), int(y*scaleY), b)
			}
		}
	})

	window.SetScrollCallback(func(w *glfw.Window, xoff, yoff float64) {
		ctx.InputScroll(int(xoff*30), int(-yoff*30))
	})

	window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		c := keyMap(key)
		if c != 0 {
			if action == glfw.Press || action == glfw.Repeat {
				ctx.InputKeyDown(c)
			} else {
				ctx.InputKeyUp(c)
			}
		}
	})

	window.SetCharCallback(func(w *glfw.Window, char rune) {
		ctx.InputText(string(char))
	})

	window.SetFramebufferSizeCallback(func(w *glfw.Window, width, height int) {
		renderer.width = width
		renderer.height = height
		sx, sy := w.GetContentScale()
		scaleX, scaleY = float64(sx), float64(sy)
	})

	for !window.ShouldClose() {
		processFrame(ctx)

		renderer.Clear(microui.NewColor(int(bg[0]), int(bg[1]), int(bg[2]), 255))
		var cmd *microui.Command
		for ctx.NextCommand(&cmd) {
			switch cmd.Kind {
			case microui.CommandText:
				renderer.DrawText(cmd.Text, cmd.Pos, cmd.Color)
			case microui.CommandRect:
				renderer.DrawRect(cmd.Rect, cmd.Color)
			case microui.CommandIcon:
				renderer.DrawIcon(cmd.IconID, cmd.Rect, cmd.Color)
			case microui.CommandClip:
				renderer.SetClipRect(cmd.Rect)
			}
		}
		renderer.Present()

		glfw.PollEvents()
	}
}
