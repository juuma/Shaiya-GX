# Interface status

The interface is being reconstructed incrementally from original EP5 artwork and recovered parameters. Phoenix is an auxiliary reference. Data and original artwork are not stored in this repository.

| Area | Current implementation | Remaining work |
|---|---|---|
| Account | Login, server list, faction, character selection/creation and account dialogs | Complete visual comparison and remaining variants |
| HUD | Player portrait, resources, mob level colours, elemental target icon, selection dots, buffs/debuffs with applicable skill levels | Remaining target states and secondary indicators |
| Action bars and skills | Two draggable/rotatable bars, shortcuts, drag and drop, original single-column skill book, confirmed reuse timers and native press feedback | Additional settings and complete state coverage |
| Inventory | Equipment, bags, drag, left-click use, right-click equipment swaps, tooltips without zero sale prices and larger full-body preview | Remaining contexts and special dialogs |
| Weapon enchantments | Original Weapon.Seff, ninety recipes across fifteen weapon families, equipped and remote levels, both dual bindings | Full visual comparison across animations; see [weapon enchantments](weapon-enchantments.md) |
| Cloaks | Original PC/CTL/3DC loading, equipment slot 7, all body families, native spring modes and animated attachment pins | Body/ground contact adapter, map wind and guild emblem overlay; full physical parity is not yet claimed |
| Status | Allocation, EXP and current/max HP/MP/SP bars | Native graph and full attribute help |
| Chat | Phoenix geometry, embedded Liberation Sans, channel colours, persistent grey histories, single input row, seven-channel picker and original SysMsg messages | Speech bubbles, remaining special notices and complete state comparison |
| Effects | Independently draggable positive/negative strips; 32px icons, 6px gaps, native yellow duration bars and the original white debuff pulse | Full effect categories and all special states |
| Maps | Native map art, ring buttons, expanded-map name and compact clock, draggable minimap/full map, pan and zoom | Remaining map services and indicator variants |
| Options | Menu and six subpages; classic glow/brightness/detail/distance sliders, cloak/helmet visibility, audio, mouse, chat and monitor resolution selection | Other service controls and shortcut editing |
| Party/raid | Member frames, invitations, leader and selection flows | Complete badges, buffs and visual variants |
| Player context | Original classic TipBox frame, SysMsg rows, hover, character/target/party picking, whisper focus and supported party/trade/duel/guild actions | Inspect, follow, personal shops and remaining social windows/services |
| Quests | Original U button, journal artwork, state icons, seven 31px rows, three scrollbars, selection and confirmed abandonment; NPC dialogs/rewards/progress | Native tracker and remaining markers |
| Trade/duels | Wager/item/money interfaces and confirmation flows | Complete comparison across all states |
| Guild | Original G button, single roster sheet, native columns/rows, sorting, applicants, rank actions and notice editor; directory/creation/ranking flows; Buddy, Block and Search Party with native transport | Complete comparison of rare social states and remaining presentation |
| NPC services | Merchant, storage, blacksmith, gems/enchant and original portal window; portal approach, confirmation, same-map and cross-map travel | Remaining service types and special states |
| Mall | Catalog, correct Cash icon atlas, gifts and confirmations | Remaining pages/states and full comparison |
| Actions | Auto attack/pickup acquire nearby targets; selected-target attacks and spells turn immediately and approach before requesting | Complete navigation/collision and remaining actions |
| Combat numbers | Original count atlas with Phoenix damage, critical, miss and green healing billboards; server-confirmed results only | Healing from item pool snapshots lacks an explicit gain event |
| Spell/map effects | Native EFT/3DE particles, mirrored halos, caster/target area announcements and release-clock synchronization | Remaining native effect branches and full per-skill comparison |

## Recent recovered parameters

