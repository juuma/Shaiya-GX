package app

import (
	"log"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

// Real-time sun shadows on fields: the map covers sunShadowRadius around the
// player at sunShadowSize texels (about 22 texels per world unit).
const (
	sunShadowSize   = 2048
	sunShadowRadius = 45
	sunShadowDepth  = 150
)

// renderSunShadow draws world objects and actors (not the ground, sky or
// effects) into the sun's depth map, then leaves it set on the scene so the
// following reflection and main passes receive it.
func (a *App) renderSunShadow() {
	if a.sunShadowFailed {
		return
	}
	if a.sunShadow == nil {
		var err error
		if a.sunShadow, err = render.NewSunShadow(sunShadowSize); err != nil {
			log.Printf("sun shadows: %v", err)
			a.sunShadowFailed = true
			return
		}
	}
	v, s := a.wv, a.scene3d
	view, projection := a.sunShadow.Fit(v.pos, s.LightDir, sunShadowRadius, sunShadowDepth)
	saved := *s
	s.View, s.Proj, s.Shadow = view, projection, true
	s.ClipPlane = [4]float32{}
	s.FogStart, s.FogEnd = 0, 0
	s.Occlusion = nil
	s.SkipTerrain = true
	a.sunCapturing = true
	a.sunShadow.Capture(func() {
		s.Begin()
		v.world.Draw(s)
		a.drawSelf()
		a.drawRemotePlayers()
		a.drawMonsters()
		a.drawNPCs()
	})
	a.sunCapturing = false
	s.View, s.Proj, s.Shadow = saved.View, saved.Proj, saved.Shadow
	s.ClipPlane = saved.ClipPlane
	s.FogStart, s.FogEnd = saved.FogStart, saved.FogEnd
	s.Occlusion = saved.Occlusion
	s.SkipTerrain = false
	s.SunShadow = a.sunShadow
	s.Begin()
}

func (a *App) drawActorShadow(c *scene.Character) {
	if !a.clientSettings().Shadows {
		return
	}
	// The sun map already shadows actors; skip the legacy silhouettes then.
	if a.sunCapturing || a.scene3d.SunShadow != nil {
		return
	}
	v := a.wv
	if c == nil || v == nil || v.world == nil || !v.world.HasShadowReceivers() {
		return
	}
	position := render.V3{c.World[12], c.World[13], c.World[14]}
	// Bound capture cost to nearby actors; distant actors keep baked world light.
	if position.Sub(v.eye).Len() > 50 {
		return
	}
	if !v.shadowAttempted {
		v.shadowAttempted = true
		var err error
		v.shadow, err = render.NewShadowMap()
		if err != nil {
			log.Printf("actor shadows: %v", err)
		}
	}
	if v.shadow != nil {
		c.DrawShadow(v.shadow, a.skinner, a.scene3d, v.world)
	}
}
