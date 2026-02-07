package audio

import "sync"

type opusEncoderKey struct {
	sampleRate    int
	channels      int
	frameDuration int
}

var opusEncoderPools sync.Map

func getOpusEncoderPool(key opusEncoderKey) *sync.Pool {
	if pool, ok := opusEncoderPools.Load(key); ok {
		return pool.(*sync.Pool)
	}
	pool := &sync.Pool{}
	actual, _ := opusEncoderPools.LoadOrStore(key, pool)
	return actual.(*sync.Pool)
}

// AcquireOpusEncoder reuses encoders keyed by sampleRate/channels/frameDuration.
func AcquireOpusEncoder(sampleRate, channels, frameDurationMs int) (*OpusEncoder, error) {
	key := opusEncoderKey{
		sampleRate:    sampleRate,
		channels:      channels,
		frameDuration: frameDurationMs,
	}
	pool := getOpusEncoderPool(key)
	if v := pool.Get(); v != nil {
		enc := v.(*OpusEncoder)
		if enc.encoder != nil {
			return enc, nil
		}
	}
	return NewOpusEncoder(sampleRate, channels, frameDurationMs)
}

// ReleaseOpusEncoder returns encoder to pool for reuse.
func ReleaseOpusEncoder(enc *OpusEncoder) {
	if enc == nil {
		return
	}
	enc.mutex.Lock()
	if enc.encoder != nil {
		_ = enc.encoder.Reset()
	}
	enc.mutex.Unlock()
	key := opusEncoderKey{
		sampleRate:    enc.sampleRate,
		channels:      enc.channels,
		frameDuration: enc.frameDuration,
	}
	getOpusEncoderPool(key).Put(enc)
}
