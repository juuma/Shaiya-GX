package render

import (
	"github.com/go-gl/gl/v4.5-core/gl"
)

// Programs of the 3D passes. Colours stay in the texture's encoded space
// (no sRGB conversion), like the D3D9 fixed function.
const sceneVS = `#version 450 core
layout(location=0) in vec3 pos;
layout(location=1) in vec3 nrm;
layout(location=2) in vec2 uv;
layout(location=3) in vec2 uv2;
layout(location=4) in mat4 instanceWorld;
layout(location=8) in mat4 instanceWorldView;
layout(location=12) in mat4 instanceWorldViewProj;
layout(std430, binding=3) readonly buffer EffectColors { vec4 colors[]; };
uniform int instanced;
uniform int effectColorsEnabled;
uniform mat4 world;
uniform mat4 worldViewProj;
uniform mat4 worldView;
uniform vec4 clipPlane;
uniform vec3 lightDir;   // raw direction the light travels
uniform float ambient;
uniform int lit;
out vec2 vUV;
out vec2 vUV2;
out float vDiffuse;
out float vDepth;
out vec4 vEffectColor;
out vec3 vWorldPos;
void main() {
	vec4 local = vec4(pos, 1.0);
	mat4 wm = instanced != 0 ? instanceWorld : world;
	mat4 mv = instanced != 0 ? instanceWorldView : worldView;
	mat4 mvp = instanced != 0 ? instanceWorldViewProj : worldViewProj;
	gl_Position = mvp * local;
	vec4 worldPos = wm * local;
	vWorldPos = worldPos.xyz;
	gl_ClipDistance[0] = dot(worldPos, clipPlane);
	vDepth = (mv * local).z;
	vUV = uv; vUV2 = uv2;
	vEffectColor = effectColorsEnabled != 0 ? colors[gl_InstanceID] : vec4(1.0);
	vec3 n = normalize((wm * vec4(nrm, 0.0)).xyz);
	vDiffuse = lit == 1 ? clamp(ambient + max(0.0, dot(n, -normalize(lightDir))), 0.0, 1.0) : 1.0;
}`

