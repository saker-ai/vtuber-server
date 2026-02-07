package audio

// StreamResampler keeps resampling state across frames.
type StreamResampler struct {
	resampler *soxrStreamResampler
	outBuf    []float32
}

// NewStreamResampler creates a streaming resampler for continuous audio.
func NewStreamResampler(inRate, outRate int) (*StreamResampler, error) {
	r, err := newSoxrStreamResampler(inRate, outRate)
	if err != nil {
		return nil, err
	}
	return &StreamResampler{resampler: r}, nil
}

// Close releases underlying resampler.
func (s *StreamResampler) Close() {
	if s == nil {
		return
	}
	if s.resampler != nil {
		s.resampler.Close()
		s.resampler = nil
	}
	s.outBuf = nil
}

// AppendPCM appends PCM16 samples for resampling.
func (s *StreamResampler) AppendPCM(pcm []int16) error {
	if s == nil || s.resampler == nil || len(pcm) == 0 {
		return nil
	}
	tmp := AcquireFloat32(len(pcm))
	tmp = Int16SliceToFloat32Into(tmp, pcm)
	out, err := s.resampler.Process(tmp)
	ReleaseFloat32(tmp)
	if err != nil {
		return err
	}
	if len(out) > 0 {
		s.outBuf = append(s.outBuf, out...)
	}
	return nil
}

// Flush flushes any remaining buffered samples.
func (s *StreamResampler) Flush() error {
	if s == nil || s.resampler == nil {
		return nil
	}
	out, err := s.resampler.Flush()
	if err != nil {
		return err
	}
	if len(out) > 0 {
		s.outBuf = append(s.outBuf, out...)
	}
	return nil
}

// PopFrame returns a fixed-size PCM16 frame if available.
func (s *StreamResampler) PopFrame(frameSize int) ([]int16, bool) {
	if s == nil || frameSize <= 0 || len(s.outBuf) < frameSize {
		return nil, false
	}
	frameFloat := s.outBuf[:frameSize]
	s.outBuf = s.outBuf[frameSize:]
	frame := AcquireInt16(frameSize)
	frame = Float32SliceToInt16SliceInto(frame, frameFloat)
	return frame, true
}

// PopRemainderPadded returns the remaining samples padded to frameSize.
func (s *StreamResampler) PopRemainderPadded(frameSize int) []int16 {
	if s == nil || frameSize <= 0 || len(s.outBuf) == 0 {
		return nil
	}
	if len(s.outBuf) > frameSize {
		s.outBuf = s.outBuf[:frameSize]
	}
	frame := AcquireInt16(frameSize)
	n := len(s.outBuf)
	if n > 0 {
		tmp := frame[:n]
		Float32SliceToInt16SliceInto(tmp, s.outBuf)
	}
	for i := n; i < frameSize; i++ {
		frame[i] = 0
	}
	s.outBuf = nil
	return frame
}
