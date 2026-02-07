package audio

import (
	"sync"

	"github.com/saker-ai/vtuber-server/pkg/audio/opusx"
)

type opusRawKey struct {
	sampleRate int
	channels   int
}

var opusRawEncoderPools sync.Map

func getOpusRawEncoderPool(key opusRawKey) *sync.Pool {
	if pool, ok := opusRawEncoderPools.Load(key); ok {
		return pool.(*sync.Pool)
	}
	pool := &sync.Pool{}
	actual, _ := opusRawEncoderPools.LoadOrStore(key, pool)
	return actual.(*sync.Pool)
}

func acquireRawOpusEncoder(sampleRate, channels int) (*opusx.Encoder, error) {
	key := opusRawKey{sampleRate: sampleRate, channels: channels}
	pool := getOpusRawEncoderPool(key)
	if v := pool.Get(); v != nil {
		if enc, ok := v.(*opusx.Encoder); ok && enc != nil {
			return enc, nil
		}
	}
	return opusx.NewEncoder(sampleRate, channels, opusx.AppAudio)
}

func releaseRawOpusEncoder(sampleRate, channels int, enc *opusx.Encoder) {
	if enc == nil {
		return
	}
	if err := enc.Reset(); err != nil {
		// Ignore reset errors; encoder will be reused or recreated later.
	}
	key := opusRawKey{sampleRate: sampleRate, channels: channels}
	getOpusRawEncoderPool(key).Put(enc)
}