// mode 0: texture * diffuse (models)
// mode 1: terrain layer: rgb = tile (uv2 wrap), alpha = mask (uv clamp)
// mode 2: lightmap multiply pass (uv)
const sceneFS = `#version 450 core
layout(binding=0) uniform sampler2D tex0;
layout(binding=1) uniform sampler2D tex1;
layout(binding=2) uniform sampler2D terrainLight;
layout(binding=3) uniform sampler2D waterReflection;
layout(binding=4) uniform sampler2D waterBackground;
layout(binding=5) uniform sampler2D waterNormalA;
layout(binding=6) uniform sampler2D waterNormalB;
uniform int hasTerrainLight;
uniform int mode;
uniform float alphaRef;
uniform vec3 fogColor;
uniform float detailBias;
uniform vec2 fogRange; // start, end; end <= start disables
uniform float tileSize; // terrain layer tiling (world units per repeat)
uniform vec4 tint;      // mode 3 colour (sky, clouds)
uniform vec2 uvOffset;
uniform int hasWaterReflection;
uniform vec4 waterViewport;
uniform int hasWaterNormals;
uniform int waterBelow;
uniform float waterTime;
layout(binding=7) uniform sampler2DShadow sunShadow;
uniform int sunShadowOn;
uniform mat4 sunShadowMatrix;
uniform vec4 sunShadowArea;   // centre xyz, radius
uniform vec4 sunShadowParams; // strength, bias, texel size
in vec2 vUV;
in vec2 vUV2;
in float vDiffuse;
in float vDepth;
in vec4 vEffectColor;
in vec3 vWorldPos;
out vec4 frag;
// Real-time sun shadow: 3x3 hardware PCF, faded out at the covered area's edge.
float sunLight() {
	if (sunShadowOn == 0) return 1.0;
	vec4 p = sunShadowMatrix * vec4(vWorldPos, 1.0);
	vec3 uvz = vec3(p.xy * 0.5 + 0.5, p.z - sunShadowParams.y);
	if (uvz.z >= 1.0) return 1.0;
	float lit = 0.0;
	for (int y = -1; y <= 1; y++)
		for (int x = -1; x <= 1; x++)
			lit += texture(sunShadow, vec3(uvz.xy + vec2(x, y) * sunShadowParams.z, uvz.z));
	float shadow = 1.0 - lit / 9.0;
	float edge = length(vWorldPos.xz - sunShadowArea.xz) / sunShadowArea.w;
	shadow *= 1.0 - smoothstep(0.75, 1.0, edge);
	return 1.0 - sunShadowParams.x * shadow;
}
void main() {
	vec4 c;
	if (mode == 7) {
		float a = alphaRef < 0.0 ? 1.0 : texture(tex0, vUV, detailBias).a;
		if (a <= alphaRef) discard;
		frag = vec4(0, 0, 0, a);
		return;
	} else if (mode == 8) {
		c = texture(tex0, vUV, detailBias) * vEffectColor;
	} else if (mode == 6) {
		c = texture(tex0, vUV2 / tileSize, detailBias) * tint;
		if (hasWaterReflection != 0) {
			vec2 uv = (gl_FragCoord.xy - waterViewport.xy) / waterViewport.zw;
			if (hasWaterNormals != 0) {
				vec2 base = vUV2 * (2000.0 / 4096.0);
				vec2 n = texture(waterNormalA, base*.15 + vec2(0, waterTime*.045)).xy*.65;
				n += texture(waterNormalA, base*.17 + vec2(0, waterTime*.015)).xy*.45;
				n += texture(waterNormalB, base*.50 + vec2(0, waterTime*.090)).xy*.16;
				n += texture(waterNormalB, base*.55 + vec2(0, waterTime*.030)).xy*.14;
				n -= vec2(.7);
				float divisor = max(vDepth, .1);
				vec2 reflectedUV = uv + (n + vec2(0,-.25)) * (-.128 / divisor);
				vec2 refractedUV = uv + n * (.07 / divisor);
				vec3 water;
				if (waterBelow != 0) {
					water = texture(waterReflection, refractedUV).rgb;
				} else {
					vec3 refraction = texture(waterBackground, refractedUV).rgb;
					vec3 reflection = texture(waterReflection, reflectedUV).rgb;
					water = mix(mix(refraction, reflection, .5), vec3(.25,.62,.40), .05);
				}
				c.rgb = mix(c.rgb, water, .5);
			} else {
				c.rgb = mix(c.rgb, texture(waterReflection, uv).rgb, waterBelow != 0 ? .5 : .1);
			}
		}
	} else if (mode == 5) {
		c = tint;
	} else if (mode == 1) {
		c = vec4(texture(tex1, vUV2 / tileSize, detailBias).rgb, texture(tex0, vUV, detailBias).a);
		if (hasTerrainLight != 0) c.rgb *= texture(terrainLight, vUV).rgb;
		c.rgb *= sunLight();
	} else if (mode == 4) {
		c = texture(tex0, vUV, detailBias);
		c.rgb *= texture(tex1, vUV2, detailBias).rgb * sunLight();
	} else if (mode == 3) {
		frag = texture(tex0, vUV + uvOffset) * tint * vEffectColor;
		return;
	} else if (mode == 2) {
		frag = vec4(texture(tex0, vUV, detailBias).rgb, 1.0);
		return;
	} else {
		c = texture(tex0, vUV, detailBias);
		c.rgb *= vDiffuse * sunLight();
	}
	if (c.a <= alphaRef) discard;
	if (fogRange.y > fogRange.x) {
		float f = clamp((fogRange.y - vDepth) / (fogRange.y - fogRange.x), 0.0, 1.0);
		c.rgb = mix(fogColor, c.rgb, f);
	}
	frag = c;
}`

