package audio

import (
	"errors"
	"sync"

	resampler "github.com/godeps/go-audio-soxr"
)

type soxrKey struct {
	inRate  int
	outRate int
	quality resampler.QualityPreset
}

type soxrStreamResampler struct {
	inRate  int
	outRate int
	quality resampler.QualityPreset
	r       *resampler.SimpleResamplerFloat32
}

var soxrPools sync.Map

func getSoxrPool(key soxrKey) *sync.Pool {
	if pool, ok := soxrPools.Load(key); ok {
		return pool.(*sync.Pool)
	}
	pool := &sync.Pool{}
	actual, _ := soxrPools.LoadOrStore(key, pool)
	return actual.(*sync.Pool)
}

func acquireSoxrResampler(inRate, outRate int, quality resampler.QualityPreset) (*resampler.SimpleResamplerFloat32, error) {
	key := soxrKey{inRate: inRate, outRate: outRate, quality: quality}
	pool := getSoxrPool(key)
	if v := pool.Get(); v != nil {
		if r, ok := v.(*resampler.SimpleResamplerFloat32); ok && r != nil {
			return r, nil
		}
	}
	return resampler.NewEngineFloat32(float64(inRate), float64(outRate), quality)
}

func releaseSoxrResampler(inRate, outRate int, quality resampler.QualityPreset, r *resampler.SimpleResamplerFloat32) {
	if r == nil {
		return
	}
	r.Reset()
	key := soxrKey{inRate: inRate, outRate: outRate, quality: quality}
	getSoxrPool(key).Put(r)
}

func newSoxrStreamResampler(inRate, outRate int) (*soxrStreamResampler, error) {
	r, err := acquireSoxrResampler(inRate, outRate, resampler.QualityHigh)
	if err != nil {
		return nil, err
	}
	return &soxrStreamResampler{
		inRate:  inRate,
		outRate: outRate,
		quality: resampler.QualityHigh,
		r:       r,
	}, nil
}

func (s *soxrStreamResampler) Process(input []float32) ([]float32, error) {
	if s == nil || s.r == nil {
		return nil, errors.New("soxr resampler is nil")
	}
	return s.r.Process(input)
}

func (s *soxrStreamResampler) Flush() ([]float32, error) {
	if s == nil || s.r == nil {
		return nil, errors.New("soxr resampler is nil")
	}
	return s.r.Flush()
}

func (s *soxrStreamResampler) Close() {
	if s == nil || s.r == nil {
		return
	}
	releaseSoxrResampler(s.inRate, s.outRate, s.quality, s.r)
	s.r = nil
}
