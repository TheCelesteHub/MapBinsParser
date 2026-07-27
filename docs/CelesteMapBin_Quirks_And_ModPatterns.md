# Map `.bin` quirks and what mods actually put in them

Companion to [CelesteMapBin_Format.md](./CelesteMapBin_Format.md), which covers the byte layout. This one covers everything that surprised us when the parser met real mods, and why `classify()` in `src-utils/mapbin.go` looks the way it does.

Everything here is measured, not assumed. Source of the numbers: a full scan of a real install — **282 mods, 9.1 GB, 565 map `.bin` files, 565 parsed successfully, 0 failures**.

## Quirk 1: trailing bytes after the document are legal

**124 of 565 maps have data after the root element closes** — up to 568 KB of it. Every single one belongs to `StrawberryJam2021.zip`.

The documents themselves are complete and correct. `5-Grandmaster/Hydro.bin` is 588,865 bytes; the `Map` element (38 rooms, `Style`, `Filler`, `meta`) ends cleanly at byte 336,876, and the remaining 252 KB is structured data referencing the same lookup table but not reachable from the root.

A parser that validates "consumed == file length" rejects a fifth of the maps in a popular collab. **Stop at the end of the root element, ignore the rest.**

## Quirk 2: `triggers` are not `entities`

Berry- and heart-named things appear under both lists. Only `entities` are collectible. 18 distinct trigger names matched a berry/heart name pattern across the scan, and every one is a false positive:

| count | trigger |
|---|---|
| 129 | `FrostHelper/CassetteTempoTrigger` |
| 98 | `CollabUtils2/SpeedBerryCollectTrigger` |
| 75 | `goldenBerryCollectTrigger` |
| 33 | `FrostHelper/OnBerryCollectActivator` |
| 28 | `Bitsbolts/StrawberryAllowfield` |
| 18 | `MaxHelpingHand/StrawberryCollectionField` |
| 17 | `CollabUtils2/RainbowBerryUnlockCutsceneTrigger` |
| 15 | `CollabUtils2/MiniHeartDoorUnlockCutsceneTrigger` |
| 11 | `CollabUtils2/SilverBerryCollectTrigger` |
| 11 | `DSidesPlatinum/DetachPlatberryTrigger` |

Counting a `SpeedBerryCollectTrigger` as a speed berry would roughly double that stat in every collab.

## Quirk 3: the entity name is not `strawberry`

This is the one that breaks naive implementations. A mod's berries are frequently a custom entity from whichever helper the mapper happened to use.

**Farshore has zero `strawberry` entities.** Its 5 berries are `GameHelper/FlagCollectBerry`. The player has collected 2 of them, and the save file records 2 — with no hint anywhere that 5 is the maximum.

Same story for hearts: vanilla Forsaken City A-side uses `birdForsakenCityGem`, not `blackGem`. Old Site uses `dreamHeartGem`.

Hence pattern matching rather than a fixed name set. 101 distinct entity names matched the collectible name patterns across 282 mods — the long tail is real, but the head is short:

| count | entity | classified as |
|---|---|---|
| 1412 | `strawberry` | red (or moon, see quirk 4) |
| 405 | `CollabUtils2/MiniHeart` | mini heart |
| 348 | `CollabUtils2/SilverBerry` | silver |
| 171 | `goldenBerry` | golden / winged golden |
| 153 | `blackGem` | heart |
| 112 | `LunaticHelper/StrawberryWithReturn` | red |
| 94 | `CollabUtils2/SpeedBerry` | speed |
| 76 | `SorbetHelper/ReturnBerry` | red |
| 72 | `PlayerGhostMode/StrawberryWithFlagAndDialog` | red |
| 47 | `fakeHeart` | denied |
| 37 | `MaxHelpingHand/CustomizableBerry` | red |
| 35 | `SpringCollab2020/returnBerry` | red |
| 28 | `MaxHelpingHand/MultiRoomStrawberrySeed` | denied |
| 22 | `heartGemDoor` | denied |
| 18 | `dreamHeartGem` | heart |
| 17 | `CollabUtils2/RainbowBerry` | rainbow |

Your instinct was right: nearly everyone uses vanilla entities, CollabUtils2, or one of about three popular helpers (MaxHelpingHand, CommunalHelper, SpringCollab2020 leftovers).

## Quirk 4: classify by attribute *value*, never presence

Attributes are always written out, whether or not they are set. All 1342 `strawberry` entities in the scan carry `checkpointID`, `moon`, `order` and `winged` keys. Checking "does it have a `moon` attribute" classifies every berry in the game as a moonberry.

By value, across all 282 mods:

- `moon=true` — 56 (real moonberries)
- `winged=true` on a `strawberry` — 57 (winged **red** berries; they count as red, exactly like vanilla)
- `winged=true` on a `goldenBerry` — 2 (winged goldens)

## Quirk 5: berry-shaped things that are not berries

Three families, all denied:

