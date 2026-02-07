package audio

import (
	"fmt"
	"sync"

	"github.com/saker-ai/vtuber-server/pkg/audio/opusx"
)

// OpusEncoder represents a opusEncoder.
type OpusEncoder struct {
	encoder       *opusx.Encoder
	sampleRate    int
	channels      int
	frameDuration int
	frameSize     int
	opusBuffer    []byte
	mutex         sync.Mutex
}

// NewOpusEncoder executes the newOpusEncoder function.
func NewOpusEncoder(sampleRate, channels, frameDurationMs int) (*OpusEncoder, error) {
	enc, err := acquireRawOpusEncoder(sampleRate, channels)
	if err != nil {
		return nil, fmt.Errorf("create opus encoder: %v", err)
	}
	applyOpusEncoderOptions(enc)

	frameSize := sampleRate * frameDurationMs / 1000
	opusBuffer := make([]byte, 4000)

	return &OpusEncoder{
		encoder:       enc,
		sampleRate:    sampleRate,
		channels:      channels,
		frameDuration: frameDurationMs,
		frameSize:     frameSize,
		opusBuffer:    opusBuffer,
	}, nil
}

// Encode executes the encode method.
func (e *OpusEncoder) Encode(pcmData []byte) ([]byte, error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	pcmSamples := BytesToInt16Slice(pcmData)

	expectedSamples := e.frameSize * e.channels
	if len(pcmSamples) < expectedSamples {
		paddedSamples := make([]int16, expectedSamples)
		copy(paddedSamples, pcmSamples)
		pcmSamples = paddedSamples
	} else if len(pcmSamples) > expectedSamples {
		pcmSamples = pcmSamples[:expectedSamples]
	}

	n, err := e.encoder.Encode(pcmSamples, e.opusBuffer)
	if err != nil {
		return nil, fmt.Errorf("opus encode: %v", err)
	}

	if n == 0 {
		return nil, nil
	}

	result := make([]byte, n)
	copy(result, e.opusBuffer[:n])

	return result, nil
}

// EncodeWithScratch encodes PCM bytes using a provided scratch buffer to reduce allocations.
func (e *OpusEncoder) EncodeWithScratch(pcmData []byte, scratch []int16) ([]byte, error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	expectedSamples := e.frameSize * e.channels
	pcmSamples := BytesToInt16SliceInto(scratch, pcmData)

	if len(pcmSamples) < expectedSamples {
		if cap(pcmSamples) < expectedSamples {
			tmp := make([]int16, expectedSamples)
			copy(tmp, pcmSamples)
			pcmSamples = tmp
		} else {
			origLen := len(pcmSamples)
			pcmSamples = pcmSamples[:expectedSamples]
			for i := origLen; i < expectedSamples; i++ {
				pcmSamples[i] = 0
			}
		}
	} else if len(pcmSamples) > expectedSamples {
		pcmSamples = pcmSamples[:expectedSamples]
	}

	n, err := e.encoder.Encode(pcmSamples, e.opusBuffer)
	if err != nil {
		return nil, fmt.Errorf("opus encode: %v", err)
	}

	if n == 0 {
		return nil, nil
	}

	result := make([]byte, n)
	copy(result, e.opusBuffer[:n])
	return result, nil
}

// SetBitrate executes the setBitrate method.
func (e *OpusEncoder) SetBitrate(bitrate int) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	return e.encoder.SetBitrate(bitrate)
}

// Close executes the close method.
func (e *OpusEncoder) Close() error {
	if e.encoder != nil {
		releaseRawOpusEncoder(e.sampleRate, e.channels, e.encoder)
	}
	e.encoder = nil
	e.opusBuffer = nil
	return nil
}

// GetFrameSize executes the getFrameSize method.
func (e *OpusEncoder) GetFrameSize() int {
	return e.frameSize
}

// GetFrameDuration executes the getFrameDuration method.
func (e *OpusEncoder) GetFrameDuration() int {
	return e.frameDuration
}

// GetFrameBytes executes the getFrameBytes method.
func (e *OpusEncoder) GetFrameBytes() int {
	return e.frameSize * e.channels * 2
}

// BytesToInt16Slice executes the bytesToInt16Slice function.
func BytesToInt16Slice(data []byte) []int16 {
	if len(data)%2 != 0 {
		tmp := make([]byte, len(data)+1)
		copy(tmp, data)
		data = tmp
	}

	result := make([]int16, len(data)/2)
	for i := 0; i < len(result); i++ {
		result[i] = int16(data[i*2]) | int16(data[i*2+1])<<8
	}
	return result
}

// BytesToInt16SliceInto fills dst with little-endian int16 samples and returns it.
func BytesToInt16SliceInto(dst []int16, data []byte) []int16 {
	needed := (len(data) + 1) / 2
	if cap(dst) < needed {
		dst = make([]int16, needed)
	} else {
		dst = dst[:needed]
	}
	for i := 0; i < needed; i++ {
		low := data[i*2]
		high := byte(0)
		if i*2+1 < len(data) {
			high = data[i*2+1]
		}
		dst[i] = int16(low) | int16(high)<<8
	}
	return dst
}
