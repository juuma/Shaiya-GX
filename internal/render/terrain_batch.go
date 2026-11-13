package render

import (
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
)

// TerrainRange identifies one cell in a shared indexed terrain mesh.
type TerrainRange struct{ First, Count uint32 }

type terrainCommand struct{ Count, Instances, First, BaseVertex, BaseInstance uint32 }

// SetTerrainRanges uploads a compact list of visible cells once, reused by
// each material layer of the same batch.
func (s *Scene3D) SetTerrainRanges(ranges []TerrainRange) {
	if len(ranges) == 0 {
		return
	}
	if s.terrainIndirect == 0 {
		gl.CreateBuffers(1, &s.terrainIndirect)
	}
	if cap(s.terrainCommands) < len(ranges) {
		s.terrainCommands = make([]terrainCommand, len(ranges))
	} else {
		s.terrainCommands = s.terrainCommands[:len(ranges)]
	}
	commands := s.terrainCommands
	for i, r := range ranges {
		commands[i] = terrainCommand{Count: r.Count, Instances: 1, First: r.First}
	}
	bytes := uint64(len(commands) * 20)
	gl.NamedBufferData(s.terrainIndirect, int(bytes), unsafe.Pointer(&commands[0]), gl.STREAM_DRAW)
	removeGPUBytes(s.terrainIndirectBytes)
	s.terrainIndirectBytes = bytes
	addGPUBytes(bytes)
}

func (s *Scene3D) DrawTerrainRanges(m *Mesh, mask, tile, lightmap *Texture, tileSize float32, first bool, count int) {
	if m == nil || m.Count == 0 || count == 0 || s.SkipTerrain {
		return
	}
	s.terrainLayerState(mask, tile, lightmap, tileSize, first)
	gl.BindVertexArray(m.vao)
	gl.BindBuffer(gl.DRAW_INDIRECT_BUFFER, s.terrainIndirect)
	gl.MultiDrawElementsIndirect(gl.TRIANGLES, gl.UNSIGNED_INT, nil, int32(count), 0)
	gl.BindBuffer(gl.DRAW_INDIRECT_BUFFER, 0)
	drawCalls++
}

// DrawTerrainRange is also used to compare grouped rendering against the
// original per-cell traversal.
func (s *Scene3D) DrawTerrainRange(m *Mesh, mask, tile, lightmap *Texture, tileSize float32, first bool, r TerrainRange) {
	if m == nil || r.Count == 0 || s.SkipTerrain {
		return
	}
	s.terrainLayerState(mask, tile, lightmap, tileSize, first)
	gl.BindVertexArray(m.vao)
	gl.DrawElementsWithOffset(gl.TRIANGLES, int32(r.Count), gl.UNSIGNED_INT, uintptr(r.First)*4)
	drawCalls++
}
