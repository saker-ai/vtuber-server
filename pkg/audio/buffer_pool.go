package audio

import "sync"

var bytesPool sync.Pool
var int16Pool sync.Pool
var float32Pool sync.Pool

// AcquireBytes returns a byte slice with length size.
func AcquireBytes(size int) []byte {
	if size <= 0 {
		return nil
	}
	if v := bytesPool.Get(); v != nil {
		buf := v.([]byte)
		if cap(buf) >= size {
			return buf[:size]
		}
	}
	return make([]byte, size)
}

// ReleaseBytes puts a byte slice back to the pool.
func ReleaseBytes(buf []byte) {
	if buf == nil {
		return
	}
	bytesPool.Put(buf[:0])
}

// AcquireInt16 returns an int16 slice with length size.
func AcquireInt16(size int) []int16 {
	if size <= 0 {
		return nil
	}
	if v := int16Pool.Get(); v != nil {
		buf := v.([]int16)
		if cap(buf) >= size {
			return buf[:size]
		}
	}
	return make([]int16, size)
}

// ReleaseInt16 puts an int16 slice back to the pool.
func ReleaseInt16(buf []int16) {
	if buf == nil {
		return
	}
	int16Pool.Put(buf[:0])
}

// AcquireFloat32 returns a float32 slice with length size.
func AcquireFloat32(size int) []float32 {
	if size <= 0 {
		return nil
	}
	if v := float32Pool.Get(); v != nil {
		buf := v.([]float32)
		if cap(buf) >= size {
			return buf[:size]
		}
	}
	return make([]float32, size)
}

// ReleaseFloat32 puts a float32 slice back to the pool.
func ReleaseFloat32(buf []float32) {
	if buf == nil {
		return
	}
	float32Pool.Put(buf[:0])
}