- Options construction: `0x4fbbb0`; subpage placement: `0x4fa800`. Main buttons are at `(64, 77 + 32*i)`, drawn at 128×32 with 128×22 hit regions. Original coordinates take precedence over Phoenix's compact spacing.
- Cash catalog rendering: `0x4d8b10` calls `0x4a3400` with type 200 and `Cash.Icon - 1`. This selects the type-100 atlas, or type-102 after index 1000; it is not a normal consumable icon.
- Duration bars: `0x4a2fa0`; a 32×5 black background below the icon contains a 30×3 yellow fill.
- Status: EXP at `(73,132)`, 183×15; resources at `(73,265 + 20*i)`, 68×15, following Phoenix's restored layout.
- Inventory preview: 136×217 opening at `(54,33)`, 34-degree FOV and height-based framing. Bind-pose arm span no longer shrinks the idle actor. This framing is calibrated against the reference composition; the complete native camera matrix has not been recovered.
- Inventory gold: native comma grouping (`0x4ce790`/`0x4cb420`) and the `0x4cb4a0` colour table. Boundaries are 1,000 (`80FFFF`), 10,000 (`0080FF`), 100,000 (`00FF40`), 1,000,000 (`FFFF00`), 10,000,000 (`FF8000`), 100,000,000 (`FF00FF`) and 1,000,000,000 (`8000FF`); smaller balances are white. The requested coloured presentation is always enabled.
- Minimap: native frame, button angles −45/−27/40 degrees and a 150×40 information plate at `(9,157)`. Full-map close uses the shared 22px control.
- Bottom bar: atlas slot 5 opens Quest (U); slot 6 opens Guild (G). Both reflect their window's open state. Mouse and keyboard share the same toggle path.
- Guild: `0x4f2960` selects `guild.tga`, not a cropped `guild_master.tga` directory. Column origins are 36/62/184/247/276, with 25/121/61/27/37px hit widths. `0x4ef9d0` draws rows from Y=112 in 23px steps; management controls are at Y=466. The notice uses `guild_message_input.tga` and its original 256×156 visible area.
- Quest journal: `0x4fedc0` and `0x4ff2c0`; seven rows at a 31px pitch, alarm icons `(26,63)`, names `(46,65)`, and original separators. Abandon is at `(402,453)`, drawn 128×32 with an 80×26 hit area. Native scrollbars cover the list `(227,52,13,237)`, objective `(227,320,13,130)` and description `(488,52,13,397)`.
- Portals: `0x494730`/`0x494490`; 512×366 artwork with the native centred origin, three destination rows and original Move buttons at `(434,80 + 101*i)`. Welcome text is at `(26,92)` with a 193px width, ten visible lines and its own scrollbar. Portraits use the NPC definition's Model, matching the native portrait setter, instead of the previous fixed face 55.

Click a portal NPC within 50 units to approach and open its conversation. `N` also chooses a nearby NPC; portals can be acquired at that distance while other NPC services retain their existing interaction range. Movement cancels the approach normally. Choose a populated destination, press Move and confirm. The client sends the NPC ID and destination index; GX Server validates availability, faction, distance and money, then supplies the balance and teleport position. Empty destinations remain inactive, including correct support for valid map-zero destinations with nonzero coordinates. Paid cross-map tests also verify scene replacement and persistence after reconnecting.

Options remains nonmodal: opening it preserves keyboard movement and automatic travel. Pointer gestures belong to the window where they started. Logout confirmations remain modal and await the server response.

Visual approval and complete comparison against the original remain ongoing. Opt-in rendering tests exercise real archives and preserve captures only in ignored local directories.

The player popup follows `0x47de10`/`0x47e1e0`: text starts at `(10,10)` with 15px rows. Its classic skin uses `TipBox.TGA`, 4px borders and the native `CD1D1F24` background. The `RightMenu_*` feature skin is absent from the local archive. The native minimum width is 90px of content plus borders; longer translated labels expand the width to fit the current font. Existing actions use original SysMsg labels, including party request/kick/leadership/leave (5002–5005), whisper (5007), battle (5008), transaction (5091), guild duel (5339) and raid leadership controls (6507–6509).

