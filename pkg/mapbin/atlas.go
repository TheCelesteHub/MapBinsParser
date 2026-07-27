package mapbin

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// SpriteMeta is one entry from a Celeste Gameplay.meta atlas index: the quad
// (within its owning .data image) a given atlas path (e.g. "tilesets/dirt" or
// "decals/1-forsakencity/introcliffsidegrass0") maps to.
type SpriteMeta struct {
	DataFile              string
	X, Y                  int
	Width, Height         int
	OffsetX, OffsetY      int
	RealWidth, RealHeight int
}

// Atlas is a decoded Celeste packed sprite atlas (Gameplay.meta + its .data files).
type Atlas struct {
	dir     string
	sprites map[string]SpriteMeta
	images  map[string]*image.RGBA
}

func readNetString(r *bufio.Reader) (string, error) {
	length := 0
	shift := uint(0)
	for {
		b, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		length |= int(b&0x7f) << shift
		if b&0x80 == 0 {
			break
		}
		shift += 7
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

func readInt16LE(r *bufio.Reader) (int16, error) {
	var v int16
	err := binary.Read(r, binary.LittleEndian, &v)
	return v, err
}

func readInt32LE(r *bufio.Reader) (int32, error) {
	var v int32
	err := binary.Read(r, binary.LittleEndian, &v)
	return v, err
}

// LoadAtlas decodes a Gameplay.meta sprite index (found under
// <celesteContentDir>/Graphics/Atlases) into an Atlas whose sprite pixels are
// decoded lazily (and cached) per owning .data file on first GetSprite call.
func LoadAtlas(atlasDir, metaFileName string) (*Atlas, error) {
	f, err := os.Open(filepath.Join(atlasDir, metaFileName))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := bufio.NewReader(f)

	if _, err := readInt32LE(r); err != nil { // version
		return nil, fmt.Errorf("read atlas version: %w", err)
	}
	if _, err := readNetString(r); err != nil { // checksum
		return nil, fmt.Errorf("read atlas checksum: %w", err)
	}
	if _, err := readInt32LE(r); err != nil { // unused header field
		return nil, fmt.Errorf("read atlas header: %w", err)
	}
	dataFileCount, err := readInt16LE(r)
	if err != nil {
		return nil, fmt.Errorf("read atlas data file count: %w", err)
	}

	atlas := &Atlas{
		dir:     atlasDir,
		sprites: make(map[string]SpriteMeta),
		images:  make(map[string]*image.RGBA),
	}

	for i := int16(0); i < dataFileCount; i++ {
		dataFile, err := readNetString(r)
		if err != nil {
			return nil, fmt.Errorf("read data file name: %w", err)
		}
		spriteCount, err := readInt16LE(r)
		if err != nil {
			return nil, fmt.Errorf("read sprite count: %w", err)
		}

		for j := int16(0); j < spriteCount; j++ {
			rawPath, err := readNetString(r)
			if err != nil {
				return nil, fmt.Errorf("read sprite path: %w", err)
			}
			path := strings.ReplaceAll(rawPath, "\\", "/")

			fields := make([]int16, 8)
			for k := range fields {
				v, err := readInt16LE(r)
				if err != nil {
					return nil, fmt.Errorf("read sprite quad for %q: %w", path, err)
				}
				fields[k] = v
			}

			atlas.sprites[path] = SpriteMeta{
				DataFile:   dataFile,
				X:          int(fields[0]),
				Y:          int(fields[1]),
				Width:      int(fields[2]),
				Height:     int(fields[3]),
				OffsetX:    int(fields[4]),
				OffsetY:    int(fields[5]),
				RealWidth:  int(fields[6]),
				RealHeight: int(fields[7]),
			}
		}
	}

	return atlas, nil
}

// loadDataImage decodes a Celeste .data pixel file: int32 width, int32 height,
// bool hasAlpha, then a run-length-encoded stream of premultiplied-alpha BGR(A)
// pixels in row-major order (runs may span row boundaries).
func loadDataImage(path string) (*image.RGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := bufio.NewReader(f)

	width, err := readInt32LE(r)
	if err != nil {
		return nil, fmt.Errorf("read data width: %w", err)
	}
	height, err := readInt32LE(r)
	if err != nil {
		return nil, fmt.Errorf("read data height: %w", err)
	}
	hasAlphaByte, err := r.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("read data hasAlpha: %w", err)
	}
	hasAlpha := hasAlphaByte != 0

	img := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))

	totalPixels := int(width) * int(height)
	repeatsLeft := 0
	var curR, curG, curB, curA uint8

	for p := 0; p < totalPixels; p++ {
		if repeatsLeft == 0 {
			rep, err := r.ReadByte()
			if err != nil {
				return nil, fmt.Errorf("read data run length: %w", err)
			}
			repeatsLeft = int(rep) - 1

			if hasAlpha {
				alpha, err := r.ReadByte()
				if err != nil {
					return nil, fmt.Errorf("read data alpha: %w", err)
				}
				if alpha > 0 {
					b, err := r.ReadByte()
					if err != nil {
						return nil, err
					}
					g, err := r.ReadByte()
					if err != nil {
						return nil, err
					}
					rr, err := r.ReadByte()
					if err != nil {
						return nil, err
					}
					// file bytes are already alpha-premultiplied, same convention as Go's image.RGBA.
					curR, curG, curB, curA = rr, g, b, alpha
				} else {
					curR, curG, curB, curA = 0, 0, 0, 0
				}
			} else {
				b, err := r.ReadByte()
				if err != nil {
					return nil, err
				}
				g, err := r.ReadByte()
				if err != nil {
					return nil, err
				}
				rr, err := r.ReadByte()
				if err != nil {
					return nil, err
				}
				curR, curG, curB, curA = rr, g, b, 255
			}
		} else {
			repeatsLeft--
		}

		x := p % int(width)
		y := p / int(width)
		img.SetRGBA(x, y, color.RGBA{R: curR, G: curG, B: curB, A: curA})
	}

	return img, nil
}

// GetSprite returns the decoded image for the given atlas path (e.g.
// "tilesets/dirt" or "decals/1-forsakencity/introcliffsidegrass0"), decoding
// and caching its owning .data file on first use.
func (a *Atlas) GetSprite(path string) (image.Image, SpriteMeta, bool) {
	meta, ok := a.sprites[path]
	if !ok {
		return nil, SpriteMeta{}, false
	}

	img, ok := a.images[meta.DataFile]
	if !ok {
		decoded, err := loadDataImage(filepath.Join(a.dir, meta.DataFile+".data"))
		if err != nil {
			return nil, SpriteMeta{}, false
		}
		img = decoded
		a.images[meta.DataFile] = img
	}

	sub := img.SubImage(image.Rect(meta.X, meta.Y, meta.X+meta.Width, meta.Y+meta.Height))
	return sub, meta, true
}