// Scene3D holds the shared 3D program and per-frame camera state.
type Scene3D struct {
	// SunShadow, when set, darkens receivers with the real-time sun map.
	SunShadow *SunShadow
	// SkipTerrain omits terrain passes (sun shadow casters exclude the ground).
	SkipTerrain bool
	sun         sunUniforms

	StaticObjectLOD                                           bool
	ViewportW, ViewportH                                      int
	uniformWorld, uniformView, uniformVP                      M4
	worldUniformsValid                                        bool
	Occlusion                                                 *TerrainOcclusion
	occlusionWorlds                                           []M4
	occlusionColors                                           [][4]float32
	HideWeaponEffects                                         bool
	DetailBias                                                float32
	uDetailBias                                               int32
	ReverseDepth                                              bool
	cameraView                                                M4
	cameraEye                                                 V3
	cameraValid                                               bool
	instanceBuffer                                            uint32
	effectColorBuffer                                         uint32
	effectColorBytes                                          uint64
	instanceBytes                                             uint64
	instanceCount                                             int
	instanceData                                              []M4
	uInstanced                                                int32
	uEffectColors                                             int32
	projection                                                M4 // current main/sky projection, combined before vertex transformation
	uTerrainLight                                             int32
	Shadow                                                    bool // native black silhouette capture, preserving texture alpha
	effectStream                                              *Mesh
	prog                                                      uint32
	uWorld, uViewProj, uView, uLight, uAmb, uLit, uMode, uRef int32
	uFogColor, uFogRange, uTile, uTint, uUVOff                int32
	uClipPlane                                                int32
	uWaterReflection, uWaterViewport                          int32
	uWaterNormals, uWaterTime                                 int32
	uWaterBelow                                               int32
	terrainIndirect                                           uint32
	terrainIndirectBytes                                      uint64
	terrainCommands                                           []terrainCommand
	View, Proj, SkyProj                                       M4
	LightDir                                                  V3
	Ambient                                                   float32
	FogColor                                                  V3
	FogStart, FogEnd                                          float32
	ClipPlane                                                 [4]float32 // world-space half-plane; zero disables clipping
	WaterReflection                                           *WaterTarget
	WaterNormals                                              [2]*Texture
	WaterTime                                                 float32
	WaterBelow                                                bool
	waterBackground                                           *Texture
}

func NewScene3D() (*Scene3D, error) {
	p, err := Program(sceneVS, sceneFS)
	if err != nil {
		return nil, err
	}
	u := func(n string) int32 { return gl.GetUniformLocation(p, gl.Str(n+"\x00")) }
	return &Scene3D{prog: p, sun: lookupSunUniforms(p), uInstanced: u("instanced"), uEffectColors: u("effectColorsEnabled"), uTerrainLight: u("hasTerrainLight"), uWorld: u("world"), uViewProj: u("worldViewProj"), uView: u("worldView"), uLight: u("lightDir"),
		uDetailBias: u("detailBias"), uAmb: u("ambient"), uLit: u("lit"), uMode: u("mode"), uRef: u("alphaRef"), uFogColor: u("fogColor"), uFogRange: u("fogRange"), uTile: u("tileSize"), uTint: u("tint"), uUVOff: u("uvOffset"), uClipPlane: u("clipPlane"), uWaterReflection: u("hasWaterReflection"), uWaterViewport: u("waterViewport"), uWaterNormals: u("hasWaterNormals"), uWaterTime: u("waterTime"), uWaterBelow: u("waterBelow"),
		LightDir: V3{-1, -1, -1}, Ambient: 128.0 / 255}, nil
}