Right-click another character, the target frame or a party/raid member to open their actions. Right-click the local portrait in a group to leave it. Whisper selects the recipient and focuses the existing chat input. The popup preserves held movement and owns its pointer gestures; changed rows cannot activate an action that replaced the one originally pressed. Group leadership, capacity, faction and guild eligibility filter supported requests; membership, trades and duels still await GX Server responses. Friends/ignore, inspect, follow and personal-shop services are not represented by invented actions.

## Cloaks and combat presentation

Cloaks use original archive files directly, without converted or recreated meshes. Equipment types 24/39 in slot 7 select a zero-based item Model: Model 0 is the first cloak. Chest ReqVg minus one selects the PC fit, with out-of-range fits falling back to zero. Each PC record selects its own body, shoulder and racial CTL texture. The shoulder is skinned and the cloth follows the record's bone pins. The same appearance mapping serves selection, inventory, the local character and other players.

The CPU solver reproduces native directed springs, masses, forces, Euler integration and both cloth modes. Synthetic one-step tests compare outputs with the original executable. Real-archive GL checks load all 3,072 body/fit/model combinations and simulate/render all 16 body families. The current runtime uses the native no-contact branch; body/ground contacts, map wind and guild emblem rendering remain incomplete.

Attacking or using a targeted skill turns toward the current target immediately and publishes the heading before the action packet. Targets outside weapon/skill range start an approach; the stored skill is sent once when in range. Manual movement, a changed target or a new action cancels/replaces that request. This uses existing movement collision handling rather than a new pathfinding system. Results and resource changes remain server-owned.

Floating numbers use `count.tga` digit cells of 26×32, the Critical region at `(260,0,156,32)` and Miss at `(416,0,96,32)`. Phoenix parameters are a one-second lifetime, 0.3-unit rise per second, 0.2-unit digit width and 1.1875 height ratio. Critical digits are 1.5 times wider; healing is RGB `2ECC71`. Numbers detach from the target's displayed position, retain world scale under camera movement and expire from a queue capped at 128 entries. Attack, skill, drain and periodic result packets supply the displayed values; ordinary HP snapshots do not invent healing events.

EFT `Unknown29` is the mirrored-texture flag. Billboards with this flag use four reflected quadrants, matching Phoenix, so light and spell halos meet inward. The baked UVs use clamp addressing to avoid opposite-edge sampling; ordinary billboards and authored 3DE meshes retain their addressing. Original map-one `lamp003` is covered by an opt-in GL check.

Caster-centred area skills send no selected victim and render their body from the server's area announcement, even when there are no recipients. Target-centred areas use their announced target. Secondary hits independently select their recipient effect through the native damage/debuff/heal branches; a library's `DA_area_` filename does not suppress a recipient effect. Player effect anchors use the existing foot-space transform; the native logical-origin subtraction of 0.9 must not be applied twice. Projectile launch height is consequently 1.6 above the feet, with user/mob arrival points 0.9 above their feet.

Instant skill release animations begin when the request is sent; matching confirmations preserve the running action instead of restarting it. Direct effects catch up to that release clock, and projectiles catch up both their native 12-key windup and elapsed flight. A late result can start directly at its elapsed impact. Melee cues retain frame overshoot instead of restarting their effect clock. Melee contact markers and authored EFT sequence delays remain in use. Timed casts start from server cast-start messages; damage and effects still require confirmed results.

A server cast-start supersedes any predicted instant release. If local SData declares no preparation duration, the character and hand effects prepare until the server supplies a result or cancellation. Preparation time is not included in the subsequent release/impact clock. This matters for Frozen Ice: the local archive declares an instant skill, while the test server announces a three-second cast. The integration test now renders throughout that wait, checks fresh release clocks and verifies changed framebuffer pixels with and without the area effect, both with and without recipients.

