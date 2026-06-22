// GLFW + OpenGL renderer for the microui demo. Ported from demo/renderer.c.
package main

import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/go-gl/glfw/v3.3/glfw"

	"microui"
)

const bufferSize = 16384

// Renderer holds the GLFW window and GL buffer state.
type Renderer struct {
	window *glfw.Window

	width  int
	height int
	bufIdx int

	texBuf  []float32
	vertBuf []float32
	colBuf  []uint8
	idxBuf  []uint32

	pinner *runtime.Pinner
}

// NewRenderer creates a new renderer with a GLFW window of the given size.
func NewRenderer(w, h int) (*Renderer, error) {
	r := &Renderer{width: w, height: h}

	r.texBuf = make([]float32, bufferSize*8)
	r.vertBuf = make([]float32, bufferSize*8)
	r.colBuf = make([]uint8, bufferSize*16)
	r.idxBuf = make([]uint32, bufferSize*6)

	r.pinner = &runtime.Pinner{}
	r.pinner.Pin(&r.texBuf[0])
	r.pinner.Pin(&r.vertBuf[0])
	r.pinner.Pin(&r.colBuf[0])
	r.pinner.Pin(&r.idxBuf[0])

	if err := glfw.Init(); err != nil {
		return nil, fmt.Errorf("glfw init: %w", err)
	}

	glfw.WindowHint(glfw.ContextVersionMajor, 2)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.ScaleToMonitor, 0)

	window, err := glfw.CreateWindow(w, h, "microui", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("create window: %w", err)
	}
	r.window = window

	window.MakeContextCurrent()

	if err := gl.Init(); err != nil {
		return nil, fmt.Errorf("gl init: %w", err)
	}

	fbWidth, fbHeight := window.GetFramebufferSize()
	r.width = fbWidth
	r.height = fbHeight

	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Disable(gl.CULL_FACE)
	gl.Disable(gl.DEPTH_TEST)
	gl.Enable(gl.SCISSOR_TEST)
	gl.Enable(gl.TEXTURE_2D)
	gl.EnableClientState(gl.VERTEX_ARRAY)
	gl.EnableClientState(gl.TEXTURE_COORD_ARRAY)
	gl.EnableClientState(gl.COLOR_ARRAY)

	var texID uint32
	gl.GenTextures(1, &texID)
	gl.BindTexture(gl.TEXTURE_2D, texID)
	gl.TexImage2D(
		gl.TEXTURE_2D, 0, gl.ALPHA,
		int32(AtlasWidth), int32(AtlasHeight), 0,
		gl.ALPHA, gl.UNSIGNED_BYTE,
		unsafe.Pointer(&atlasTexture[0]),
	)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	if e := gl.GetError(); e != 0 {
		return nil, fmt.Errorf("gl error after init: %d", e)
	}

	return r, nil
}

// flush submits buffered quads to GL.
func (r *Renderer) flush() {
	if r.bufIdx == 0 {
		return
	}

	gl.Viewport(0, 0, int32(r.width), int32(r.height))
	gl.MatrixMode(gl.PROJECTION)
	gl.PushMatrix()
	gl.LoadIdentity()
	gl.Ortho(0, float64(r.width), float64(r.height), 0, -1, 1)
	gl.MatrixMode(gl.MODELVIEW)
	gl.PushMatrix()
	gl.LoadIdentity()

	gl.TexCoordPointer(2, gl.FLOAT, 0, unsafe.Pointer(&r.texBuf[0]))
	gl.VertexPointer(2, gl.FLOAT, 0, unsafe.Pointer(&r.vertBuf[0]))
	gl.ColorPointer(4, gl.UNSIGNED_BYTE, 0, unsafe.Pointer(&r.colBuf[0]))
	gl.DrawElements(gl.TRIANGLES, int32(r.bufIdx*6), gl.UNSIGNED_INT, unsafe.Pointer(&r.idxBuf[0]))

	gl.MatrixMode(gl.MODELVIEW)
	gl.PopMatrix()
	gl.MatrixMode(gl.PROJECTION)
	gl.PopMatrix()

	r.bufIdx = 0
}

