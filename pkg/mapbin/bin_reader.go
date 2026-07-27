package mapbin

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strings"
)

type BinReader struct {
	data   []byte
	pos    int
	lookup []string
	counts *MapCollectibleCounts
}

func NewBinReader(data []byte, counts *MapCollectibleCounts) *BinReader {
	return &BinReader{data: data, counts: counts}
}

func (r *BinReader) u8() (byte, error) {
	if r.pos >= len(r.data) {
		return 0, io.ErrUnexpectedEOF
	}
	b := r.data[r.pos]
	r.pos++
	return b, nil
}

func (r *BinReader) u16() (int, error) {
	if r.pos+2 > len(r.data) {
		return 0, io.ErrUnexpectedEOF
	}
	v := binary.LittleEndian.Uint16(r.data[r.pos:])
	r.pos += 2
	return int(v), nil
}

func (r *BinReader) skip(n int) error {
	if n < 0 || r.pos+n > len(r.data) {
		return io.ErrUnexpectedEOF
	}
	r.pos += n
	return nil
}

func (r *BinReader) varint() (int, error) {
	result, shift := 0, 0
	for {
		b, err := r.u8()
		if err != nil {
			return 0, err
		}
		result |= int(b&0x7F) << shift
		if b&0x80 == 0 {
			return result, nil
		}
		shift += 7
		if shift > 28 {
			return 0, fmt.Errorf("varint too long at offset %d", r.pos)
		}
	}
}

func (r *BinReader) str() (string, error) {
	n, err := r.varint()
	if err != nil {
		return "", err
	}
	if r.pos+n > len(r.data) {
		return "", io.ErrUnexpectedEOF
	}
	s := string(r.data[r.pos : r.pos+n])
	r.pos += n
	return s, nil
}

func (r *BinReader) lookupAt(idx int) (string, error) {
	if idx < 0 || idx >= len(r.lookup) {
		return "", fmt.Errorf("lookup index %d out of range (%d entries)", idx, len(r.lookup))
	}
	return r.lookup[idx], nil
}

func (r *BinReader) ReadHeader() error {
	magic, err := r.str()
	if err != nil {
		return err
	}
	if magic != "CELESTE MAP" {
		return fmt.Errorf("not a Celeste map (magic %q)", magic)
	}
	if _, err = r.str(); err != nil {
		return err
	}
	count, err := r.u16()
	if err != nil {
		return err
	}
	r.lookup = make([]string, count)
	for i := range count {
		if r.lookup[i], err = r.str(); err != nil {
			return err
		}
	}
	return nil
}

func (r *BinReader) readAttributes(count int) (bool, bool, error) {
	moon, winged := false, false

	for range count {
		keyIdx, err := r.u16()
		if err != nil {
			return false, false, err
		}
		key, err := r.lookupAt(keyIdx)
		if err != nil {
			return false, false, err
		}
		valueType, err := r.u8()
		if err != nil {
			return false, false, err
		}

		boolValue := false
		switch valueType {
		case 0:
			b, readErr := r.u8()
			if readErr != nil {
				return false, false, readErr
			}
			boolValue = b != 0
		case 1:
			err = r.skip(1)
		case 2:
			err = r.skip(2)
		case 3, 4:
			err = r.skip(4)
		case 5:
			err = r.skip(2)
		case 6:
			var n int
			if n, err = r.varint(); err == nil {
				err = r.skip(n)
			}
		case 7:
			var n int
			if n, err = r.u16(); err == nil {
				err = r.skip(n)
			}
		default:
			return false, false, fmt.Errorf("unknown attribute type %d at offset %d", valueType, r.pos)
		}
		if err != nil {
			return false, false, err
		}

		switch key {
		case "moon":
			moon = boolValue
		case "winged":
			winged = boolValue
		}
	}

	return moon, winged, nil
}

func (r *BinReader) ReadElement(container string) error {
	nameIdx, err := r.u16()
	if err != nil {
		return err
	}
	name, err := r.lookupAt(nameIdx)
	if err != nil {
		return err
	}
	attrCount, err := r.u8()
	if err != nil {
		return err
	}
	moon, winged, err := r.readAttributes(int(attrCount))
	if err != nil {
		return err
	}

	if container == "entities" {
		CountEntity(r.counts, name, moon, winged)
	}

	childCount, err := r.u16()
	if err != nil {
		return err
	}
	childContainer := container
	if name == "entities" || name == "triggers" {
		childContainer = name
	}
	for range childCount {
		if err := r.ReadElement(childContainer); err != nil {
			return err
		}
	}
	return nil
}

func decodeRLEString(data []byte) (string, error) {
	var builder strings.Builder
	for i := 0; i+1 < len(data); i += 2 {
		count := int(data[i])
		char := data[i+1]
		for c := 0; c < count; c++ {
			builder.WriteByte(char)
		}
	}
	return builder.String(), nil
}

func (r *BinReader) readAttributeValue() (interface{}, error) {
	valueType, err := r.u8()
	if err != nil {
		return nil, err
	}
	switch valueType {
	case 0:
		b, err := r.u8()
		if err != nil {
			return nil, err
		}
		return b != 0, nil
	case 1:
		b, err := r.u8()
		if err != nil {
			return nil, err
		}
		return int(b), nil
	case 2:
		if r.pos+2 > len(r.data) {
			return nil, io.ErrUnexpectedEOF
		}
		v := int16(binary.LittleEndian.Uint16(r.data[r.pos:]))
		r.pos += 2
		return int(v), nil
	case 3:
		if r.pos+4 > len(r.data) {
			return nil, io.ErrUnexpectedEOF
		}
		v := int32(binary.LittleEndian.Uint32(r.data[r.pos:]))
		r.pos += 4
		return int(v), nil
	case 4:
		if r.pos+4 > len(r.data) {
			return nil, io.ErrUnexpectedEOF
		}
		bits := binary.LittleEndian.Uint32(r.data[r.pos:])
		r.pos += 4
		return float64(math.Float32frombits(bits)), nil
	case 5:
		idx, err := r.u16()
		if err != nil {
			return nil, err
		}
		return r.lookupAt(idx)
	case 6:
		return r.str()
	case 7:
		n, err := r.u16()
		if err != nil {
			return nil, err
		}
		if r.pos+n > len(r.data) {
			return nil, io.ErrUnexpectedEOF
		}
		rleData := r.data[r.pos : r.pos+n]
		r.pos += n
		return decodeRLEString(rleData)
	default:
		return nil, fmt.Errorf("unknown attribute type %d at offset %d", valueType, r.pos)
	}
}

func (r *BinReader) ReadAllAttributes(count int) (map[string]interface{}, error) {
	attrs := make(map[string]interface{}, count)
	for i := 0; i < count; i++ {
		keyIdx, err := r.u16()
		if err != nil {
			return nil, err
		}
		key, err := r.lookupAt(keyIdx)
		if err != nil {
			return nil, err
		}
		val, err := r.readAttributeValue()
		if err != nil {
			return nil, err
		}
		attrs[key] = val
	}
	return attrs, nil
}
