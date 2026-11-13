package render

import (
	"fmt"
	"math"

	"github.com/go-gl/gl/v4.5-core/gl"
)

// SunShadow is a real-time directional shadow map: one orthographic depth
// view from the sun, centred on the player and snapped to whole texels so the
// edges do not shimmer while moving. Receivers sample it with 3x3 hardware PCF.
type SunShadow struct {
	fbo, depth uint32
	size       int32
	bytes      uint64
	// Matrix maps world positions to light clip space (z in [0,1]).
	Matrix M4
	// Center and Radius bound the covered area; shadows fade out at the edge.
	Center   V3
	Radius   float32
	Strength float32
	// Bias is the depth comparison offset in normalized light depth.
	Bias float32
}

func NewSunShadow(size int) (*SunShadow, error) {
	if size < 256 || size > 8192 {
		return nil, fmt.Errorf("sun shadow: invalid size %d", size)
	}
	t := &SunShadow{size: int32(size), Strength: .45}
	gl.CreateTextures(gl.TEXTURE_2D, 1, &t.depth)
	gl.TextureStorage2D(t.depth, 1, gl.DEPTH_COMPONENT32F, t.size, t.size)
	gl.TextureParameteri(t.depth, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TextureParameteri(t.depth, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TextureParameteri(t.depth, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
	gl.TextureParameteri(t.depth, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)
	border := [4]float32{1, 1, 1, 1}
	gl.TextureParameterfv(t.depth, gl.TEXTURE_BORDER_COLOR, &border[0])
	gl.TextureParameteri(t.depth, gl.TEXTURE_COMPARE_MODE, gl.COMPARE_REF_TO_TEXTURE)
	gl.TextureParameteri(t.depth, gl.TEXTURE_COMPARE_FUNC, gl.LEQUAL)
	gl.CreateFramebuffers(1, &t.fbo)
	gl.NamedFramebufferTexture(t.fbo, gl.DEPTH_ATTACHMENT, t.depth, 0)
	gl.NamedFramebufferDrawBuffer(t.fbo, gl.NONE)
	gl.NamedFramebufferReadBuffer(t.fbo, gl.NONE)
	if status := gl.CheckNamedFramebufferStatus(t.fbo, gl.FRAMEBUFFER); status != gl.FRAMEBUFFER_COMPLETE {
		t.Delete()
		return nil, fmt.Errorf("sun shadow: framebuffer %#x", status)
	}
	t.bytes = uint64(size) * uint64(size) * 4
	addGPUBytes(t.bytes)
	return t, nil
}

// Fit aims the light along lightDir (the direction the light travels) at the
// area of the given radius around center, and returns the light view and
// projection used for the caster pass.
func (t *SunShadow) Fit(center, lightDir V3, radius, depthRange float32) (view, projection M4) {
	l := lightDir.Norm()
	up := V3{0, 1, 0}
	if math.Abs(float64(l[1])) > .99 {
		up = V3{0, 0, 1}
	}
	// A view anchored at the world origin keeps the texel grid fixed in space.
	view = LookAtLH(V3{}, l, up)
	c := transformPoint3(center, view)
	texel := 2 * radius / float32(t.size)
	cx := float32(math.Floor(float64(c[0]/texel))) * texel
	cy := float32(math.Floor(float64(c[1]/texel))) * texel
	near, far := c[2]-depthRange, c[2]+depthRange
	projection = orthoLH(cx-radius, cx+radius, cy-radius, cy+radius, near, far)
	t.Matrix = view.Mul(projection)
	t.Center, t.Radius = center, radius
	// About 0.15 world units, expressed in the [0,1] light depth.
	t.Bias = .15 / (far - near)
	return view, projection
}

// Capture renders depth-only casters into the map. It preserves the caller's
// framebuffer, viewport, scissor, masks and clear depth.
func (t *SunShadow) Capture(draw func()) {
	var fbo, read int32
	var viewport, scissor [4]int32
	var colorMask [4]bool
	var depthMask bool
	var clearDepth float64
	gl.GetIntegerv(gl.DRAW_FRAMEBUFFER_BINDING, &fbo)
	gl.GetIntegerv(gl.READ_FRAMEBUFFER_BINDING, &read)
	gl.GetIntegerv(gl.VIEWPORT, &viewport[0])
	gl.GetIntegerv(gl.SCISSOR_BOX, &scissor[0])
	gl.GetBooleanv(gl.COLOR_WRITEMASK, &colorMask[0])
	gl.GetBooleanv(gl.DEPTH_WRITEMASK, &depthMask)
	gl.GetDoublev(gl.DEPTH_CLEAR_VALUE, &clearDepth)
	scissored := gl.IsEnabled(gl.SCISSOR_TEST)
	gl.BindFramebuffer(gl.FRAMEBUFFER, t.fbo)
	gl.Viewport(0, 0, t.size, t.size)
	gl.Disable(gl.SCISSOR_TEST)
	gl.ColorMask(false, false, false, false)
	gl.DepthMask(true)
	gl.ClearDepth(1)
	gl.Clear(gl.DEPTH_BUFFER_BIT)
	// Slope-scaled offset keeps lit surfaces from shadowing themselves.
	gl.Enable(gl.POLYGON_OFFSET_FILL)
	gl.PolygonOffset(2, 4)
	draw()
	gl.Disable(gl.POLYGON_OFFSET_FILL)
	gl.ColorMask(colorMask[0], colorMask[1], colorMask[2], colorMask[3])
	gl.DepthMask(depthMask)
	gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, uint32(fbo))
	gl.BindFramebuffer(gl.READ_FRAMEBUFFER, uint32(read))
	gl.Viewport(viewport[0], viewport[1], viewport[2], viewport[3])
	gl.Scissor(scissor[0], scissor[1], scissor[2], scissor[3])
	if scissored {
		gl.Enable(gl.SCISSOR_TEST)
	}
	gl.ClearDepth(clearDepth)
}

func (t *SunShadow) Delete() {
	if t == nil {
		return
	}
	gl.DeleteFramebuffers(1, &t.fbo)
	gl.DeleteTextures(1, &t.depth)
	removeGPUBytes(t.bytes)
	*t = SunShadow{}
}

// sunUniforms are the receiver uniforms shared by the scene and skin programs.
type sunUniforms struct{ on, matrix, area, params int32 }

func lookupSunUniforms(p uint32) sunUniforms {
	u := func(n string) int32 { return gl.GetUniformLocation(p, gl.Str(n+"\x00")) }
	return sunUniforms{on: u("sunShadowOn"), matrix: u("sunShadowMatrix"), area: u("sunShadowArea"), params: u("sunShadowParams")}
}

// idleShadowTexture is a 1x1 comparison depth texture kept on unit 7 while
// sun shadows are off, so the shadow sampler never sees an incompatible texture.
var idleShadowTexture uint32

func bindIdleShadow() {
	if idleShadowTexture == 0 {
		gl.CreateTextures(gl.TEXTURE_2D, 1, &idleShadowTexture)
		gl.TextureStorage2D(idleShadowTexture, 1, gl.DEPTH_COMPONENT32F, 1, 1)
		gl.TextureParameteri(idleShadowTexture, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
		gl.TextureParameteri(idleShadowTexture, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
		gl.TextureParameteri(idleShadowTexture, gl.TEXTURE_COMPARE_MODE, gl.COMPARE_REF_TO_TEXTURE)
		gl.TextureParameteri(idleShadowTexture, gl.TEXTURE_COMPARE_FUNC, gl.LEQUAL)
	}
	gl.BindTextureUnit(7, idleShadowTexture)
}

func (u sunUniforms) upload(s *Scene3D) {
	t := s.SunShadow
	if t == nil || t.depth == 0 || s.Shadow {
		gl.Uniform1i(u.on, 0)
		bindIdleShadow()
		return
	}
	gl.Uniform1i(u.on, 1)
	gl.UniformMatrix4fv(u.matrix, 1, false, &t.Matrix[0])
	gl.Uniform4f(u.area, t.Center[0], t.Center[1], t.Center[2], t.Radius)
	gl.Uniform4f(u.params, t.Strength, t.Bias, 1/float32(t.size), 0)
	gl.BindTextureUnit(7, t.depth)
}

func transformPoint3(v V3, m M4) V3 {
	return V3{v[0]*m[0] + v[1]*m[4] + v[2]*m[8] + m[12], v[0]*m[1] + v[1]*m[5] + v[2]*m[9] + m[13], v[0]*m[2] + v[1]*m[6] + v[2]*m[10] + m[14]}
}

// orthoLH is the D3D-style orthographic projection with depth in [0,1].
func orthoLH(left, right, bottom, top, near, far float32) M4 {
	return M4{2 / (right - left), 0, 0, 0, 0, 2 / (top - bottom), 0, 0, 0, 0, 1 / (far - near), 0, -(left + right) / (right - left), -(top + bottom) / (top - bottom), -near / (far - near), 1}
}