// Begin sets the frame state: D3D depth range, no culling (EP5 draws
// two-sided), depth test on.
func (s *Scene3D) Begin() {
	s.worldUniformsValid = false
	gl.ClipControl(gl.LOWER_LEFT, gl.ZERO_TO_ONE)
	gl.Enable(gl.DEPTH_TEST)
	s.applyDepthMode()
	gl.DepthMask(true)
	gl.Disable(gl.CULL_FACE)
	gl.UseProgram(s.prog)
	if s.ClipPlane != [4]float32{} {
		gl.Enable(gl.CLIP_DISTANCE0)
	} else {
		gl.Disable(gl.CLIP_DISTANCE0)
	}
	gl.Uniform4fv(s.uClipPlane, 1, &s.ClipPlane[0])
	s.projection = s.depthProjection(s.Proj)
	gl.Uniform3f(s.uLight, s.LightDir[0], s.LightDir[1], s.LightDir[2])
	gl.Uniform1f(s.uAmb, s.Ambient)
	gl.Uniform3f(s.uFogColor, s.FogColor[0], s.FogColor[1], s.FogColor[2])
	gl.Uniform2f(s.uFogRange, s.FogStart, s.FogEnd)
	gl.Uniform1f(s.uDetailBias, s.DetailBias)
	s.sun.upload(s)
}

// End restores the 2D state.
func (s *Scene3D) End() {
	gl.Disable(gl.CLIP_DISTANCE0)
	gl.Disable(gl.DEPTH_TEST)
	gl.DepthMask(true)
	gl.Disable(gl.BLEND)
}

// DrawSolid draws a static coloured surface with the world's depth and fog.
func (s *Scene3D) DrawSolid(m *Mesh, tint [4]float32) {
	id := Identity()
	s.setWorld(id)
	gl.Uniform1i(s.uMode, 5)
	gl.Uniform1i(s.uLit, 0)
	gl.Uniform1f(s.uRef, -1)
	gl.Uniform4f(s.uTint, tint[0], tint[1], tint[2], tint[3])
	s.blend(false)
	m.Draw()
}

// DrawWater is the native WTR flipbook surface: tiled albedo * ARGB,
// alpha blending, fog and read-only scene depth.
func (s *Scene3D) DrawWater(m *Mesh, t *Texture, tileSize float32, tint [4]float32) {
	id := Identity()
	s.setWorld(id)
	gl.Uniform1i(s.uMode, 6)
	gl.Uniform1i(s.uLit, 0)
	gl.Uniform1f(s.uRef, 0)
	gl.Uniform1f(s.uTile, tileSize)
	gl.Uniform4f(s.uTint, tint[0], tint[1], tint[2], tint[3])
	if s.WaterReflection != nil && s.WaterReflection.color != 0 {
		var viewport [4]int32
		gl.GetIntegerv(gl.VIEWPORT, &viewport[0])
		gl.Uniform1i(s.uWaterReflection, 1)
		below := int32(0)
		if s.WaterBelow {
			below = 1
		}
		gl.Uniform1i(s.uWaterBelow, below)
		gl.Uniform4f(s.uWaterViewport, float32(viewport[0]), float32(viewport[1]), float32(viewport[2]), float32(viewport[3]))
		gl.BindTextureUnit(3, s.WaterReflection.color)
		if (s.WaterBelow || s.waterBackground != nil) && s.WaterNormals[0] != nil && s.WaterNormals[1] != nil {
			gl.Uniform1i(s.uWaterNormals, 1)
			gl.Uniform1f(s.uWaterTime, s.WaterTime)
			if !s.WaterBelow {
				gl.BindTextureUnit(4, s.waterBackground.ID)
			}
			gl.BindTextureUnit(5, s.WaterNormals[0].ID)
			gl.BindTextureUnit(6, s.WaterNormals[1].ID)
		} else {
			gl.Uniform1i(s.uWaterNormals, 0)
		}
	} else {
		gl.Uniform1i(s.uWaterReflection, 0)
	}
	gl.BindTextureUnit(0, t.ID)
	s.blend(true)
	gl.DepthMask(false)
	m.Draw()
	gl.DepthMask(true)
	gl.Disable(gl.BLEND)
}

