package codec

import (
	"encoding/binary"
	"errors"
	"time"
)

const (
	// Version1 uses raw audio payload frames.
	Version1 = 1
	// Version2 uses a fixed-width header with payload type and size.
	Version2 = 2
	// Version3 uses a compact fixed-width header with payload type and size.
	Version3 = 3

	payloadTypeAudio = 0
	payloadTypeCmd   = 1
)

// PayloadKind describes the decoded payload category.
type PayloadKind int

const (
	// PayloadKindAudio indicates audio bytes.
	PayloadKindAudio PayloadKind = iota
	// PayloadKindCommand indicates JSON command bytes.
	PayloadKindCommand
)

// NormalizeVersion returns a supported protocol version.
func NormalizeVersion(version int) int {
	switch version {
	case Version2, Version3:
		return version
	default:
		return Version1
	}
}

// Decode parses a binary frame according to protocol version.
func Decode(version int, frame []byte) ([]byte, PayloadKind, error) {
	switch NormalizeVersion(version) {
	case Version2:
		return decodeV2(frame)
	case Version3:
		return decodeV3(frame)
	default:
		return frame, PayloadKindAudio, nil
	}
}

// Pack creates a binary frame according to protocol version.
func Pack(version int, payload []byte) []byte {
	switch NormalizeVersion(version) {
	case Version2:
		return packV2(payload)
	case Version3:
		return packV3(payload)
	default:
		return payload
	}
}

func decodeV2(frame []byte) ([]byte, PayloadKind, error) {
	const headerSize = 16
	if len(frame) < headerSize {
		return nil, PayloadKindAudio, errors.New("xiaozhi binary v2 frame too short")
	}
	msgType := binary.BigEndian.Uint16(frame[2:4])
	payloadSize := binary.BigEndian.Uint32(frame[12:16])
	if int(payloadSize) > len(frame)-headerSize {
		return nil, PayloadKindAudio, errors.New("xiaozhi binary v2 invalid payload size")
	}
	payload := frame[headerSize : headerSize+int(payloadSize)]
	switch msgType {
	case payloadTypeAudio:
		return payload, PayloadKindAudio, nil
	case payloadTypeCmd:
		return payload, PayloadKindCommand, nil
	default:
		return nil, PayloadKindAudio, errors.New("xiaozhi binary v2 unsupported payload type")
	}
}

func decodeV3(frame []byte) ([]byte, PayloadKind, error) {
	const headerSize = 4
	if len(frame) < headerSize {
		return nil, PayloadKindAudio, errors.New("xiaozhi binary v3 frame too short")
	}
	msgType := frame[0]
	payloadSize := binary.BigEndian.Uint16(frame[2:4])
	if int(payloadSize) > len(frame)-headerSize {
		return nil, PayloadKindAudio, errors.New("xiaozhi binary v3 invalid payload size")
	}
	payload := frame[headerSize : headerSize+int(payloadSize)]
	switch msgType {
	case payloadTypeAudio:
		return payload, PayloadKindAudio, nil
	case payloadTypeCmd:
		return payload, PayloadKindCommand, nil
	default:
		return nil, PayloadKindAudio, errors.New("xiaozhi binary v3 unsupported payload type")
	}
}

func packV2(payload []byte) []byte {
	const headerSize = 16
	head := make([]byte, headerSize)
	binary.BigEndian.PutUint16(head[0:2], Version2)
	binary.BigEndian.PutUint16(head[2:4], payloadTypeAudio)
	binary.BigEndian.PutUint32(head[4:8], 0)
	binary.BigEndian.PutUint32(head[8:12], uint32(time.Now().UnixMilli()))
	binary.BigEndian.PutUint32(head[12:16], uint32(len(payload)))
	return append(head, payload...)
}

func packV3(payload []byte) []byte {
	head := make([]byte, 4)
	head[0] = payloadTypeAudio
	head[1] = 0
	binary.BigEndian.PutUint16(head[2:4], uint16(len(payload)))
	return append(head, payload...)
}