Confirmed skills also create the separate caster release sequence used by native `0x57bdc0`/`0x57e320`. `IN_*` sequence zero supplies the release body; sequence one remains the preparation on the hands. Physical skills select the activation library through `0x5768e0`, except the native ranged weapon families, which use the generic release. This fills the missing early visual phase without deleting the later offsets authored in the target library. Release effects capture their world transform, recover elapsed animation time and are not recreated by secondary area hits. Native attack-speed modulation and the remaining special effect branches still need full comparison.

Login VANI assets now use the frame interval stored at header offset 48, rather than a fixed 12 fps. The original login water and rocks specify 33 ms per pose. Animated UVs use repeat addressing, including the negative and greater-than-one coordinates in the basin water. The falling water uses an unlit alpha pass following the Phoenix reference; the native flat-water geometry and UV poses remain intact. Real-asset GL checks cover the frame clock, addressing, early spell release, delayed target column and fountain emitters.

## Phoenix chat and native targets

The login logo stays visible during server selection and disappears with the login scenery.

Chat uses runtime `sysmsg-uni.txt` entries, including channel templates 353/354/355/370 and channel names 5086–5089/5377/5388. The formatter replaces native placeholders once; message bodies cannot introduce more placeholders. Normal messages use the native `name : message` format. Combat, experience, acquired gold/items, item use, learned skills, level changes, party invitations/membership, quest outcomes, trade and supported guild lifecycle notices use their original message IDs. Diagnostic commands retain explanatory fallback text where no native equivalent has been connected. Special service notices and speech bubbles are still incomplete.

The chat presentation follows Phoenix `chat_hud.gd` and `ep4_chat.gd`, per the user's explicit preference. The native SysMsg formatter and gameplay message connections remain in place. Histories are 350px wide with a 4px gap, 112/144px default heights and six/eight fully visible 16px text rows. Their fixed background is RGBA `(0.114, 0.122, 0.141, 0.54)`. Text uses Open Sans at 12px with a single soft black shadow; the mode label uses 11px, or 10px for Whisper. The entry row is always visible and combines a 64×22 mode button with a 286×22 nine-patch input built from `chat.tga`.

Native golden chat frames, hover-only backgrounds, Alpha/filter/help windows and the extra whisper-recipient row have been retired. Old filter/Alpha preferences no longer affect received messages. Resizing supports zero through twenty configured rows; history holds 300 messages globally and preserves independent scroll anchors. Row sizes and the selected channel persist; Whisper is not restored on startup. Clicking the message input or selecting a channel focuses the same entry row. With Whisper selected, enter `name message` when there is no known recipient; `!name message` and `/w name message` also work. Raid uses the server's party-chat transport.

| Channel | Phoenix RGB |
|---|---|
| Normal | `FFFFFF` |
| Party | `A5F078` |
| Whisper | `D65B75` |
| Guild | `EFA4D5` |
| Trade | `D7B48D` |
| Area | `70B9F5` |
| Raid | `FF7F00` |

System notices retain their original SysMsg text and native palette. The channel picker remains above movable action bars and owns its pointer gestures.

Mob names and target captions share `0x4cb250`'s signed monster-level minus player-level table. This intentionally retains the original high-level grey range. Element tiers repeat the four original target icons using `(element - 1) % 4`.

| Level difference | RGB |
|---|---|
| ≤ −6 | `FFFFFF` |
| −5…−4 | `80FFFF` |
| −3…−2 | `0000FF` |
| −1…+1 | `00FF00` |
| +2…+3 | `FFFF00` |
| +4…+5 | `FF8000` |
| +6…+7 | `FF0000` |
| +8…+9 | `FF00FF` |
| ≥ +10 | `808080` |

Selected monster names use original `name_select` dots, with the same 10px spacing from the text edge as Phoenix. The English client's grave-accent key (`0x43cd51`, DirectInput `0x29`) cycles living monsters by increasing distance and wraps. Backspace (`0x43c7a5`, `0x4cb2d0`) restores the previous selection; text editing still receives repeated Backspace events. The native locale-specific Tab variant is not substituted for the existing walk/run binding.

