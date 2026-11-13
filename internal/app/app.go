// Package app wires the screens, the network session and the game state.
// Everything here runs on the main thread; network calls run in goroutines
// and post their results back through App.post.
package app

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/audio"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/Spelunkern/Shaiya-GX/internal/session"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type Config struct {
	DisplayModes     func() []platform.DisplayMode
	ApplyDisplayMode func(platform.DisplayMode) error
	OverrideDisplay  bool
	RememberLayout   bool
	Volume           float64
	LoginAddr        string
	WorldPort        int
	Version          uint32
}

type screen int

const (
	screenLogin screen = iota
	screenServers
	screenCharacters
	screenWorld
)

type App struct {
	appliedDisplay                      platform.DisplayMode
	displayConfigured                   bool
	social                              socialView
	socialUI                            *ui.SocialPanel
	socialInviteUI                      *ui.Message
	appliedHideCloak, appliedHideHelmet bool
	combatNumbers                       []combatNumber
	buffLayout                          ui.BuffLayout
	gameMenu                            *ui.GameMenu
	menuOpen                            bool
	menuConfirm                         *ui.Message
	menuIntent                          int
	menuPending                         bool
	menuDeadline                        float64
	deathUI                             *ui.DeathDialog
	guildMessage                        *ui.Message
	guildConfirm                        *ui.Message
	guildVoteMessage                    *ui.Message
	guildDuelMessage                    *ui.Message
	guildDuel                           guildDuelView
	guildCreateUI                       *ui.GuildCreateDialog
	guildUI                             *ui.GuildPanel
	guildDirectoryUI                    *ui.GuildDirectory
	guild                               guildView
	playerContext                       playerContext
	playerMenuUI                        *ui.PlayerMenu
	duelUI                              *ui.ExchangePanel
	duelMessage                         *ui.Message
	duel                                duelView
	performance                         performanceHUD
	bank                                warehouseView
	bankUI                              *ui.Warehouse
	blacksmithUI                        *ui.Blacksmith
	gateUI                              *ui.Gatekeeper
	gateConfirm                         *ui.Message
	gemConfirm                          *ui.Message
	party                               partyView
	partyUI                             *ui.PartyPanel
	partyMessage                        *ui.Message
	windowDrag                          windowDrag
	sound                               *audio.Engine
	soundErrors                         map[string]bool
	cfg                                 Config
	arc                                 *archive.Archive
	sprites                             *render.Sprites
	actionBar                           *ui.ActionBar
	inventoryUI                         *ui.Inventory
	statusUI                            *ui.Status
	marketUI                            *ui.Market
	mallUI                              *ui.Mall
	mallConfirm                         *ui.Message
	mall                                mallView
	quantityUI                          *ui.Quantity
	appearanceUI                        *ui.AppearanceEditor
	appearanceEdit                      *appearanceEdit
	skillBook                           *ui.SkillBook
	factionUI                           *ui.FactionScreen
	accountMessage                      *ui.Message
	account                             accountAction
	factionDeadline                     float64
	hud                                 *ui.HUD
	minimap                             *ui.Minimap
	bottom                              *ui.BottomBar
	subActionsUI                        *ui.SubActions
	subActionsOpen                      bool
	font                                *ui.Font
	tooltip                             []itemDetailLine
	tooltipFont                         *ui.Font
	tooltipItem                         bool
	tooltipDescription                  string
	tooltipSockets                      []tooltipSocket
	chatCache                           []game.ChatLine
	chatEntries                         []ui.ChatEntry
	msg                                 text.SysMsg
	white                               *render.Texture
	cache                               map[string]*render.Texture
	assets                              *scene.Assets
	scene3d                             *render.Scene3D
	frameTarget                         *render.WaterTarget
	sceneGlow                           *render.SceneGlow
	sceneGlowFailed                     bool
	frameWidth, frameHeight             int
	worlds                              map[string]*scene.World
	skinner                             *render.Skinner
	sunShadow                           *render.SunShadow
	sunShadowFailed, sunCapturing       bool
	items                               map[sdata.ItemKey]*sdata.Item
	mallCatalog                         []sdata.CashProduct
	blessingMax                         [2]uint32
	blessingLoaded                      bool
	actor                               *scene.Character
	actorID                             uint32
	actorAppearance                     scene.Appearance
	createActor                         *scene.Character
	createAppearance                    scene.Appearance
	createDeadline                      float64
	login3                              *scene.LoginScene
	login3Failed                        bool
	wv                                  *worldView
	lastMouseX                          int
	lastMouseY                          int

	screen   screen
	login    *ui.Login
	servers  *ui.ServerList
	chars    *ui.CharSelect
	create   *ui.Creation
	creating bool

	chatOpen         bool
	chatInput        ui.TextField
	chatUI           *ui.ChatPanel
	chatHistory      []string
	chatHistoryIndex int
	chatDraft        string

	npcDefs           map[sdata.NPCKey]sdata.NPC
	questDefs         []sdata.Quest
	questDataLoaded   bool
	questStartedAt    map[uint16]time.Time
	questSnapshotAt   map[uint16]time.Time
	questUI           *ui.QuestLog
	questConfirm      *ui.Message
	questLog          questLogView
	exchangeUI        *ui.ExchangePanel
	exchangeMoney     *ui.ExchangeMoney
	exchangeInvite    *ui.Message
	exchange          exchangeView
	npcQuestUI        *ui.NPCQuest
	npcQuest          npcQuestView
	shop              shopView
	blacksmith        blacksmithView
	gate              gatekeeperView
	vehicle           vehicleView
	gemLink           gemLinkView
	gemRemove         gemRemoveView
	enchant           enchantView
	compose           composeView
	repair            repairView
	attackVariant     uint8
	combatNotice      string
	combatNoticeUntil float64
	inventory         inventoryView
	status            statusView
	skills            skillsView
	buffs             buffsView
	skillDefs         map[sdata.SkillKey]sdata.Skill

	sess  *session.Login
	world *session.World
	state *game.State

	clock   float64
	frameDT float64

	posted chan func()
	Quit   bool
}

