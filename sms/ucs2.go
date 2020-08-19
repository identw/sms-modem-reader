package sms

import (
	"errors"
	"unicode/utf16"
)

var (
	asciiHex = map[byte]byte {
		48: 0x0,
		49: 0x1,
		50: 0x2,
		51: 0x3,
		52: 0x4,
		53: 0x5,
		54: 0x6,
		55: 0x7,
		56: 0x8,
		57: 0x9,
		65: 0xA,
		66: 0xB,
		67: 0xC,
		68: 0xD,
		69: 0xE,
		70: 0xF,
	}
)

// ErrUnevenNumber happens when the number of octets (bytes) in the input is uneven.
var ErrUnevenNumber = errors.New("decode ucs2: uneven number of octets")

// ErrIncorrectDataLength happens when the length of octets is less than the header, defined by the first entry of the octets
var ErrIncorrectDataLength = errors.New("decode ucs2: incorrect data length in first entry of octets")

// EncodeUcs2 encodes the given UTF-8 text into UCS2 (UTF-16) encoding and returns the produced octets.
func EncodeUcs2(str string) []byte {
	buf := utf16.Encode([]rune(str))
	octets := make([]byte, 0, len(buf)*2)
	for _, n := range buf {
		octets = append(octets, byte(n&0xFF00>>8), byte(n&0x00FF))
	}
	return octets
}

// DecodeUcs2 decodes the given UCS2 (UTF-16) octet data into a UTF-8 encoded string.
func DecodeUcs2(octets []byte, startsWithHeader bool) (str string, err error) {
	octetsLng := len(octets)
	headerLng := 0

	if octetsLng == 0 {
		err = ErrIncorrectDataLength
		return
	}

	if startsWithHeader {
		// just ignore header
		headerLng = int(octets[0]) + 1
		if (octetsLng - headerLng) <= 0 {
			err = ErrIncorrectDataLength
			return
		}

		octetsLng = octetsLng - headerLng
	}

	if octetsLng%2 != 0 {
		err = ErrUnevenNumber
		return
	}
	buf := make([]uint16, 0, octetsLng/2)
	for i := 0; i < octetsLng; i += 2 {
		buf = append(buf, uint16(octets[i+headerLng])<<8|uint16(octets[i+1+headerLng]))
	}
	runes := utf16.Decode(buf)
	return string(runes), nil
}

func bytesToHex(b []byte) []byte {
	output := make([]byte, 0)
	for i := 0; i < len(b); i += 2 {
		if asciiHex[b[i]] == 0x0 {
			output = append(output, asciiHex[b[i+1]])
		} else {
			output = append(output, asciiHex[b[i]] << 4 + asciiHex[b[i+1]])
		}	
	}
	return output
}