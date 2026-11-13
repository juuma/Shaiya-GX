package render

import (
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
)

// SkinVertex is the GPU layout of a skinned vertex: two bone influences
// (EP5 3DC: weight w for bone 0, 1-w for bone 1).
type SkinVertex struct {
	Pos, Normal V3
	UV          [2]float32
	Weight      float32
	Bone0       float32
	Bone1       float32
}

const MaxBones = 256 // EP5 bone indices occupy one byte.

const skinVS = `#version 450 core
layout(location=0) in vec3 pos;
layout(location=1) in vec3 nrm;
layout(location=2) in vec2 uv;
layout(location=3) in vec3 skin; // weight, bone0, bone1
layout(std140, binding=0) uniform BonePalette { mat4 bones[256]; };
uniform int boneCount;
uniform mat4 world;
uniform mat4 worldViewProj;
uniform mat4 worldView;
uniform vec4 clipPlane;
uniform vec3 lightDir;
uniform float ambient;
out vec2 vUV;
out vec2 vUV2;
out float vDiffuse;
out float vDepth;
out vec4 vEffectColor;
out vec3 vWorldPos;
void main() {
	vEffectColor = vec4(1.0);
	vec4 p = vec4(pos, 1.0);
	vec3 n = nrm;
	if (boneCount > 0) {
		p = vec4(0.0); n = vec3(0.0);
		if (skin.x > 0.0) {
			mat4 b = bones[clamp(int(skin.y), 0, boneCount-1)];
			p += skin.x * (b * vec4(pos, 1.0));
			n += skin.x * (b * vec4(nrm, 0.0)).xyz;
		}
		if (skin.x < 1.0) {
			mat4 b = bones[clamp(int(skin.z), 0, boneCount-1)];
			p += (1.0-skin.x) * (b * vec4(pos, 1.0));
			n += (1.0-skin.x) * (b * vec4(nrm, 0.0)).xyz;
		}
	}
	vec4 local = vec4(p.xyz, 1.0);
	gl_Position = worldViewProj * local;
	vec4 worldPos = world * local;
	vWorldPos = worldPos.xyz;
	gl_ClipDistance[0] = dot(worldPos, clipPlane);
	vDepth = (worldView * local).z;
	vUV = uv; vUV2 = vec2(0.0);
	n = normalize((world * vec4(n, 0.0)).xyz);
	vDiffuse = clamp(ambient + max(0.0, dot(n, -normalize(lightDir))), 0.0, 1.0);
}`

// SkinMesh is a skinned triangle list.
type SkinMesh struct {
	vao, vbo, ebo uint32
	Count         int32
	bytes         uint64
}

func NewSkinMesh(v []SkinVertex, idx []uint32) *SkinMesh {
	m := &SkinMesh{Count: int32(len(idx))}
	if len(v) == 0 || len(idx) == 0 {
		return m
	}
	size := int(unsafe.Sizeof(SkinVertex{}))
	gl.CreateBuffers(1, &m.vbo)
	gl.NamedBufferStorage(m.vbo, len(v)*size, unsafe.Pointer(&v[0]), 0)
	gl.CreateBuffers(1, &m.ebo)
	gl.NamedBufferStorage(m.ebo, len(idx)*4, unsafe.Pointer(&idx[0]), 0)
	m.bytes = uint64(len(v)*size + len(idx)*4)
	addGPUBytes(m.bytes)
	gl.CreateVertexArrays(1, &m.vao)
	gl.VertexArrayVertexBuffer(m.vao, 0, m.vbo, 0, int32(size))
	gl.VertexArrayElementBuffer(m.vao, m.ebo)
	for i, a := range []struct {
		n   int32
		off uint32
	}{{3, 0}, {3, 12}, {2, 24}, {3, 32}} {
		gl.EnableVertexArrayAttrib(m.vao, uint32(i))
		gl.VertexArrayAttribFormat(m.vao, uint32(i), a.n, gl.FLOAT, false, a.off)
		gl.VertexArrayAttribBinding(m.vao, uint32(i), 0)
	}
	return m
}

func (m *SkinMesh) Delete() {
	if m == nil {
		return
	}
	gl.DeleteVertexArrays(1, &m.vao)
	gl.DeleteBuffers(1, &m.vbo)
	gl.DeleteBuffers(1, &m.ebo)
	removeGPUBytes(m.bytes)
	*m = SkinMesh{}
}

