package auth

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

type passkeyCBORParser struct {
	data []byte
}

func decodePasskeyCBOR(data []byte) (interface{}, int, error) {
	parser := passkeyCBORParser{data: data}
	return parser.parse(0)
}

func (p passkeyCBORParser) readArgument(pos int, info byte) (uint64, int, error) {
	switch {
	case info < 24:
		return uint64(info), pos, nil
	case info == 24:
		if pos >= len(p.data) {
			return 0, pos, errors.New("truncated cbor uint8")
		}
		return uint64(p.data[pos]), pos + 1, nil
	case info == 25:
		if pos+2 > len(p.data) {
			return 0, pos, errors.New("truncated cbor uint16")
		}
		return uint64(binary.BigEndian.Uint16(p.data[pos : pos+2])), pos + 2, nil
	case info == 26:
		if pos+4 > len(p.data) {
			return 0, pos, errors.New("truncated cbor uint32")
		}
		return uint64(binary.BigEndian.Uint32(p.data[pos : pos+4])), pos + 4, nil
	case info == 27:
		if pos+8 > len(p.data) {
			return 0, pos, errors.New("truncated cbor uint64")
		}
		return binary.BigEndian.Uint64(p.data[pos : pos+8]), pos + 8, nil
	default:
		return 0, pos, fmt.Errorf("unsupported cbor additional info %d", info)
	}
}

func (p passkeyCBORParser) parse(pos int) (interface{}, int, error) {
	if pos >= len(p.data) {
		return nil, pos, errors.New("empty cbor value")
	}

	head := p.data[pos]
	pos++
	major := head >> 5
	info := head & 0x1f
	arg, next, err := p.readArgument(pos, info)
	if err != nil {
		return nil, pos, err
	}
	pos = next

	switch major {
	case 0:
		if arg > math.MaxInt64 {
			return nil, pos, errors.New("cbor integer overflow")
		}
		return int64(arg), pos, nil
	case 1:
		if arg > math.MaxInt64 {
			return nil, pos, errors.New("cbor integer overflow")
		}
		return -1 - int64(arg), pos, nil
	case 2:
		if arg > uint64(len(p.data)-pos) {
			return nil, pos, errors.New("truncated cbor bytes")
		}
		end := pos + int(arg)
		return p.data[pos:end], end, nil
	case 3:
		if arg > uint64(len(p.data)-pos) {
			return nil, pos, errors.New("truncated cbor text")
		}
		end := pos + int(arg)
		return string(p.data[pos:end]), end, nil
	case 4:
		if arg > math.MaxInt32 {
			return nil, pos, errors.New("cbor array too large")
		}
		items := make([]interface{}, 0, int(arg))
		for i := 0; i < int(arg); i++ {
			value, itemNext, err := p.parse(pos)
			if err != nil {
				return nil, pos, err
			}
			pos = itemNext
			items = append(items, value)
		}
		return items, pos, nil
	case 5:
		if arg > math.MaxInt32 {
			return nil, pos, errors.New("cbor map too large")
		}
		items := make(map[interface{}]interface{}, int(arg))
		for i := 0; i < int(arg); i++ {
			key, keyNext, err := p.parse(pos)
			if err != nil {
				return nil, pos, err
			}
			pos = keyNext

			value, valueNext, err := p.parse(pos)
			if err != nil {
				return nil, pos, err
			}
			pos = valueNext
			items[key] = value
		}
		return items, pos, nil
	case 7:
		switch info {
		case 20:
			return false, pos, nil
		case 21:
			return true, pos, nil
		case 22:
			return nil, pos, nil
		default:
			return nil, pos, fmt.Errorf("unsupported cbor simple value %d", info)
		}
	default:
		return nil, pos, fmt.Errorf("unsupported cbor major type %d", major)
	}
}

func passkeyMapBytes(m map[interface{}]interface{}, key interface{}) ([]byte, bool) {
	value, ok := m[key]
	if !ok {
		return nil, false
	}
	bytes, ok := value.([]byte)
	return bytes, ok
}

func passkeyMapInt(m map[interface{}]interface{}, key interface{}) (int64, bool) {
	value, ok := m[key]
	if !ok {
		return 0, false
	}
	n, ok := value.(int64)
	return n, ok
}
