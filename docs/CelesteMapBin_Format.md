# Celeste map `.bin` format (BinaryPacker)

Reference for `src-utils/mapbin.go`. Companion document: [CelesteMapBin_Quirks_And_ModPatterns.md](./CelesteMapBin_Quirks_And_ModPatterns.md), which covers what real mods actually put inside these files.

## Why this exists

`.celeste` save files record only what a player **collected**. There is no maximum anywhere in them — `AreaModeStats TotalStrawberries="22"` is a collected count, identical to the length of the `<Strawberries><EntityID>` list beside it. Celeste itself learns the maximums at runtime by parsing map `.bin` files into `AreaData`.

So `2 / ?` becomes `2 / 5` only by reading the maps. This document is the format that makes that possible.

Neither Loenn nor Ahorn document this format publicly; everything below was derived by hexdumping real maps and validated against **565 map files across 282 installed mods (9.1 GB) — 565 parsed, 0 failures**.

## Layout

Every `.bin` is one BinaryPacker document:

```
<varint-string>        "CELESTE MAP"          magic
<varint-string>        package name           (unused by us)
uint16 LE              lookup table length    N
N x <varint-string>    lookup table           every element and attribute name, deduplicated
<element>              root element           always "Map"
```

`varint-string` is .NET `BinaryWriter.Write(string)`: a 7-bit encoded length prefix (low 7 bits per byte, high bit means "another byte follows") followed by raw UTF-8 bytes.

An element is:

```
uint16 LE   nameIdx        index into the lookup table
uint8       attrCount
attrCount x  { uint16 LE keyIdx, uint8 type, value }
uint16 LE   childCount
childCount x <element>     recursive
```

### Attribute value types

| type | encoding | bytes |
|---|---|---|
| `0` | bool | 1 |
| `1` | unsigned byte | 1 |
| `2` | int16 LE | 2 |
| `3` | int32 LE | 4 |
| `4` | float32 LE | 4 |
| `5` | `uint16` index into the lookup table | 2 |
| `6` | inline `varint-string` | varies |
| `7` | run-length encoded string: `uint16` **byte** length, then `(count uint8, char uint8)` pairs | varies |

Type `7` is what tile data (`solids`, `bg`) uses. The `uint16` is the length of the encoded pairs in bytes, not the length of the decoded string.

### Document shape

```
Map
├── levels
│   └── level            one per room; "name" attribute is the room id
│       ├── solids       tile data
│       ├── bg
│       ├── entities     <- collectibles live here
│       ├── triggers     <- never collectibles, see the quirks doc
│       └── ...
├── Style
├── Filler
└── meta
```

A worked example, the first bytes of `Content/Maps/1-ForsakenCity.bin`:

```
0b 43 45 4c 45 53 54 45 20 4d 41 50   len=11, "CELESTE MAP"
0e 31 2d 46 6f 72 73 61 6b 65 6e ...  len=14, "1-ForsakenCity"
04 01                                 lookup table: 260 entries
03 4d 61 70                           [0] "Map"
06 46 69 6c 6c 65 72                  [1] "Filler"
04 72 65 63 74                        [2] "rect"
01 78                                 [3] "x"
...
```

## Reading it: `src-utils/mapbin.go`

The Go reader is a flat top-to-bottom walk, four named steps:

1. **`readHeader`** — magic, package name, lookup table.
2. **`readElement(container)`** — the recursive walk. `container` carries `"entities"` or `"triggers"` down into children so the classifier knows which list an element came from.
3. **`readAttributes`** — reads every attribute but keeps only `moon` and `winged`; everything else is skipped in place by type.
4. **`count(name, moon, winged)`** — classification (see the quirks doc for the rules and the data behind them).

`CountCollectibles(modPath)` drives it over every `Maps/**/*.bin` in a mod, zip or unpacked folder, keyed by SID (`Maps/Crylone/farshore/farshore.bin` -> `Crylone/farshore/farshore`). A map that fails to parse lands in `failed` instead of aborting the mod.

**The reader stops when the root element closes and ignores whatever follows.** Roughly 20% of real maps carry trailing bytes — see the quirks doc. Asserting EOF would reject them.

## Calling it

CLI (both `utilities-*` and `zip_utils-*` binaries expose it):

```bash
bin/utilities-win_x64.exe zip count-collectibles --mod "C:/.../Celeste/Mods/Farshore.zip"
```

```json
{"success":true,"maps":{"Crylone/farshore/farshore":{"red":5,"golden":1,"wingedGolden":0,"moon":0,"hearts":0,"miniHearts":0,"silver":0,"speed":0,"rainbow":0,"platinum":0}}}
```

From TypeScript: `Zip_Go.countCollectibles(modPath)`.

Consumed by `LocalModsStatsCalculator.#GetCollectibleTotals()`, which caches results per mod under the storage key `localmods_collectibletotals:<modId>` and recomputes when the mod's `sizeBytes` changes. `#ApplyCollectibleTotals()` then writes them into `global.*.total` and each played chapter's `berriesAvailable`.

Vanilla Celeste deliberately does **not** go through this path — its totals stay hardcoded in `createEmptyVanillaModBasicStats()`. The parser agrees with them anyway (175 red, 25 golden, 1 moon), which is exactly why they make a good regression anchor: `testing/go-utils-tests/Zip_Go_CountCollectibles.test.ts`.