// pushQuad buffers a single textured quad (two triangles).
func (r *Renderer) pushQuad(dst, src microui.Rect, color microui.Color) {
	if r.bufIdx == bufferSize {
		r.flush()
	}

	tvi := r.bufIdx * 8
	ci := r.bufIdx * 16
	ei := r.bufIdx * 4
	ii := r.bufIdx * 6
	r.bufIdx++

	// Texture coords.
	x := float32(src.X) / float32(AtlasWidth)
	y := float32(src.Y) / float32(AtlasHeight)
	w := float32(src.W) / float32(AtlasWidth)
	h := float32(src.H) / float32(AtlasHeight)
	r.texBuf[tvi+0] = x
	r.texBuf[tvi+1] = y
	r.texBuf[tvi+2] = x + w
	r.texBuf[tvi+3] = y
	r.texBuf[tvi+4] = x
	r.texBuf[tvi+5] = y + h
	r.texBuf[tvi+6] = x + w
	r.texBuf[tvi+7] = y + h

	// Vertex coords.
	r.vertBuf[tvi+0] = float32(dst.X)
	r.vertBuf[tvi+1] = float32(dst.Y)
	r.vertBuf[tvi+2] = float32(dst.X + dst.W)
	r.vertBuf[tvi+3] = float32(dst.Y)
	r.vertBuf[tvi+4] = float32(dst.X)
	r.vertBuf[tvi+5] = float32(dst.Y + dst.H)
	r.vertBuf[tvi+6] = float32(dst.X + dst.W)
	r.vertBuf[tvi+7] = float32(dst.Y + dst.H)

	// Colors (4 bytes per vertex, 4 vertices).
	for i := 0; i < 4; i++ {
		r.colBuf[ci+i*4+0] = color.R
		r.colBuf[ci+i*4+1] = color.G
		r.colBuf[ci+i*4+2] = color.B
		r.colBuf[ci+i*4+3] = color.A
	}

	// Indices.
	r.idxBuf[ii+0] = uint32(ei + 0)
	r.idxBuf[ii+1] = uint32(ei + 1)
	r.idxBuf[ii+2] = uint32(ei + 2)
	r.idxBuf[ii+3] = uint32(ei + 2)
	r.idxBuf[ii+4] = uint32(ei + 3)
	r.idxBuf[ii+5] = uint32(ei + 1)
}

// DrawRect draws a filled rect.
func (r *Renderer) DrawRect(rect microui.Rect, color microui.Color) {
	r.pushQuad(rect, atlas[AtlasWhite], color)
}

// DrawText draws text starting at pos, skipping UTF-8 continuation bytes.
func (r *Renderer) DrawText(text string, pos microui.Vec2, color microui.Color) {
	dst := microui.NewRect(pos.X, pos.Y, 0, 0)
	for i := 0; i < len(text); i++ {
		c := text[i]
		if c&0xc0 == 0x80 {
			continue
		}
		chr := int(c)
		if chr > 127 {
			chr = 127
		}
		src := atlas[AtlasFont+chr]
		dst.W = src.W
		dst.H = src.H
		r.pushQuad(dst, src, color)
		dst.X += dst.W
	}
}

// DrawIcon draws an icon centered in rect.
func (r *Renderer) DrawIcon(id int, rect microui.Rect, color microui.Color) {
	src := atlas[id]
	x := rect.X + (rect.W-src.W)/2
	y := rect.Y + (rect.H-src.H)/2
	r.pushQuad(microui.NewRect(x, y, src.W, src.H), src, color)
}

// GetTextWidth returns the pixel width of text.
func (r *Renderer) GetTextWidth(text string) int {
	res := 0
	for i := 0; i < len(text); i++ {
		c := text[i]
		if c&0xc0 == 0x80 {
			continue
		}
		chr := int(c)
		if chr > 127 {
			chr = 127
		}
		res += atlas[AtlasFont+chr].W
	}
	return res
}

// GetTextHeight returns the text height.
func (r *Renderer) GetTextHeight() int { return 18 }

// SetClipRect sets the GL scissor rect (Y-flipped for GL coords).
func (r *Renderer) SetClipRect(rect microui.Rect) {
	r.flush()
	gl.Scissor(int32(rect.X), int32(r.height-(rect.Y+rect.H)), int32(rect.W), int32(rect.H))
}

// Clear clears the framebuffer.
func (r *Renderer) Clear(color microui.Color) {
	r.flush()
	gl.ClearColor(float32(color.R)/255, float32(color.G)/255, float32(color.B)/255, float32(color.A)/255)
	gl.Clear(gl.COLOR_BUFFER_BIT)
}

// Present swaps the window buffers.
func (r *Renderer) Present() {
	r.flush()
	r.window.SwapBuffers()
}