**Decoys.** `FemtoHelper/CustomFakeHeart` (304), `CollabUtils2/FakeMiniHeart` (87), `fakeHeart` (47), `BrokemiaHelper/trollStrawberry` (52) + its SpringCollab and Chronia clones (14), `KoseiHelper/FlagFakeBerry` (6). These vanish or do nothing when touched. Matched by `/troll|fake/i`.

**Machinery named after collectibles.** Gates, doors, blocks, controllers, spawners, respawn points:
`CollabUtils2/GoldenBerryPlayerRespawnPoint` (200), `SJ2021/StrawberryJamJar` (111 — the lobby display jar), `heartGemDoor` (22), `CollabUtils2/MiniHeartDoor` (23), `VivHelper/GoldenBerryToFlag` (10), `KoseiHelper/SetFlagOnBerryCollectionController` (14), `MaxHelpingHand/SaveFileStrawberryGate` (3), `LunaticHelper/StrawberryGate` (3), `Bitsbolts/GrablessBerryBlock` (5). Matched by a suffix pattern: `trigger|controller|gate|door|block|spawner|respawnpoint|toflag|seed|jar|cabin|switch|field|activator`.

**Seeds.** `MaxHelpingHand/MultiRoomStrawberrySeed` (28) and `SpringCollab2020/MultiRoomStrawberrySeed` (5). Seeds belong to a parent berry that is already counted; counting them would multiply that berry by its seed count. Covered by the `seed` suffix.

**Named individually** because no pattern catches them: `vitellary/keyberry` (a key reskinned as a berry), `FactoryHelper/MachineHeart`, `CommunalHelper/CustomSummitGem` (summit gem, not a crystal heart), `ScugHelper/SquareGem`, `HeliosHelper/SolHeartsideSwitch`, `IntoTheJungleCodeMod/LanternHeartSpawner`.

Cassettes are ignored wholesale. There is no cassette field in `ModBasicStats`, and `cassetteBlock` (3017 occurrences) plus 20 other cassette-named block variants would be pure noise.

## Quirk 6: a map can hold more hearts than a player can collect

Vanilla `2-OldSite` contains **two** `blackGem` entities — one in `lvl_end_s1`, one in `lvl_s2` — but the chapter awards one heart. Scanning all of vanilla gives 25 hearts against the true 24.

The same happens in mods: 10 of StrawberryJam2021's 111 chapters place 2+ `CollabUtils2/MiniHeart` (`3-Advanced/Viv` places 4), presumably conditional or cutscene placements.

This is the accepted ceiling of counting placed entities. It is why vanilla keeps its hardcoded totals instead of using the scan. Berries do not suffer from it — vanilla sums to exactly 175 red, 25 golden and 1 moonberry.

## Quirk 7: sides are separate files, chapters are not

The save file folds A/B/C sides of one chapter into a single `AreaStats` SID with up to three `AreaModeStats`. On disk they are separate maps: `1-ForsakenCity.bin`, `1H-ForsakenCity.bin`, `1X-ForsakenCity.bin`, each its own SID in the scan output.

So global totals sum every `.bin` in the mod (correct — a B-side golden is genuinely available), but per-side `berriesAvailable` is only filled where a scanned SID matches a played chapter exactly, which in practice means the A-side. B/C sides hold no red berries anyway, which is why vanilla still sums to exactly 175.

Global totals also include chapters the player has never opened. That is intentional: unplayed content is still available content.

## Quirk 8: what "a mod" looks like on disk

Relevant to the scan, from the same 282-mod sample:

- Most mods are `.zip`; some are unpacked folders. `CountCollectibles` handles both through one code path.
- Most mods have **no maps at all** — helpers, skins, tools, UI. 282 mods yielded only 565 map files, and a large majority of those come from a handful of collabs. Mods without a `Maps/` directory return an empty result instantly.
- Collabs dominate the map count. StrawberryJam2021 alone is 128 maps, laid out as `0-Lobbies/`, `0-Gyms/`, `1-Beginner/` ... `5-Grandmaster/`, each difficulty tier holding one map per author plus a `ZZ-HeartSide.bin`.
- Lobby and gym maps hold no collectibles by design. Heart-side maps hold the crystal heart but no silver berry.

## The validation anchors

Kept as tests in `testing/go-utils-tests/Zip_Go_CountCollectibles.test.ts`:

| target | expectation | why it proves something |
|---|---|---|
| vanilla `Content/Maps` | 175 red, 25 golden, 1 moon | matches the hardcoded vanilla totals exactly |
| `Farshore.zip` | 5 red, 1 golden | the berries are `GameHelper/FlagCollectBerry`, so a `strawberry`-only implementation returns 0 |
| `StrawberryJam2021.zip` | 128 maps parsed, 0 failures; all 111 real chapters have exactly 1 silver berry and at least 1 mini heart | covers trailing bytes, CollabUtils2 entities, and the trigger/entity split at scale |
| synthetic map built in the test | decoys, seeds, gates and triggers all excluded | the deny list, without needing a mod installed |
