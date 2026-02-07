package audio

import "math"

func float32ToInt16(sample float32) int16 {
	if sample > 1.0 {
		return 32767
	}
	if sample < -1.0 {
		return -32768
	}
	return int16(sample * 32767)
}

// Float32SliceToInt16SliceInto fills dst with float32 converted to int16 and returns the slice.
func Float32SliceToInt16SliceInto(dst []int16, samples []float32) []int16 {
	if cap(dst) < len(samples) {
		dst = make([]int16, len(samples))
	} else {
		dst = dst[:len(samples)]
	}
	for i, sample := range samples {
		dst[i] = float32ToInt16(sample)
	}
	return dst
}

// Int16SliceToFloat32Into fills dst with int16 converted to float32 and returns the slice.
func Int16SliceToFloat32Into(dst []float32, samples []int16) []float32 {
	if cap(dst) < len(samples) {
		dst = make([]float32, len(samples))
	} else {
		dst = dst[:len(samples)]
	}
	for i, sample := range samples {
		dst[i] = float32(sample) / float32(math.MaxInt16)
	}
	return dst
}

// Int16SliceToBytesInto converts int16 samples to little-endian bytes.
func Int16SliceToBytesInto(dst []byte, samples []int16) []byte {
	needed := len(samples) * 2
	if cap(dst) < needed {
		dst = make([]byte, needed)
	} else {
		dst = dst[:needed]
	}
	for i, sample := range samples {
		offset := i * 2
		dst[offset] = byte(sample)
		dst[offset+1] = byte(sample >> 8)
	}
	return dst
}
