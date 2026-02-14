package codec

import (
	"encoding/binary"
	"testing"
)

func TestPackDecodeV2Audio(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03, 0x04}
	frame := Pack(Version2, payload)

	got, kind, err := Decode(Version2, frame)
	if err != nil {
		t.Fatalf("Decode(v2) returned error: %v", err)
	}
	if kind != PayloadKindAudio {
		t.Fatalf("Decode(v2) kind=%v, want %v", kind, PayloadKindAudio)
	}
	if string(got) != string(payload) {
		t.Fatalf("Decode(v2) payload=%v, want %v", got, payload)
	}
}

func TestPackDecodeV3Audio(t *testing.T) {
	payload := []byte{0x09, 0x08, 0x07}
	frame := Pack(Version3, payload)

	got, kind, err := Decode(Version3, frame)
	if err != nil {
		t.Fatalf("Decode(v3) returned error: %v", err)
	}
	if kind != PayloadKindAudio {
		t.Fatalf("Decode(v3) kind=%v, want %v", kind, PayloadKindAudio)
	}
	if string(got) != string(payload) {
		t.Fatalf("Decode(v3) payload=%v, want %v", got, payload)
	}
}

func TestDecodeV2CommandPayload(t *testing.T) {
	payload := []byte(`{"type":"hello"}`)
	frame := make([]byte, 16+len(payload))
	binary.BigEndian.PutUint16(frame[0:2], Version2)
	binary.BigEndian.PutUint16(frame[2:4], payloadTypeCmd)
	binary.BigEndian.PutUint32(frame[12:16], uint32(len(payload)))
	copy(frame[16:], payload)

	got, kind, err := Decode(Version2, frame)
	if err != nil {
		t.Fatalf("Decode(v2 cmd) returned error: %v", err)
	}
	if kind != PayloadKindCommand {
		t.Fatalf("Decode(v2 cmd) kind=%v, want %v", kind, PayloadKindCommand)
	}
	if string(got) != string(payload) {
		t.Fatalf("Decode(v2 cmd) payload=%q, want %q", string(got), string(payload))
	}
}

func TestDecodeV2InvalidPayloadSize(t *testing.T) {
	frame := make([]byte, 16)
	binary.BigEndian.PutUint16(frame[0:2], Version2)
	binary.BigEndian.PutUint16(frame[2:4], payloadTypeAudio)
	binary.BigEndian.PutUint32(frame[12:16], 10)

	_, _, err := Decode(Version2, frame)
	if err == nil {
		t.Fatal("Decode(v2) error=nil, want non-nil")
	}
}