func New(cfg Config, arc *archive.Archive) (*App, error) {
	a := &App{cfg: cfg, arc: arc, cache: map[string]*render.Texture{}, posted: make(chan func(), 64),
		assets: scene.NewAssets(arc), worlds: map[string]*scene.World{}}
	var err error
	if a.scene3d, err = render.NewScene3D(); err != nil {
		return nil, err
	}
	if a.skinner, err = render.NewSkinner(); err != nil {
		return nil, err
	}
	if a.sprites, err = render.NewSprites(8192); err != nil {
		return nil, err
	}
	if a.font, err = ui.LoadFont(13); err != nil {
		return nil, err
	}
	if a.tooltipFont, err = ui.LoadTooltipFont(); err != nil {
		return nil, err
	}
	if b, err := arc.ReadFile("sysmsg-uni.txt"); err == nil {
		a.msg = text.ParseSysMsg(b)
	} else {
		a.msg = text.SysMsg{}
	}
	a.white = render.Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{255, 255, 255, 255}}})
	if a.login, err = ui.NewLogin(a); err != nil {
		return nil, err
	}
	if a.servers, err = ui.NewServerList(a, a.msg, a.white); err != nil {
		return nil, err
	}
	if a.chars, err = ui.NewCharSelect(a, a.msg); err != nil {
		return nil, err
	}
	if a.create, err = ui.NewCharCreate(a, a.msg); err != nil {
		return nil, err
	}
	if a.accountMessage, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.deathUI, err = ui.NewDeathDialog(a); err != nil {
		return nil, err
	}
	if a.hud, err = ui.NewHUD(a); err != nil {
		return nil, err
	}
	if a.minimap, err = ui.NewMinimap(a); err != nil {
		return nil, err
	}
	if a.bottom, err = ui.NewBottomBar(a); err != nil {
		return nil, err
	}
	if a.gameMenu, err = ui.NewGameMenu(a); err != nil {
		return nil, err
	}
	if cfg.DisplayModes != nil {
		a.gameMenu.SetDisplayModes(cfg.DisplayModes())
	}
	if a.menuConfirm, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.subActionsUI, err = ui.NewSubActions(a); err != nil {
		return nil, err
	}
	if a.chatUI, err = ui.NewChatPanel(a); err != nil {
		return nil, err
	}
	if a.playerMenuUI, err = ui.NewPlayerMenu(a); err != nil {
		return nil, err
	}
	if a.actionBar, err = ui.NewActionBar(a); err != nil {
		return nil, err
	}
	if a.inventoryUI, err = ui.NewInventory(a); err != nil {
		return nil, err
	}
	if a.duelUI, err = ui.NewDuelPanel(a); err != nil {
		return nil, err
	}
	if a.duelMessage, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.guildMessage, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.guildConfirm, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.guildVoteMessage, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.guildDuelMessage, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.guildCreateUI, err = ui.NewGuildCreateDialog(a); err != nil {
		return nil, err
	}
	if a.guildUI, err = ui.NewGuildPanel(a); err != nil {
		return nil, err
	}
	if a.socialUI, err = ui.NewSocialPanel(a); err != nil {
		return nil, err
	}
	if a.socialInviteUI, err = ui.NewMessage(a); err != nil {
		return nil, err
	}

	if a.guildDirectoryUI, err = ui.NewGuildDirectory(a); err != nil {
		return nil, err
	}
	if a.exchangeUI, err = ui.NewExchangePanel(a); err != nil {
		return nil, err
	}
	if a.exchangeMoney, err = ui.NewExchangeMoney(a); err != nil {
		return nil, err
	}
	if a.exchangeInvite, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.questUI, err = ui.NewQuestLog(a); err != nil {
		return nil, err
	}
	if a.npcQuestUI, err = ui.NewNPCQuest(a); err != nil {
		return nil, err
	}
	if a.questConfirm, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.gateUI, err = ui.NewGatekeeper(a); err != nil {
		return nil, err
	}
	if a.gateConfirm, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.blacksmithUI, err = ui.NewBlacksmith(a); err != nil {
		return nil, err
	}
	if a.gemConfirm, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.bankUI, err = ui.NewWarehouse(a); err != nil {
		return nil, err
	}
	if a.statusUI, err = ui.NewStatus(a); err != nil {
		return nil, err
	}
	if a.marketUI, err = ui.NewMarket(a); err != nil {
		return nil, err
	}
	if a.mallUI, err = ui.NewMall(a, a.white); err != nil {
		return nil, err
	}
	if a.mallConfirm, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.quantityUI, err = ui.NewQuantity(a); err != nil {
		return nil, err
	}
	if a.appearanceUI, err = ui.NewAppearanceEditor(a); err != nil {
		return nil, err
	}
	if a.partyUI, err = ui.NewPartyPanel(a); err != nil {
		return nil, err
	}
	if a.partyMessage, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.skillBook, err = ui.NewSkillBook(a); err != nil {
		return nil, err
	}
	if a.factionUI, err = ui.NewFaction(a, a.msg); err != nil {
		return nil, err
	}
	if cfg.Volume > 0 {
		a.sound, err = audio.New(arc.ReadFile, cfg.Volume)
		if err != nil {
			log.Printf("audio unavailable: %v", err)
		}
		a.soundErrors = map[string]bool{}
	}
	a.loadSavedID()
	a.loadUILayout()
	a.loadClientSettings()
	return a, nil
}

