package app

import (
	"log"
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// worldView is the in-game 3D view: map, own character and the native
// third-person camera (0x444b10 reset, 0x4397d0 orbit/zoom, 0x43bfa0
// movement; Phoenix tools/reference/ep5_camera_controls.md).
type worldView struct {
	requestedSkill          *requestedSkillPresentation
	releaseEffects          []skillRelease
	death                   deathView
	localActions            [ui.TotalActionSlots]uint8 // zero means empty; otherwise panel index + 1
	localActionDrag         subActionDrag
	shadow                  *render.ShadowMap
	shadowAttempted         bool
	waterReflection         *render.WaterTarget
	waterWidth, waterHeight int
	waterFailed             bool
	mapView                 mapView
	mobClick                mobClick
	portrait                *render.Texture
	portraitAppearance      scene.Appearance
	portraitLoaded          bool
	dead                    bool
	uiLeft                  int
	uiRight                 bool
	npcs                    map[uint32]*npcActor
	mapID                   uint16
	actors                  map[uint32]*remoteActor
	mobs                    map[uint32]*mobActor
	world                   *scene.World
	self                    *scene.Character
	selfMount               *scene.Vehicle
	selfMountShape          uint8
	selfMountRace           uint8
	selfMountLoaded         bool
	pos                     render.V3
	facing                  render.V3 // logical heading; diagonals must not feed back into this
	visualFacing            render.V3 // rendered diagonal heading, zero means logical heading
	combatHeading           render.V3 // heading to restore after facing an attack target
	eye                     render.V3 // desired eye; follows the position delta
	moving                  bool
	walking                 bool
	autoRun                 bool
	travel                  bool
	approachTarget          uint32
	approachSkill           []byte
	approachLearned         proto.LearnedSkill
	approachLoot            bool
	approachNPC             bool
	destination             render.V3
	travelEffect            *scene.Effect
	firstPerson             bool
	sittingCamera           bool
	mouselook               bool
	cameraReady             bool
	cameraEye, cameraTarget render.V3
	firstDirection          render.V3
	jumping                 bool
	jumpFall                float32
	jumpFloor               float32
	controlTime             float64
	forwardRelease          float64
	forwardReleased         bool
	backRelease             float64
	backReleased            bool
	evading                 bool
	evadeAction             int
	evadeDirection          render.V3
	evadeTime               float32
	sideRelease             [2]float64
	sideReleased            [2]bool
	sitPending              bool
	sitDeadline             float64
	sitAttack               uint32
	selfGestureSerial       uint64
	waterDirty              bool
	waterFallSpeed          float32
	sendT                   float64
	motion                  uint8
	reviveAt                float64
	effect                  *scene.Effect
	castEffects             []castEffect
	projectiles             []skillProjectile
	impacts                 []skillImpact
	impactCues              []skillImpactCue
	attackSounds            []attackSound
	effectErrors            map[string]bool
	groundBatches           [4][]render.M4
	groundColors            [4][][4]float32
	groundStart             map[uint32]float64
}

const (
	runSpeed   = 5.0 // native normal speed base
	zoomMin    = 2.1 // distance to the character origin
	zoomMax    = 13.0
	camArrowYw = 150.0 * math.Pi / 180
	camArrowPt = 80.0 * math.Pi / 180
	mouseScale = 0.1 * math.Pi / 180 // per pixel at sensitivity 3
)

func (a *App) enterWorld() {
	a.releaseWorld()
	st := a.state
	v := &worldView{mapID: st.MapID, pos: render.V3{st.Self.X, st.Self.Y, st.Self.Z}}
	w, err := scene.LoadMap(a.assets, st.MapID)
	if err != nil {
		log.Printf("map %d: %v", st.MapID, err)
	}
	v.world = w
	// Facing from the 0x0105 direction (u16 angle); default +Z.
	ang := float64(st.Self.Dir) / 65536 * 2 * math.Pi
	v.facing = render.V3{float32(math.Sin(ang)), 0, float32(math.Cos(ang))}
	a.refreshSelf(v)
	for f := range a.assets.Missing {
		log.Printf("map %d: missing %s", st.MapID, f)
		delete(a.assets.Missing, f)
	}
	v.resetCamera()
	a.wv = v
	a.loadLocalActions()
}

func (v *worldView) target() render.V3 {
	if v.firstPerson {
		return v.eye.Add(v.firstDirection)
	}
	height := float32(.5)
	if v.sittingCamera || v.world.SwimmingAt(v.pos[0], v.pos[1], v.pos[2]) {
		height = -.2
	}
	return v.cameraOrigin().Add(render.V3{0, height, 0})
}

func (v *worldView) resetCamera() {
	v.firstPerson = false
	v.eye = v.cameraOrigin().Sub(v.facing.Scale(5)).Add(render.V3{0, 1.5, 0})
	v.snapRenderedCamera()
}

// rotateEye turns the eye around the target: yaw about world up, pitch
// about view x up; pitch is rejected when the eye-to-origin direction gets
// within 0.95 of the vertical.
func (v *worldView) rotateEye(yaw, pitch float64) {
	t := v.target()
	off := v.eye.Sub(t)
	c, s := float32(math.Cos(yaw)), float32(math.Sin(yaw))
	off = render.V3{off[0]*c - off[2]*s, off[1], off[0]*s + off[2]*c}
	yawed := t.Add(off)
	if pitch != 0 {
		axis := off.Cross(render.V3{0, 1, 0}).Norm()
		rotated := rotateAxis(off, axis, float32(pitch))
		dir := t.Add(rotated).Sub(v.cameraOrigin()).Norm()
		if dir[1] >= .95 || dir[1] <= -.95 {
			// Native 0x4397d0 rejects the complete orbit at a pole.
			return
		}
		off = rotated
	}
	v.eye = t.Add(off)
	if v.world.HasTerrain() {
		v.eye = cameraOrbitFloor(yawed, v.eye, v.world.Height(v.eye[0], v.eye[2])+.5)
	}
}

func rotateAxis(p, k render.V3, ang float32) render.V3 {
	c, s := float32(math.Cos(float64(ang))), float32(math.Sin(float64(ang)))
	return p.Scale(c).Add(k.Cross(p).Scale(s)).Add(k.Scale(k.Dot(p) * (1 - c)))
}

func (a *App) updateWorld(in *platform.Input, dt float64) {
	// Phoenix limits simulation steps to 1/60s and stalled frames to 250ms.
	// Edge events and physical mouse deltas belong to the first step only.
	remaining := min(max(dt, 0), .25)
	controls := *in
	for {
		step := min(remaining, 1.0/60)
		a.updateWorldStep(&controls, step)
		remaining -= step
		if remaining < 1e-8 || a.wv == nil || a.state == nil {
			return
		}
		controls.Chars, controls.Keys, controls.Actions = nil, nil, nil
		controls.Released = ""
		controls.Pressed, controls.Clicked, controls.RightPressed, controls.MiddlePressed = false, false, false, false
		controls.Wheel, controls.MouseDX, controls.MouseDY = 0, 0, 0
		controls.MouseLook = true
	}
}

func (a *App) updateWorldStep(in *platform.Input, dt float64) {
	v := a.wv
	if v == nil {
		return
	}
	v.sittingCamera = a.state.Self.Sitting
	settings := a.clientSettings()
	pointer := *in
	if !settings.UseMouse {
		pointer.RightDown, pointer.RightPressed, pointer.MouseDown, pointer.Wheel = false, false, false, 0
	}
	if pointer.RightDown {
		dx, dy := float64(pointer.MouseX-a.lastMouseX), float64(pointer.MouseY-a.lastMouseY)
		if pointer.MouseLook {
			dx, dy = pointer.MouseDX, pointer.MouseDY
		}
		factor := float64(settings.MouseSensitivity) / 3
		pointer.MouseDX, pointer.MouseDY = dx*factor, dy*factor
		if settings.InvertY {
			pointer.MouseDY = -pointer.MouseDY
		}
		pointer.MouseLook = true
	}
	in = &pointer
	v.mouselook = in.RightDown
	yaw, pitch := v.cameraInput(in, dt, a.lastMouseX, a.lastMouseY)
	a.lastMouseX, a.lastMouseY = in.MouseX, in.MouseY
	v.controlTime += dt
	backEvade := false
	evadeMotion := uint8(0)
	if in.SuppressAutoRun {
		v.forwardReleased = false
		v.backReleased = false
		v.sideReleased = [2]bool{}
	}
	for _, key := range in.Released {
		if key == 'Q' || key == 'E' {
			side := 0
			if key == 'E' {
				side = 1
			}
			if v.sideReleased[side] && v.controlTime-v.sideRelease[side] < .301 {
				evadeMotion = proto.MotionEvadeLeft
				if side == 1 {
					evadeMotion = proto.MotionEvadeRight
				}
				v.sideReleased[side] = false
			} else {
				v.sideRelease[side], v.sideReleased[side] = v.controlTime, true
			}
		}
		if key == 'S' {
			if v.backReleased && v.controlTime-v.backRelease < .301 {
				backEvade = true
				v.backReleased = false
			} else {
				v.backRelease, v.backReleased = v.controlTime, true
			}
		}
		if key == 'W' {
			if v.forwardReleased && v.controlTime-v.forwardRelease < .301 {
				v.autoRun = true
				v.forwardReleased = false
			} else {
				v.forwardRelease, v.forwardReleased = v.controlTime, true
			}
		}
	}
	if in.HasKey(platform.KeyTab) {
		v.walking = !v.walking
	}
	toggleSit := false
	for _, c := range in.Chars {
		if c == 'c' || c == 'C' {
			toggleSit = true
		}
	}
	if a.state.Self.Dead || toggleSit || in.Space || in.Held('W') || in.Held('S') || in.Held('A') || in.Held('D') || in.Held('Q') || in.Held('E') || in.HasKey(platform.KeyNumLock) || in.RightDown && in.MouseDown {
		v.travel = false
	}
	if a.state.Self.Dead || toggleSit || in.Held('S') || backEvade {
		v.autoRun = false
	} else if in.HasKey(platform.KeyNumLock) {
		v.autoRun = !v.autoRun
	}
	intent := characterControls(in, v.autoRun)
	if !a.updateActionApproach(in) {
		return
	}
	if !in.SuppressAutoRun && (intent.forward || intent.back || intent.side != 0 || intent.turn != 0 || in.RightDown) {
		v.restoreMovementHeading()
	}
	var dir render.V3
	back := intent.back && !intent.forward
	strafe := intent.side
	forward := intent.forward
	speed := playerMovementSpeed(a.state.Self.Speed, a.state.Self.Mounted)
	if a.state.Self.Dead || a.state.Self.Mounted {
		v.sitAttack = 0
	}
	swimming := v.world.SwimmingAt(v.pos[0], v.pos[1], v.pos[2])
	speed = swimmingMovementSpeed(speed, swimming)
	remaining := float32(math.MaxFloat32)
	if v.travel && !in.SuppressAutoRun {
		delta := v.destination.Sub(v.pos)
		delta[1] = 0
		remaining = delta.Len()
		if remaining < .005 {
			v.travel = false
		}
		forward = v.travel
	}
	wantsMove := forward || back || strafe != 0 || intent.turn != 0 || in.HasKey(platform.KeySpace)
	if !v.sitPending && !a.state.Self.Dead && !a.state.Self.Mounted && !swimming {
		motion := uint8(0)
		request := false
		if a.state.Self.Sitting && (toggleSit || wantsMove || v.sitAttack != 0) {
			motion = proto.MotionStandUp
			request = true
		} else if !a.state.Self.Sitting && toggleSit && !wantsMove {
			motion = proto.MotionSitDown
			request = true
		}
		if request {
			if !a.requestPosture(motion) {
				return
			}
		}
	}
	blocked := a.state.Self.Sitting || v.sitPending || v.self != nil && v.self.PlayingAction(11)
	if blocked {
		dir = render.V3{}
	} else {
		turn := float64(intent.turn) * 1.8 * dt // Phoenix KEY_ORBIT_SPEED.
		if speed == 0 {
			turn = 0
		}
		yaw += turn
		v.turnFacing(turn)
	}
	if yaw != 0 || pitch != 0 {
		v.orbitCamera(yaw, pitch)
	}
	if in.RightDown && !blocked && speed > 0 && !a.state.Self.Dead {
		// Right drag steers immediately; A/D become strafe keys.
		v.facing = cameraHeading(v.target().Sub(v.eye), v.facing)
	}
	fwd := v.facing
	if fwd.Len() == 0 {
		fwd = cameraHeading(v.target().Sub(v.eye), render.V3{0, 0, 1})
		v.facing = fwd
	}
	right := render.V3{0, 1, 0}.Cross(fwd)
	if !blocked {
		dir = controlDirection(fwd, intent)
		if v.travel && !in.SuppressAutoRun {
			dir = v.destination.Sub(v.pos)
			dir[1] = 0
			remaining = dir.Len()
			if remaining > .005 {
				v.facing = dir.Norm()
			}
		}
	}
	v.visualFacing = render.V3{}
	if v.evading && (v.self == nil || !v.self.PlayingAction(v.evadeAction) || a.state.Self.Dead || blocked || swimming) {
		v.evading = false
	}
	if backEvade {
		evadeMotion = proto.MotionBackstep
	}
	if evadeMotion != 0 && !v.evading && !blocked && !swimming && !v.jumping && speed > 0 && !a.state.Self.Dead && !a.state.Self.Mounted && v.self != nil {
		action, _ := proto.LocomotionAction(evadeMotion)
		v.evading = v.self.PlayAction(action, false)
		if v.evading {
			v.autoRun, v.travel = false, false
			v.evadeAction, v.evadeTime = action, 0
			v.evadeDirection = fwd.Scale(-1)
			if action == 14 {
				v.evadeDirection = right.Scale(-1)
			} else if action == 15 {
				v.evadeDirection = right
			}
			if err := a.world.Send(proto.MotionPacket(evadeMotion)); err != nil {
				a.backToLogin("Unable to send evasive movement.")
				return
			}
		}
	}
	if v.evading {
		dir = v.evadeDirection
		back, forward = v.evadeAction == 13, false
		if v.evadeAction == 14 {
			strafe = -1
		} else if v.evadeAction == 15 {
			strafe = 1
		}
	}
	jumpPressed := in.HasKey(platform.KeySpace)
	if jumpPressed && !blocked && !swimming && !v.jumping && !v.evading && speed > 0 && !a.state.Self.Dead && !a.state.Self.Mounted {
		v.beginJump()
		if err := a.world.Send(proto.MotionPacket(proto.MotionJump)); err != nil {
			a.backToLogin("Unable to send jump.")
			return
		}
	}
	moving := dir.Len() > 0 && speed > 0 && !a.state.Self.Dead
	diving := false
	if moving {
		horizontal := dir.Norm()
		if forward && strafe != 0 || swimming || a.state.Self.Mounted {
			v.visualFacing = horizontal
		}
		dir = horizontal
		if swimming {
			dir = swimDirection(dir, v.target().Sub(v.eye), forward, in.Held('S'))
			diving = dir[1] < -0.02
		}
		if v.evading && v.evadeAction != 13 {
			speed = 3
			if v.evadeTime >= .5 {
				speed = 0
			}
		} else if !forward && strafe != 0 && !swimming {
			speed = backwardMovementSpeed(a.state.Self.Speed, a.state.Self.Mounted)
		} else if back && !swimming {
			if v.evading {
				speed = 2.5
			} else {
				speed = backwardMovementSpeed(a.state.Self.Speed, a.state.Self.Mounted)
			}
		} else if back || v.walking && !swimming {
			speed /= 2
		}
		moving = v.step(dir.Scale(min(speed*float32(dt), remaining)))
	}
	if v.evading {
		v.evadeTime += float32(dt)
	}
	if v.jumping {
		if v.advanceJump(float32(dt)) {
			v.waterDirty = true
		}
	} else if v.settleIntoWater(float32(dt)) {
		v.waterDirty = true
	}
	swimming = v.world.SwimmingAt(v.pos[0], v.pos[1], v.pos[2])
	if swimming && in.Space && speed > 0 && !a.state.Self.Dead && v.riseInWater(float32(dt)) {
		v.waterDirty = true
	} else if !diving && v.floatAtSurface(float32(dt)) {
		v.waterDirty = true
	}
	swimming = v.world.SwimmingAt(v.pos[0], v.pos[1], v.pos[2])
	if v.self != nil {
		if moving || v.jumping {
			v.self.CancelCast()
			cancelGesture(v.self)
			a.stopCastEffects(a.state.SelfID)
		}
		if a.state.Self.Dead && !v.dead {
			v.self.PlayAction(9, true)
		}
		if !a.state.Self.Dead && v.dead {
			v.self.ReleaseAction()
		}
		v.dead = a.state.Self.Dead
		if v.selfGestureSerial != a.state.Self.GestureSerial {
			v.selfGestureSerial = a.state.Self.GestureSerial
			if action, ok := proto.GestureAction(a.state.Self.Gesture); ok && !a.state.Self.Dead && !a.state.Self.Mounted && !moving {
				v.self.PlayAction(action, false)
			}
		}
		v.self.Mounted = a.state.Self.Mounted
		switch {
		case a.state.Self.Mounted:
			v.self.SetMountedClip(moving, scene.VehicleAlternate(a.state.Self.MountShape))
		case v.jumping:
			v.self.SetClip("008_jump")
		case a.state.Self.Sitting:
			v.self.SetClip("012_sit")
		case swimming && moving:
			v.self.SetClip("007_swim")
		case swimming:
			v.self.SetClip("006_swnormal")
		case moving && back:
			v.self.SetClip("003_bstep")
		case moving && !forward && strafe < 0:
			v.self.SetClip("004_lstep")
		case moving && !forward && strafe > 0:
			v.self.SetClip("005_rstep")
		case moving && v.walking:
			v.self.SetClip("001_walk")
		case moving:
			v.self.SetClip("002_run")
		default:
			v.self.SetClip("000_normal")
		}
		v.self.Update(float32(dt))
	}
	if v.sitAttack != 0 && !a.state.Self.Sitting && !v.sitPending && (v.self == nil || !v.self.PlayingAction(11)) {
		target := v.sitAttack
		v.sitAttack = 0
		if !wantsMove && a.state.Target == target && !a.attackTarget() {
			return
		}
	}
	// 0x0501: on start/stop, and every 0.5 s while moving (NO VERIFICADO:
	// the original's send cadence).
	v.sendT -= dt
	motion := uint8(0)
	if moving {
		switch {
		case back:
			motion = 2
		case !forward && strafe < 0:
			motion = 3
		case !forward && strafe > 0:
			motion = 4
		case !v.walking || swimming:
			motion = 1
		}
	}
	if v.jumping {
		motion = 4 | proto.MotionFlag80
	}
	v.constrainCamera()
	v.updateRenderedCamera(float32(dt))
	if moving != v.moving || motion != v.motion || ((moving || v.jumping) && v.sendT <= 0) || (v.waterDirty && (v.sendT <= 0 || (swimming && v.pos[1] >= scene.SwimSurfaceY))) {
		if err := a.world.Send(v.movePacket(motion)); err != nil {
			a.backToLogin("Unable to send movement.")
			return
		}
		v.sendT, v.moving, v.motion = 0.5, moving, motion
		v.waterDirty = false
	}
	a.state.Self.X, a.state.Self.Y, a.state.Self.Z = v.pos[0], v.pos[1], v.pos[2]
}

func (a *App) drawWorldView(w, h int) {
	v := a.wv
	if v == nil || v.world == nil || w <= 0 || h <= 0 {
		return
	}
	eye, target := v.eye, v.target()
	if v.cameraReady {
		eye, target = v.cameraEye, v.cameraTarget
	}
	s := a.scene3d
	s.StaticObjectLOD = true
	s.ViewportW, s.ViewportH = w, h
	s.HideWeaponEffects = !a.clientSettings().SkillEffects
	s.DetailBias = float32(2 - a.clientSettings().Details)
	s.SetCameraView(eye, target, render.V3{0, 1, 0})
	aspect := projectionAspect(w, h)
	distanceScale := []float32{.5, .75, 1}[a.clientSettings().VisualDistance]
	far := max(float32(200), max(float32(400), v.world.FogEnd)*distanceScale)
	// Enclose every frustum corner, including wide viewports and pitched cameras.
	tangent := float32(math.Tan(nativeWorldFOV / 2))
	radius := far * float32(math.Sqrt(float64(1+tangent*tangent*(1+aspect*aspect))))
	if a.world != nil {
		v.world.EnableAsyncStreaming()
	}
	v.world.UpdateVisible(eye, radius)
	s.Proj = render.PerspectiveFovLH(nativeWorldFOV, aspect, 0.1, far)
	s.SkyProj = render.PerspectiveFovLH(nativeWorldFOV, aspect, 10, 100000)
	v.world.PrepareTerrainOcclusion(s, eye)
	s.FogColor, s.FogStart, s.FogEnd = v.world.FogColor, v.world.FogStart*distanceScale, v.world.FogEnd*distanceScale
	underwaterColor, underwater := v.world.UnderwaterFogAt(eye)
	if underwater {
		// Phoenix's underwater range; the original's fog curve is not yet known.
		s.FogColor, s.FogStart, s.FogEnd = underwaterColor, 5, 65
		render.ClearBackground(underwaterColor)
	}
	s.LightDir = render.V3{-1, -1, -1}
	s.Begin()
	v.world.Update(float32(a.frameDT))
	a.prepareSelfForDraw()
	a.prepareRemotePlayers()
	a.prepareMonsters()
	a.prepareNPCs()
	a.updateCastEffects()
	a.updateTravelEffect(float32(a.frameDT))
	a.updateResolvedEffects()
	if v.effect != nil {
		v.effect.Update(float32(a.frameDT))
	}
	s.SunShadow = nil
	if a.clientSettings().Shadows && v.world.HasTerrain() && !underwater {
		a.renderSunShadow()
	}
	s.WaterReflection = nil
	waterVisible := a.clientSettings().Water && (underwater || eye[1] > 0) && v.world.WaterVisible(s.View, s.Proj)
	if waterVisible {
		width, height := max(1, w/2), max(1, h/2)
		if v.waterReflection != nil && (v.waterWidth != width || v.waterHeight != height) {
			v.waterReflection.Delete()
			v.waterReflection, v.waterFailed = nil, false
		}
		if v.waterReflection == nil && !v.waterFailed {
			var err error
			v.waterReflection, err = render.NewWaterTarget(width, height)
			v.waterWidth, v.waterHeight = width, height
			if err != nil {
				v.waterFailed = true
				log.Printf("water reflection: %v", err)
			}
		}
	}
	if v.waterReflection != nil && waterVisible && !underwater {
		view := s.View
		mirror := render.Identity()
		mirror[5] = -1
		reflectedView := mirror.Mul(view)
		reflectedEye := eye
		reflectedEye[1] = -reflectedEye[1]
		v.world.PrepareMapEffects(s, eye, scene.EffectCamera{Eye: reflectedEye, View: reflectedView, Projection: s.Proj})
		v.waterReflection.Capture(s.FogColor, func() {
			s.View, s.ClipPlane = reflectedView, [4]float32{0, 1, 0, 0}
			s.Begin()
			a.drawWorldContents(reflectedEye, false)
			v.world.RenderMapEffects(s)
		})
		s.View, s.ClipPlane = view, [4]float32{}
		s.Begin()
		s.WaterReflection = v.waterReflection
	} else if v.waterReflection != nil && waterVisible && underwater {
		v.world.PrepareMapEffects(s, eye)
		fogColor, fogStart, fogEnd := s.FogColor, s.FogStart, s.FogEnd
		s.FogColor, s.FogStart, s.FogEnd = v.world.FogColor, v.world.FogStart*distanceScale, v.world.FogEnd*distanceScale
		v.waterReflection.Capture(s.FogColor, func() {
			s.ClipPlane = [4]float32{0, 1, 0, 0}
			s.Begin()
			a.drawWorldContents(eye, false)
			v.world.RenderMapEffects(s)
		})
		s.ClipPlane = [4]float32{}
		s.FogColor, s.FogStart, s.FogEnd = fogColor, fogStart, fogEnd
		s.Begin()
		s.WaterReflection, s.WaterBelow = v.waterReflection, true
	} else {
		v.world.PrepareMapEffects(s, eye)
	}
	a.drawWorldContents(eye, underwater)
	if s.WaterReflection != nil && !s.WaterBelow && v.world.HasWaterNormals() {
		s.CaptureWaterBackground()
	}
	// Water Surface Effect controls reflection/refraction, not water visibility.
	v.world.DrawWater(s)
	v.world.RenderMapEffects(s)
	s.WaterReflection = nil
	s.WaterBelow = false
	s.SunShadow = nil
	s.End()
}

func (a *App) drawWorldContents(eye render.V3, underwater bool) {
	v, s := a.wv, a.scene3d
	if !underwater && v.world.Sky != nil {
		v.world.Sky.Draw(s, eye)
	}
	v.world.Draw(s)
	a.drawGroundItems()
	a.drawSelf()
	a.drawRemotePlayers()
	a.drawMonsters()
	a.drawNPCs()
	a.drawCastEffects()
	a.drawResolvedEffects()
	if v.travel && v.travelEffect != nil {
		v.travelEffect.Draw(s)
	}
	if v.effect != nil {
		v.effect.Draw(s)
	}
}

// Prepare the mount and rider exactly once; camera passes can draw this pose
// repeatedly without advancing the mount animation or changing the seat.
func (a *App) prepareSelfForDraw() {
	v := a.wv
	if v.self != nil {
		riderPos := v.pos
		heading := v.facing
		if v.visualFacing.Len() > 0 {
			heading = v.visualFacing
		}
		if a.state.Self.Mounted {
			if slot, ok := a.selfSlot(); ok {
				if !v.selfMountLoaded || v.selfMountShape != a.state.Self.MountShape || v.selfMountRace != slot.Race {
					v.selfMountShape, v.selfMountRace, v.selfMountLoaded = a.state.Self.MountShape, slot.Race, true
					var err error
					v.selfMount, err = a.assets.BuildVehicle(slot.Race, v.selfMountShape)
					if err != nil {
						log.Printf("mount: %v", err)
					}
				}
				if v.selfMount != nil {
					v.selfMount.SetMoving(v.moving)
					v.selfMount.Update(float32(a.frameDT))
					v.selfMount.World = render.Basis(v.pos, heading.Scale(-1), render.V3{0, 1, 0})
					riderPos = v.selfMount.Seat()
				}
			}
		} else {
			v.selfMount, v.selfMountLoaded = nil, false
		}
		v.self.World = render.Basis(riderPos, heading.Scale(-1), render.V3{0, 1, 0})
	}
}

func (a *App) drawSelf() {
	v := a.wv
	if v.self == nil || v.firstPerson && !a.scene3d.Shadow && a.scene3d.ClipPlane == ([4]float32{}) {
		return
	}
	frustum := render.ViewFrustum(a.scene3d.View, a.scene3d.Proj)
	if a.state.Self.Mounted && v.selfMount != nil {
		if v.selfMount.Visible(frustum, a.scene3d.ClipPlane) {
			v.selfMount.Draw(a.skinner, a.scene3d)
		}
		a.drawActorShadow(v.selfMount.Character)
	}
	if v.self.Visible(frustum, a.scene3d.ClipPlane) {
		v.self.Draw(a.skinner, a.scene3d)
	}
	a.drawActorShadow(v.self)
}

var _ = game.PhaseInWorld

// selfSlot combines roster identity with server-confirmed equipment. Both the
// world actor and the inventory preview consume the same appearance source.
func (a *App) selfSlot() (proto.CharacterSlot, bool) {
	for _, slot := range a.state.Slots {
		if slot.ID != a.state.SelfID {
			continue
		}
		if a.state.InventoryReady {
			for i := range slot.EquipType {
				it := a.state.Inventory[[2]uint8{0, uint8(i)}]
				slot.EquipType[i], slot.EquipTypeID[i] = it.Type, it.TypeID
			}
		}
		return slot, true
	}
	return proto.CharacterSlot{}, false
}

// Rebuild equipment only from the confirmed inventory, never the local drag.
func (a *App) refreshSelf(v *worldView) {
	slot, ok := a.selfSlot()
	if !ok {
		return
	}
	ch, err := scene.BuildCharacter(a.assets, a.appearance(&slot), "000_normal")
	if err != nil {
		log.Printf("self character: %v", err)
	}
	if ch != nil && a.state.Self.Dead {
		ch.PlayAction(9, true)
	}
	v.self = ch
}

func (a *App) releaseWorld() {
	a.combatNumbers = nil
	a.appearanceEdit = nil
	a.mall = mallView{}
	if a.mallConfirm != nil {
		a.mallConfirm.Visible = false
	}
	a.finishWindowDrag()
	if a.state == nil || a.state.Phase != game.PhaseInWorld && a.state.Phase != game.PhaseLoading {
		a.subActionsOpen = false
		a.playerContext = playerContext{}
		a.questLog = questLogView{}
		if a.questConfirm != nil {
			a.questConfirm.Visible = false
		}
		a.buffs = buffsView{}
		a.party = partyView{}
		a.guild = guildView{}
		a.social = socialView{}
		if a.socialInviteUI != nil {
			a.socialInviteUI.Visible = false
		}
		if a.socialUI != nil {
			a.socialUI.Focus = false
			a.socialUI.EditingMemo = 0
			a.socialUI.Name = ui.TextField{Max: 20}
		}
		a.guildDuel = guildDuelView{}
		if a.guildDuelMessage != nil {
			a.guildDuelMessage.Visible = false
		}
		if a.guildMessage != nil {
			a.guildMessage.Visible = false
		}
		if a.guildConfirm != nil {
			a.guildConfirm.Visible = false
		}
		if a.guildVoteMessage != nil {
			a.guildVoteMessage.Visible = false
		}
		if a.guildCreateUI != nil {
			a.guildCreateUI.Visible = false
		}
		if a.partyMessage != nil {
			a.partyMessage.Visible = false
		}
	}
	if a.wv != nil {
		a.wv.shadow.Delete()
		a.wv.waterReflection.Delete()
		a.scene3d.ReleaseWaterBackground()
		if a.wv.mapView.image != nil {
			a.wv.mapView.image.Delete()
		}
		if a.wv.portrait != nil {
			a.wv.portrait.Delete()
		}
		a.wv.world.Close()
	}
	a.wv = nil
}

// SceneReady is used by offline captures, not by the player's movement/input.
func (a *App) SceneReady() bool {
	if a.screen == screenWorld {
		return a.wv != nil && a.wv.world != nil && a.wv.world.Ready()
	}
	return true
}
