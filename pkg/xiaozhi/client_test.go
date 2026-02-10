package xiaozhi

import (
	"encoding/binary"
	"testing"
)

func TestPackDecodeBinaryProtocol2Audio(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03, 0x04}

	frame := packBinaryProtocol2(payload)
	got, kind, err := decodeBinaryProtocol2(frame)
	if err != nil {
		t.Fatalf("decodeBinaryProtocol2 returned error: %v", err)
	}
	if kind != binaryPayloadKindAudio {
		t.Fatalf("decodeBinaryProtocol2 kind=%v, want %v", kind, binaryPayloadKindAudio)
	}
	if string(got) != string(payload) {
		t.Fatalf("decodeBinaryProtocol2 payload=%v, want %v", got, payload)
	}
}

func TestPackDecodeBinaryProtocol3Audio(t *testing.T) {
	payload := []byte{0x09, 0x08, 0x07}

	frame := packBinaryProtocol3(payload)
	got, kind, err := decodeBinaryProtocol3(frame)
	if err != nil {
		t.Fatalf("decodeBinaryProtocol3 returned error: %v", err)
	}
	if kind != binaryPayloadKindAudio {
		t.Fatalf("decodeBinaryProtocol3 kind=%v, want %v", kind, binaryPayloadKindAudio)
	}
	if string(got) != string(payload) {
		t.Fatalf("decodeBinaryProtocol3 payload=%v, want %v", got, payload)
	}
}

func TestDecodeBinaryProtocol2CmdPayload(t *testing.T) {
	payload := []byte(`{"type":"hello"}`)
	frame := make([]byte, 16+len(payload))
	binary.BigEndian.PutUint16(frame[0:2], protocolV2)
	binary.BigEndian.PutUint16(frame[2:4], binaryPayloadTypeCmd)
	binary.BigEndian.PutUint32(frame[12:16], uint32(len(payload)))
	copy(frame[16:], payload)

	got, kind, err := decodeBinaryProtocol2(frame)
	if err != nil {
		t.Fatalf("decodeBinaryProtocol2 returned error: %v", err)
	}
	if kind != binaryPayloadKindCmd {
		t.Fatalf("decodeBinaryProtocol2 kind=%v, want %v", kind, binaryPayloadKindCmd)
	}
	if string(got) != string(payload) {
		t.Fatalf("decodeBinaryProtocol2 payload=%q, want %q", string(got), string(payload))
	}
}

func TestDecodeBinaryProtocol2InvalidPayloadSize(t *testing.T) {
	frame := make([]byte, 16)
	binary.BigEndian.PutUint16(frame[0:2], protocolV2)
	binary.BigEndian.PutUint16(frame[2:4], binaryPayloadTypeAudio)
	binary.BigEndian.PutUint32(frame[12:16], 10)

	_, _, err := decodeBinaryProtocol2(frame)
	if err == nil {
		t.Fatal("decodeBinaryProtocol2 error=nil, want non-nil")
	}
}

func TestInitialDownstreamAudioUsesOutputFormat(t *testing.T) {
	params := AudioParams{
		Format:       "opus",
		OutputFormat: "wav",
		SampleRate:   16000,
		Channels:     1,
	}

	downstream := initialDownstreamAudio(params)
	if downstream.Format != "wav" {
		t.Fatalf("downstream format=%q, want %q", downstream.Format, "wav")
	}
	if downstream.OutputFormat != "" {
		t.Fatalf("downstream output_format=%q, want empty", downstream.OutputFormat)
	}
}

func TestNormalizeListenMode(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "auto", want: "auto"},
		{in: " MANUAL ", want: "manual"},
		{in: "Realtime", want: "realtime"},
		{in: "", want: "auto"},
		{in: "invalid", want: "auto"},
	}
	for _, tt := range tests {
		if got := normalizeListenMode(tt.in); got != tt.want {
			t.Fatalf("normalizeListenMode(%q)=%q, want %q", tt.in, got, tt.want)
		}
	}
}