// Interface implements ui.Textures: data/interface/<name>, cached.
func (a *App) Interface(name string) (*render.Texture, error) {
	key := strings.ToLower(name)
	if t, ok := a.cache[key]; ok {
		return t, nil
	}
	b, err := a.arc.ReadFile("interface/" + name)
	if err != nil {
		return nil, err
	}
	t, err := img.DecodeResource("interface/"+name, b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	tex := render.Upload(t)
	a.cache[key] = tex
	return tex, nil
}

func (a *App) post(f func()) { a.posted <- f }

// Update runs one frame of logic.
func (a *App) Update(in *platform.Input, w, h int, dt float64) {
	started := time.Now()
	defer func() { a.performance.updateCPU = time.Since(started) }()
	a.frameDT = dt
	a.skills.heldActions = 0
	for {
		select {
		case f := <-a.posted:
			f()
			continue
		default:
		}
		break
	}
	a.pumpWorld()
	a.updateAudio()
	switch a.screen {
	case screenLogin:
		switch a.login.Update(in, w, h, dt) {
		case ui.LoginSubmit:
			a.playUI()
			a.submitLogin()
		case ui.LoginQuit:
			a.Quit = true
		}
	case screenServers:
		switch a.servers.Update(in, w, h, dt) {
		case ui.ServerConnect:
			a.playUI()
			a.connectWorld()
		case ui.ServerBack:
			a.playUI()
			a.backToLogin("")
		}
	case screenCharacters, screenWorld:
		a.updateCharacters(in, w, h, dt)
	}
}

func (a *App) Draw(w, h int) {
	if w <= 0 || h <= 0 {
		return
	}
	if a.frameTarget == nil || a.frameWidth != w || a.frameHeight != h {
		if a.frameTarget != nil {
			a.frameTarget.Delete()
		}
		var err error
		a.frameTarget, err = render.NewWaterTarget(w, h)
		if err != nil {
			log.Printf("frame target: %v", err)
			a.drawFrame(w, h)
			return
		}
		a.frameWidth, a.frameHeight = w, h
	}
	previousDepth := a.scene3d.ReverseDepth
	a.scene3d.SetReverseDepth(true)
	defer a.scene3d.SetReverseDepth(previousDepth)
	a.frameTarget.Capture(render.V3{}, func() { a.drawFrame(w, h) })
	a.frameTarget.PresentBrightness(a.clientSettings().Brightness)
}

func (a *App) drawFrame(w, h int) {
	restoreAlpha := render.KeepSceneAlphaOpaque()
	defer restoreAlpha()
	a.sprites.InvalidateNameDepth()
	if a.performance.meter == nil {
		a.performance.meter = render.NewFrameMeter()
	}
	a.performance.meter.Begin()
	a.performance.sample(a.frameDT)
	render.ResetDrawCalls()
	started := time.Now()
	defer func() { a.performance.meter.End(); a.performance.drawCPU = time.Since(started) }()
	if a.screen == screenLogin || a.screen == screenServers {
		if ls := a.loginBackdrop(); ls != nil {
			ls.Update(a.frameDT)
			ls.Draw(a.scene3d, projectionAspect(w, h))
		}
	}
	if a.screen == screenWorld {
		a.drawWorldView(w, h)
	}
	if a.screen == screenCharacters && a.state != nil && a.state.Phase != game.PhaseFaction {
		fury := a.state.Faction.Faction == 1
		if a.creating {
			a.drawCreation(w, h)
		} else {
			a.drawWorld(a.selectWorld(fury), selectEye, selectTarget, w, h)
		}
	}
	if a.scene3d.ReverseDepth && !a.sceneGlowFailed {
		if a.sceneGlow == nil {
			var err error
			a.sceneGlow, err = render.NewSceneGlow()
			if err != nil {
				log.Printf("default scene glow: %v", err)
				a.sceneGlowFailed = true
			}
		}
		if a.sceneGlow != nil {
			a.sceneGlow.Level = a.clientSettings().Glow
			if err := a.sceneGlow.Apply(); err != nil {
				log.Printf("default scene glow: %v", err)
				a.sceneGlowFailed = true
			}
		}
	}
	a.sprites.Begin(w, h)
	switch a.screen {
	case screenLogin:
		a.login.Draw(a.sprites, a.font, w, h)
	case screenServers:
		a.login.DrawLogo(a.sprites, w, h)
		a.servers.Draw(a.sprites, a.font, w, h)
	case screenCharacters, screenWorld:
		a.drawCharacters(w, h)
	}
	if a.accountMessage != nil {
		a.accountMessage.Draw(a.sprites, a.font, w, h)
	}
	if a.screen == screenWorld && a.partyMessage != nil {
		a.partyMessage.Draw(a.sprites, a.font, w, h)
	}
	if a.screen == screenWorld && a.exchangeInvite != nil {
		a.exchangeInvite.Draw(a.sprites, a.font, w, h)
	}
	if a.screen == screenWorld && a.guildMessage != nil {
		a.guildMessage.Draw(a.sprites, a.font, w, h)
	}
	if a.screen == screenWorld && a.guildConfirm != nil {
		a.guildConfirm.Draw(a.sprites, a.font, w, h)
		if a.socialInviteUI != nil {
			a.socialInviteUI.Draw(a.sprites, a.font, w, h)
		}
	}
	if a.screen == screenWorld && a.guildDuelMessage != nil {
		a.guildDuelMessage.Draw(a.sprites, a.font, w, h)
	}
	if a.screen == screenWorld && a.duelMessage != nil {
		a.duelMessage.Draw(a.sprites, a.font, w, h)
	}
	if a.screen == screenWorld && a.guildCreateUI != nil {
		a.guildCreateUI.Draw(a.sprites, a.font, w, h)
	}
	if a.screen == screenWorld && a.guildVoteMessage != nil {
		a.guildVoteMessage.Draw(a.sprites, a.font, w, h)
	}
	if a.screen == screenWorld {
		a.drawMallConfirm(w, h)
	}
	a.drawPerformance(w)
	a.sprites.End()
}

func (a *App) submitLogin() {
	id, pw := a.login.ID.Text, a.login.Password.Text
	if id == "" || pw == "" || a.login.Busy {
		return
	}
	a.login.Busy, a.login.Message = true, "Connecting..."
	a.saveID()
	go func() {
		l, err := session.Dial(a.cfg.LoginAddr, id, pw)
		a.post(func() {
			a.login.Busy = false
			if err != nil {
				a.login.Message = loginMessage(err)
				return
			}
			a.sess = l
			a.login.Message, a.login.Password.Text = "", ""
			a.servers.Servers, a.servers.Selected = l.Servers, -1
			if len(l.Servers) > 0 {
				a.servers.Selected = 0
			}
			a.screen = screenServers
		})
	}()
}

// NO VERIFICADO: the original maps each 0xA102 code to its own message.
func loginMessage(err error) string {
	if le, ok := err.(session.LoginError); ok {
		return fmt.Sprintf("Login failed (code %d).", le.Code)
	}
	return "Unable to connect to the login server."
}

func (a *App) backToLogin(msg string) {
	a.menuOpen, a.menuPending = false, false
	a.buffs = buffsView{}
	a.party = partyView{}
	if a.partyMessage != nil {
		a.partyMessage.Visible = false
	}
	a.releaseWorld()
	if a.world != nil {
		a.world.Close()
		a.world = nil
	}
	a.sess, a.state = nil, nil
	a.creating = false
	a.account = accountAction{}
	if a.accountMessage != nil {
		a.accountMessage.Visible = false
	}
	a.chatOpen = false
	if a.factionUI != nil {
		a.factionUI.Reset()
	}
	a.inventory = inventoryView{}
	a.skills = skillsView{}
	a.status = statusView{}
	if a.statusUI != nil {
		a.statusUI.Clear()
	}
	a.shop = shopView{}
	a.blacksmith = blacksmithView{}
	a.gate = gatekeeperView{}
	a.vehicle = vehicleView{}
	if a.gateConfirm != nil {
		a.gateConfirm.Visible = false
	}
	a.gemLink = gemLinkView{}
	a.gemRemove = gemRemoveView{}
	a.enchant = enchantView{}
	a.compose = composeView{}
	a.repair = repairView{}
	a.bank = warehouseView{}
	a.questLog = questLogView{}
	a.exchange = exchangeView{}
	a.playerContext = playerContext{}
	a.duel = duelView{}
	a.guild = guildView{}
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
	if a.duelMessage != nil {
		a.duelMessage.Visible = false
	}
	if a.exchangeInvite != nil {
		a.exchangeInvite.Visible = false
	}
	a.npcQuest = npcQuestView{}
	if a.questConfirm != nil {
		a.questConfirm.Visible = false
	}
	a.screen = screenLogin
	a.login.Message = msg
	a.login.Focus = 1
}

func (a *App) connectWorld() {
	if a.servers.Busy || a.sess == nil || a.servers.Selected < 0 {
		return
	}
	sv := a.servers.Servers[a.servers.Selected]
	a.servers.Busy = true
	l := a.sess
	go func() {
		ip, err := l.SelectServer(sv.ID, a.cfg.Version)
		var w *session.World
		if err == nil {
			w, err = l.DialWorld(ip, a.cfg.WorldPort)
		}
		a.post(func() {
			a.servers.Busy = false
			if err != nil {
				a.backToLogin("Unable to connect to the game server.")
				return
			}
			a.world, a.state = w, game.New()
			a.screen = screenCharacters
		})
	}()
}

// pumpWorld applies every packet received since the last frame.
func (a *App) pumpWorld() {
	if a.world == nil {
		return
	}
	for {
		select {
		case p, ok := <-a.world.Packets:
			if !ok {
				a.backToLogin("Disconnected from the server.")
				return
			}
			wasRaid := a.state.Party.Raid
			var previousAppearance scene.Appearance
			if proto.Op(p) == proto.ItemBagToBag || proto.Op(p) == proto.ChangeAppearance || proto.Op(p) == proto.EnchantAdd {
				if slot, ok := a.selfSlot(); ok {
					previousAppearance = a.appearance(&slot)
				}
			}
			previousMoney := a.state.Details.Money
			ev, err := a.state.Apply(p)
			if err != nil {
				log.Printf("packet: %v", err)
				continue
			}
			if !wasRaid && a.state.Party.Raid {
				a.party.raidHidden = false
			}
			if ev.Op == proto.Logout && a.state.Phase == game.PhaseCharSelect {
				if a.menuPending && a.menuIntent == ui.MenuExit {
					a.Quit = true
				}
				a.menuOpen, a.menuPending = false, false
				a.releaseWorld()
			}
			if ev.Op == proto.AccountFaction && a.factionUI != nil {
				a.factionUI.Busy = false
			}
			if ev.Op == proto.GuildInvite {
				a.guild.expires = a.clock + 30
			}
			switch ev.Op {
			case proto.GuildListBegin, proto.GuildList, proto.GuildListEnd, proto.GuildNew, proto.GuildDisbanded:
				a.guild.directoryDirty = true
			case proto.GuildOnlineList, proto.GuildOfflineList, proto.GuildMemberAdded, proto.GuildApplicantList, proto.GuildApplicant, proto.GuildApplicantRemoved, proto.GuildStatus, proto.GuildJoinResult, proto.GuildCreate, proto.GuildLeave, proto.GuildExpel:
				a.guild.rosterDirty = true
			}
			a.confirmedDuel(p)
			a.confirmedGuildDuel(p)
			a.confirmedGuildRanking(p)
			if ev.Op == proto.GuildRemark {
				a.guild.remarkPending = false
				if a.state.Guild.RemarkResult == 1 {
					a.guild.directoryDirty = true
					a.questMessage("Guild notice updated.")
				} else {
					a.questMessage("Guild notice was refused.")
				}
			}
			if a.state == nil {
				return
			}
			a.confirmedExchange(p)
			if a.state == nil {
				return
			}
			a.confirmedGemLink(p)
			if a.state == nil {
				return
			}
			a.confirmedGemRemove(p)
			if a.state == nil {
				return
			}
			a.confirmedRepair(p)
			if a.state == nil {
				return
			}
			a.confirmedEnchant(p)
			if a.state == nil {
				return
			}
			a.confirmedCompose(p)
			if a.state == nil {
				return
			}
			a.confirmedGatekeeper(p)
			if a.state == nil {
				return
			}
			a.confirmedVehicle(p)
			if a.state == nil {
				return
			}
			a.confirmedItems(ev.Op)
			a.confirmedSystemChat(p, previousMoney)
			a.confirmedPickup(p)
			a.confirmedMall(p)
			a.confirmedAccount(p)
			a.confirmedQuest(p)
			if a.inventory.selected {
				if _, ok := a.state.Inventory[a.inventory.source]; !ok {
					a.inventory.selected = false
				}
			}
			a.confirmedAnimation(p)
			a.confirmedGesture(ev.Op)
			a.confirmedSkills(ev.Op)
			a.confirmedSkillFeedback(ev.Op)
			if ev.Op == proto.SkillDetectInfo {
				v := a.state.LastDetectInfo
				mode := fmt.Sprintf("%d", v.Mode)
				if int(v.Mode) < 4 {
					mode = [4]string{"Easy", "Normal", "Hard", "Ultimate"}[v.Mode]
				}
				a.questMessage(fmt.Sprintf("Detection: level %d, %s mode, defense element %d.", v.Level, mode, v.DefenseElement))
			}
			a.confirmedBuffs(ev.Op)
			a.confirmedSocial(ev.Op)
			a.confirmedEffects(p)
			a.confirmedSound(p)
			if ev.Op == proto.SetStatPoint || ev.Op == proto.StatsReset {
				a.status.pending = false
				if a.statusUI != nil {
					a.statusUI.Clear()
				}
			}
			if ev.Op == proto.UseItem && proto.NewReader(p).U32() == a.state.SelfID {
				a.inventory.pending = false
				if !a.compose.pending {
					a.inventory.message = "Item used."
					if a.state.LastItemUse.Type == 0 {
						a.inventory.message = "This item cannot be used."
					}
				}
			}
			if ev.Op == proto.LearnNewSkill {
				a.skills.pending = false
				result := a.state.LastLearn
				if result.Result == 0 {
					a.loadSkills()
					cost := uint16(a.skillDefs[sdata.SkillKey{ID: result.ID, Level: result.Level}].Points)
					a.state.Details.SkillPoint -= min(a.state.Details.SkillPoint, cost)
				}
			}
			if ev.Op == proto.ShopBuy || ev.Op == proto.ShopSell {
				a.shop.pending = false
				a.inventory.pending = false
				if ev.Op == proto.ShopBuy && a.state.LastShop.Result == 0 {
					r := a.state.LastShop
					key := [2]uint8{r.Bag, r.Slot}
					it := a.state.Inventory[key]
					if def := a.itemsItem(it.Type, it.TypeID); def != nil {
						it.Quality = def.Quality
						a.state.Inventory[key] = it
					}
				}
			}
			if ev.Op == proto.ItemBagToBag {
				if a.bank.guildPending {
					items, err := proto.DecodeItemMove(p)
					if err == nil && [2]uint8{items[0].Bag, items[0].Slot} == a.bank.sourceMove && [2]uint8{items[1].Bag, items[1].Slot} == a.bank.destMove {
						a.inventory.pending = false
						a.bank.guildPending = false
					}
				} else {
					a.inventory.pending = false
				}
				if slot, ok := a.selfSlot(); ok && a.appearance(&slot) != previousAppearance && a.wv != nil {
					a.refreshSelf(a.wv)
				}
			}
			if (ev.Op == proto.ChangeAppearance || ev.Op == proto.EnchantAdd) && a.wv != nil {
				if slot, ok := a.selfSlot(); ok && a.appearance(&slot) != previousAppearance {
					a.refreshSelf(a.wv)
				}
			}
			if ev.Op == proto.GuildBankResult && a.bank.guildPending {
				a.bank.guildPending = false
				a.inventory.pending = false
				msg := "Guild bank request refused."
				switch a.state.Guild.BankResult.Result {
				case 1:
					msg = "The guild must be ranked to use its bank."
				case 2:
					msg = "Insufficient guild bank permissions or page unavailable."
				case 4:
					msg = "Your guild rank cannot deposit items."
				}
				a.questMessage(msg)
			}
			a.confirmedPosition(p)
			if proto.Op(p) == proto.CharacterSkillBar && a.screen == screenWorld {
				a.reconcileLocalActions()
			}
			if proto.Op(p) == proto.CharacterSkillBar && a.state.Phase == game.PhaseLoading {
				a.world.Send(proto.MapLoadedPacket()) // map "loaded"
			}
		default:
			return
		}
	}
}

// ---- saved ID (LOGIN_ID_SAVE / LOGIN ID of the original) ----

func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "shaiya-go", "login.txt")
}

func (a *App) loadSavedID() {
	b, err := os.ReadFile(configPath())
	if err != nil {
		return
	}
	if id := strings.TrimSpace(string(b)); id != "" {
		a.login.ID.Text, a.login.SaveID.Checked, a.login.Focus = id, true, 1
	}
}

func (a *App) saveID() {
	p := configPath()
	if p == "" {
		return
	}
	if !a.login.SaveID.Checked {
		os.Remove(p)
		return
	}
	os.MkdirAll(filepath.Dir(p), 0o700)
	os.WriteFile(p, []byte(a.login.ID.Text), 0o600)
}