## Item details, cooldowns and cursors

Item properties use original SysMsg labels, yellow titles/base bonuses, orange remake values and white prices. Recovery amounts on consumable families are no longer shown as permanent equipment bonuses. Classes, stack metadata and extra attribute requirements have been removed from the tooltip. Original inline description controls and confirmed gem sockets remain. Full modifier-key/rarity presentation still needs comparison.

K has nine rows in one column: icons at `(46,78+43*i)`, names at `(88,81+43*i)`, requirements at y=97+43*i, and 32px learn buttons at `(278,84+43*i)` with 22px hit regions. Original SysMsg labels and red unmet requirements replace the compact two-column labels. Learning, scrolling, drag/drop, tooltips and skill use share those rows.

GX Server already validates skill reuse by SkillID and consumables by ReqIg. The client now suppresses requests during confirmed reuse, restores skill delays from the login snapshot and shares consumable timers across inventory/shortcut instances. ReqIg groups 1/2/3/4/5/6/10/11 mean 15/20/25/30/60/120/600/2 seconds. Timers start on accepted results. Inventory, K and action bars show the native eight-triangle sweep (`0x4a27d0`) and remaining seconds. Action icons use the original white `0x50FFFFFF` press overlay (`0x4e0160`), including held number/keypad keys; completed timers flash briefly. Flash timing is a presentation setting, not a recovered native constant. Consumable reconnect persistence follows the current GX Server session behavior; the protocol supplies no consumable timer snapshot.

Accepted healing potions followed by the authoritative resource snapshot emit green SysMsg 100 for each restored pool, using the actual clamped gain. Ordinary regeneration does not create potion notices. Buff badges display their transmitted level except absent/special level 100.

All seven `Cursor/01.png` through `07.png` files load from loose data into OS cursors. Lossless conversion preserves native dimensions and RGBA pixels, including the original AND-mask transparency. All supplied cursors use a `(0,0)` hotspot; the loader applies that directly. The CUR decoder remains an import utility. Context selection covers attack, loot, NPC conversation, bar interaction, dragging and skill aiming, with the default cursor elsewhere. Remaining specialized service/targeting states still need comparison. Cursor handles are released before the window system closes.

## Inventory, monster presentation and graphics controls

An ordinary left click requests use of the inventory item. A drag still moves
items; a release outside its source does not consume it. Right clicking a bag
item selects its equipment slot and requests the native move/swap, including
replacement. Accessory pairs choose an empty slot first. Results remain
server-authoritative. Provisional inventory instructions/waiting messages are
removed, and items with no sale value omit that line entirely.

Monsters use Monster.SData normal/chase timing and attack animation fields.
Walking ends on arrival, damage reactions cannot interrupt attacks, and attack
facing follows the victim. Monster skill results use their own 0x060b/c/d layouts,
including the attack-slot byte, level-one definition lookup and sequence-two
native impact route. Wire level100 is a marker. Repeated recipients of one area
attack do not restart its animation/audio. Speed changes and target snapshots
retain the monster's current server modifiers. Full per-monster comparison is
still ongoing.

The original debuff overlay (`0x4bff80`/`0x4c02d0`) fades white from alpha one at
one unit per second, wrapping at one. Icon hit regions and authoritative buff
lifetime remain intact. Login/server selection music is `Sound/Music/bgm_title.ogg`,
confirmed at `0x42caf0`; it replaces the previous `bgm_login001.wav` selection.

The four graphic thumbs use the original slider art at Y184/234/284/334, with
ranges glow0–2, brightness0–4, details0–2 and distance0–2. Glow uses native
neighbor/center weights (0/0,22/17,44/35). Brightness applies the windowed power
curve and +10 black offset to the completed game frame, including UI, without
changing desktop gamma. The original byte input divisor is256 (`0x541540`).
Details select texture LOD bias2/1/0 without regenerating assets. Distance uses
half/three-quarter/full view and fog ranges; this scaling is an implementation
adaptation, not a claim that the entire native distance pipeline is recovered.
Changes preview live, Cancel restores them, Reset restores defaults, and OK
persists them. Opening Options remains nonmodal.

