package xiaozhi

import (
	"testing"

	xzcodec "github.com/saker-ai/vtuber-server/internal/transport/xiaozhi/codec"
)

func TestPackDecodeBinaryProtocol2Audio(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03, 0x04}

	frame := xzcodec.Pack(xzcodec.Version2, payload)
	got, kind, err := xzcodec.Decode(xzcodec.Version2, frame)
	if err != nil {
		t.Fatalf("Decode(v2) returned error: %v", err)
	}
	if kind != xzcodec.PayloadKindAudio {
		t.Fatalf("Decode(v2) kind=%v, want %v", kind, xzcodec.PayloadKindAudio)
	}
	if string(got) != string(payload) {
		t.Fatalf("Decode(v2) payload=%v, want %v", got, payload)
	}
}

func TestPackDecodeBinaryProtocol3Audio(t *testing.T) {
	payload := []byte{0x09, 0x08, 0x07}

	frame := xzcodec.Pack(xzcodec.Version3, payload)
	got, kind, err := xzcodec.Decode(xzcodec.Version3, frame)
	if err != nil {
		t.Fatalf("Decode(v3) returned error: %v", err)
	}
	if kind != xzcodec.PayloadKindAudio {
		t.Fatalf("Decode(v3) kind=%v, want %v", kind, xzcodec.PayloadKindAudio)
	}
	if string(got) != string(payload) {
		t.Fatalf("Decode(v3) payload=%v, want %v", got, payload)
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