// DrawModel draws a mesh with texture * clamped Gouraud diffuse.
func (s *Scene3D) DrawModel(m *Mesh, t *Texture, world M4, alphaRef float32, blend bool) {
	if s.meshOccluded(m, world) {
		return
	}
	s.setWorld(world)
	gl.Uniform1i(s.uMode, 0)
	if s.Shadow {
		gl.Uniform1i(s.uMode, 7)
		blend = false
	}
	gl.Uniform1i(s.uLit, 1)
	gl.Uniform1f(s.uRef, alphaRef)
	s.blend(blend)
	bindDiffuse(t)
	m.Draw()
}

// DrawTerrainLayer draws one layer pass: first opaque, later alpha-blended.
func (s *Scene3D) DrawTerrainLayer(m *Mesh, mask, tile, lightmap *Texture, tileSize float32, first bool) {
	if s.SkipTerrain {
		return
	}
	s.terrainLayerState(mask, tile, lightmap, tileSize, first)
	m.Draw()
}

func (s *Scene3D) terrainLayerState(mask, tile, lightmap *Texture, tileSize float32, first bool) {
	gl.Uniform1f(s.uTile, tileSize)
	id := Identity()
	s.setWorld(id)
	gl.Uniform1i(s.uMode, 1)
	gl.Uniform1i(s.uLit, 0)
	if first {
		gl.Uniform1f(s.uRef, -1)
		s.blend(false)
	} else {
		gl.Uniform1f(s.uRef, 7.5/255)
		s.blend(true)
	}
	gl.BindTextureUnit(0, mask.ID)
	gl.BindTextureUnit(1, tile.ID)
	gl.Uniform1i(s.uTerrainLight, 0)
	if lightmap != nil {
		gl.BindTextureUnit(2, lightmap.ID)
		gl.Uniform1i(s.uTerrainLight, 1)
	}
}

// DrawLightmap multiplies the frame by the terrain lightmap.
func (s *Scene3D) DrawLightmap(m *Mesh, lm *Texture) {
	if s.SkipTerrain {
		return
	}
	id := Identity()
	s.setWorld(id)
	gl.Uniform1i(s.uMode, 2)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.DST_COLOR, gl.ZERO)
	gl.BindTextureUnit(0, lm.ID)
	m.Draw()
	gl.Disable(gl.BLEND)
}

func (s *Scene3D) blend(on bool) {
	if on {
		gl.Enable(gl.BLEND)
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	} else {
		gl.Disable(gl.BLEND)
	}
}

// SetWrap sets a texture's addressing (terrain tiles repeat).
func (t *Texture) SetWrap(repeat bool) {
	mode := int32(gl.CLAMP_TO_EDGE)
	if repeat {
		mode = gl.REPEAT
	}
	gl.TextureParameteri(t.ID, gl.TEXTURE_WRAP_S, mode)
	gl.TextureParameteri(t.ID, gl.TEXTURE_WRAP_T, mode)
}

// BeginSky switches to the sky projection (near 10, far 100000) without
// depth; EndSky restores the scene projection and depth.
func (s *Scene3D) BeginSky() {
	gl.Disable(gl.CLIP_DISTANCE0)
	s.projection = s.depthProjection(s.SkyProj)
	gl.Disable(gl.DEPTH_TEST)
	gl.DepthMask(false)
}

func (s *Scene3D) EndSky() {
	if s.ClipPlane != [4]float32{} {
		gl.Enable(gl.CLIP_DISTANCE0)
	}
	s.projection = s.depthProjection(s.Proj)
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthMask(true)
	gl.Disable(gl.BLEND)
}

// DrawUnlit draws texture * (1,1,1,alpha) with alpha blending.
func (s *Scene3D) DrawUnlit(m *Mesh, t *Texture, world M4, alpha float32, uvOff [2]float32) {
	s.setWorld(world)
	gl.Uniform1i(s.uMode, 3)
	gl.Uniform4f(s.uTint, 1, 1, 1, alpha)
	gl.Uniform2f(s.uUVOff, uvOff[0], uvOff[1])
	s.blend(true)
	bindDiffuse(t)
	m.Draw()
}