Guild's side navigation now reaches the original Buddy, Block and Search Party
frames. Buddy supports invitation/acceptance, presence, whisper, deletion and
notes; right click a row to edit its note in the name field and confirm with
Enter. Block persists independently and filters chat. The fourth native page is
party search; mail in the auxiliary AzerothCore UI is not an EP5 transport.
GX Server supplies existing 0x2201–0x220e packets and existing SQLite tables. Native
Buddy result codes are0 rejection/1 acceptance/2 different area (SysMsg399–401);
block uses402/403 and search registration404/405. Candidate filtering and rare
original service states still need full comparison. Presence refreshes every
three seconds; membership changes follow ACKs.


## Resource checks, self-targeting and loot

Skill requests check the known MP/SP budget before sending or starting predicted animation. Failures use SysMsg 214 with the original MP/SP labels. Queued approach skills recheck on arrival; the server remains authoritative. The original cost value 1 retains its consume-current-pool meaning.

F2 or a left click on the local portrait selects the player as a normal friendly target; the target frame displays their name and current HP. Dropped items have no floating text. Click their models or use H to approach and pick up, stopping within the server's accepted ten-unit range with a safety margin. Pickup and removal still require server confirmation.

Friendly skill results are sent after the recipient applies the heal. Green numbers show effective HP gained rather than the requested amount; zero gain displays no number. Apply the updated GX Server build as well as the client to use the corrected healing announcements.

## Display selection

Graphics options list `Window` and the resolutions advertised by the current monitor. The original resolution arrow opens a scrollable list; mouse, wheel and arrow keys select a pending mode. Enter closes the list and Escape dismisses it before closing the page. Press **OK** to apply and save; **Cancel** discards the selection. Keyboard navigation does not open chat or release movement already held.

Fullscreen uses the existing GLFW window and OpenGL context. Returning to `Window` restores its previous size, position and maximized state. Normal window geometry is saved separately from the chosen display mode. Old settings default to a window; explicit `-w`/`-h` arguments override a saved fullscreen choice for that launch. Unsupported modes are refused and driver errors remain visible in the options page. The renderer stays at 32-bit colour. Wayland uses the primary monitor because it does not expose global window positions.

Validation includes a virtual-monitor fullscreen/window round trip, real OpenGL window resizing with texture preservation, the rendered options popup, and two-client input checks. A real-monitor fullscreen transition and native Windows runtime have not been validated in this local patch.

## Water and scene transparency

**Water Surface Effect** changes reflection/refraction work, not the visibility
of the surface. With the option disabled, the original animated WTR water stays
visible and the extra reflection scene and refraction capture are skipped.
Re-enabling the option restores the advanced path, including when a reflection
target already exists from an earlier frame.

Scene destination alpha stays opaque throughout drawing, including reflected
scenes, so Dispel's destination-alpha blend layer cannot darken the background
into black shapes. Individual textures retain source-alpha transparency. The
previous colour-write mask is restored afterward. Shadow silhouette targets
still write their own alpha coverage and restore the enclosing scene mask.
Regression checks exercise the real Dispel EFT, translucent layering, nested
shadow captures and the water option in a rendered map.

## Real-time sun shadows

On fields, **Shadow Effect** renders a 2048×2048 orthographic depth map from the scene light every frame, covering 45 units around the player and snapped to whole texels so edges do not shimmer while moving. World objects and actors cast shadows; terrain, sky and effects do not. Terrain, objects and actors receive them through 3×3 hardware PCF, faded out toward the edge of the covered area, darkening by 45% so the baked lightmaps still read correctly. While the sun map is active the legacy projected actor silhouettes are skipped; dungeons have no sun and keep the legacy silhouettes.
