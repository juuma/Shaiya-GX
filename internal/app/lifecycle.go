package app

import (
	"log"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
)

func (a *App) Close() {
	a.saveClientSettings()
	a.tooltipFont.Delete()
	a.performance.meter.Delete()
	a.performance.meter = nil
	if a.world != nil && a.state != nil && a.state.Phase == game.PhaseInWorld &&
		a.wv != nil && a.wv.mapID == a.state.MapID && a.wv.moving {
		if err := a.world.Send(a.wv.movePacket(0)); err != nil {
			log.Printf("final movement: %v", err)
		}
	}
	a.releaseWorld()
	if a.login3 != nil {
		a.login3.World.Close()
	}
	for _, w := range a.worlds {
		w.Close()
	}
	a.skinner.Delete()
	a.sunShadow.Delete()
	if a.frameTarget != nil {
		a.frameTarget.Delete()
	}
	if a.sceneGlow != nil {
		a.sceneGlow.Delete()
	}
	a.scene3d.Delete()
	if a.world != nil {
		if err := a.world.Shutdown(10 * time.Second); err != nil {
			log.Printf("world shutdown: %v", err)
		}
		a.world = nil
	}
	if a.sound != nil {
		if err := a.sound.Close(); err != nil {
			log.Printf("audio close: %v", err)
		}
	}
}