// DrawLightmapped draws DG geometry: texture * lightmap (clamp), unlit.
func (s *Scene3D) DrawLightmapped(m *Mesh, t, lm *Texture) {
	if s.meshOccluded(m, Identity()) {
		return
	}
	id := Identity()
	s.setWorld(id)
	gl.Uniform1i(s.uMode, 4)
	gl.Uniform1f(s.uRef, NativeWorldAlphaRef(true))
	s.blend(false)
	bindDiffuse(t)
	gl.BindTextureUnit(1, lm.ID)
	m.Draw()
}

// DrawTinted draws texture*tint unlit. blend: alpha blending; additive:
// SRCALPHA/ONE. Blended draws do not write depth.
func (s *Scene3D) DrawTinted(m *Mesh, t *Texture, world M4, tint [4]float32, blend, additive bool) {
	if t == nil {
		return
	}
	s.setWorld(world)
	gl.Uniform1i(s.uMode, 3)
	gl.Uniform4f(s.uTint, tint[0], tint[1], tint[2], tint[3])
	gl.Uniform2f(s.uUVOff, 0, 0)
	if blend {
		gl.Enable(gl.BLEND)
		if additive {
			gl.BlendFunc(gl.SRC_ALPHA, gl.ONE)
		} else {
			gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
		}
		gl.DepthMask(false)
	} else {
		gl.Disable(gl.BLEND)
	}
	gl.BindTextureUnit(0, t.ID)
	m.Draw()
	gl.DepthMask(true)
	gl.Disable(gl.BLEND)
}

// Combine model and camera transforms before the shader adds small local
// positions. This avoids float cancellation at map coordinates, as Godot's
// MODELVIEW_MATRIX, while retaining the native LH camera and projection.
func (s *Scene3D) setWorld(world M4) {
	gl.Uniform1i(s.uInstanced, 0)
	mv := s.modelView(world)
	mvp := mv.Mul(s.projection)
	if s.worldUniformsValid && s.uniformWorld == world && s.uniformView == mv && s.uniformVP == mvp {
		return
	}
	s.uniformWorld, s.uniformView, s.uniformVP = world, mv, mvp
	s.worldUniformsValid = true
	gl.UniformMatrix4fv(s.uWorld, 1, false, &s.uniformWorld[0])
	gl.UniformMatrix4fv(s.uView, 1, false, &s.uniformView[0])
	gl.UniformMatrix4fv(s.uViewProj, 1, false, &s.uniformVP[0])
}

// SetCameraView retains the eye independently of the rounded view translation.
func (s *Scene3D) SetCameraView(eye, target, up V3) {
	s.View = LookAtLH(eye, target, up)
	s.cameraView, s.cameraEye, s.cameraValid = s.View, eye, true
}

func (s *Scene3D) modelView(world M4) M4 {
	mv := world.Mul(s.View)
	if s.cameraValid && s.View == s.cameraView {
		// Subtract large positions before rotating, instead of cancelling two
		// separately rounded world-space dot products.
		for j := 0; j < 3; j++ {
			var t float64
			for i := 0; i < 3; i++ {
				t += (float64(world[12+i]) - float64(s.cameraEye[i])) * float64(s.View[i*4+j])
			}
			mv[12+j] = float32(t)
		}
	}
	return mv
}

// Keep CPU projection/frustum coordinates native; invert only the GPU depth.
func (s *Scene3D) depthProjection(p M4) M4 {
	if s.ReverseDepth && !s.Shadow {
		for i := 0; i < 4; i++ {
			p[i*4+2] = p[i*4+3] - p[i*4+2]
		}
	}
	return p
}

func (s *Scene3D) applyDepthMode() {
	if s.ReverseDepth && !s.Shadow {
		gl.DepthFunc(gl.GEQUAL)
		gl.ClearDepth(0)
	} else {
		gl.DepthFunc(gl.LEQUAL)
		gl.ClearDepth(1)
	}
}

// SetReverseDepth scopes depth state without rebinding the drawing program.
func (s *Scene3D) SetReverseDepth(enabled bool) {
	s.ReverseDepth = enabled
	s.applyDepthMode()
}