// Skinner draws skinned meshes with the shared fragment stage.
type Skinner struct {
	uniformWorld, uniformView, uniformVP               M4
	camera                                             *Scene3D
	view, projection                                   M4
	palette                                            uint32
	prog                                               uint32
	uBoneCount, uWorld, uViewProj, uView, uLight, uAmb int32
	uMode, uRef, uFogColor, uFogRange                  int32
	uDetailBias                                        int32
	uClipPlane                                         int32
	sun                                                sunUniforms
}

func NewSkinner() (*Skinner, error) {
	p, err := Program(skinVS, sceneFS)
	if err != nil {
		return nil, err
	}
	u := func(n string) int32 { return gl.GetUniformLocation(p, gl.Str(n+"\x00")) }
	var palette uint32
	gl.CreateBuffers(1, &palette)
	gl.NamedBufferStorage(palette, MaxBones*64, nil, gl.DYNAMIC_STORAGE_BIT)
	addGPUBytes(MaxBones * 64)
	return &Skinner{prog: p, palette: palette, sun: lookupSunUniforms(p), uBoneCount: u("boneCount"), uWorld: u("world"), uViewProj: u("worldViewProj"), uView: u("worldView"),
		uLight: u("lightDir"), uAmb: u("ambient"), uMode: u("mode"), uRef: u("alphaRef"),
		uDetailBias: u("detailBias"), uFogColor: u("fogColor"), uFogRange: u("fogRange"), uClipPlane: u("clipPlane")}, nil
}

// Begin binds the program with the scene's camera, light and fog.
func (k *Skinner) Begin(s *Scene3D, ambient float32) {
	gl.UseProgram(k.prog)
	gl.BindBufferBase(gl.UNIFORM_BUFFER, 0, k.palette)
	gl.Uniform4fv(k.uClipPlane, 1, &s.ClipPlane[0])
	k.view, k.projection = s.View, s.depthProjection(s.Proj)
	k.camera = s
	gl.Uniform3f(k.uLight, s.LightDir[0], s.LightDir[1], s.LightDir[2])
	gl.Uniform1f(k.uAmb, ambient)
	gl.Uniform3f(k.uFogColor, s.FogColor[0], s.FogColor[1], s.FogColor[2])
	gl.Uniform2f(k.uFogRange, s.FogStart, s.FogEnd)
	gl.Uniform1f(k.uDetailBias, s.DetailBias)
	gl.Uniform1i(k.uMode, 0)
	if s.Shadow {
		gl.Uniform1i(k.uMode, 7)
	}
	k.sun.upload(s)
}

// SetPose uploads the per-bone skin matrices (B*F, row-vector).
func (k *Skinner) SetPose(world M4, bones []M4) {
	mv := k.camera.modelView(world)
	mvp := mv.Mul(k.projection)
	k.uniformWorld, k.uniformView, k.uniformVP = world, mv, mvp
	gl.UniformMatrix4fv(k.uWorld, 1, false, &k.uniformWorld[0])
	gl.UniformMatrix4fv(k.uView, 1, false, &k.uniformView[0])
	gl.UniformMatrix4fv(k.uViewProj, 1, false, &k.uniformVP[0])
	n := min(len(bones), MaxBones)
	gl.Uniform1i(k.uBoneCount, int32(n))
	if n > 0 {
		gl.NamedBufferSubData(k.palette, 0, n*64, unsafe.Pointer(&bones[0][0]))
	}
}

func (k *Skinner) Delete() {
	if k == nil || k.palette == 0 {
		return
	}
	gl.DeleteBuffers(1, &k.palette)
	gl.DeleteProgram(k.prog)
	removeGPUBytes(MaxBones * 64)
	k.palette, k.prog = 0, 0
}

func (k *Skinner) Draw(m *SkinMesh, t *Texture, alphaRef float32) {
	if m.Count == 0 {
		return
	}
	gl.Uniform1f(k.uRef, alphaRef)
	gl.Disable(gl.BLEND)
	bindDiffuse(t)
	gl.BindVertexArray(m.vao)
	gl.DrawElements(gl.TRIANGLES, m.Count, gl.UNSIGNED_INT, nil)
	drawCalls++
}
