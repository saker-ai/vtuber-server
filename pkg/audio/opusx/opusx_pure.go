//go:build !cgo

package opusx

import "github.com/godeps/opus"

func Backend() string {
	return "pure-godeps/opus"
}

type Application = opus.Application

type Bandwidth = opus.Bandwidth

const (
	AppVoIP               = opus.AppVoIP
	AppAudio              = opus.AppAudio
	AppRestrictedLowdelay = opus.AppRestrictedLowdelay
)

var (
	Narrowband    = opus.Narrowband
	Mediumband    = opus.Mediumband
	Wideband      = opus.Wideband
	SuperWideband = opus.SuperWideband
	Fullband      = opus.Fullband
)

type Encoder struct {
	enc *opus.Encoder
}

type Decoder struct {
	dec *opus.Decoder
}

func NewEncoder(sampleRate, channels int, app Application) (*Encoder, error) {
	enc, err := opus.NewEncoder(sampleRate, channels, app)
	if err != nil {
		return nil, err
	}
	return &Encoder{enc: enc}, nil
}

func NewDecoder(sampleRate, channels int) (*Decoder, error) {
	dec, err := opus.NewDecoder(sampleRate, channels)
	if err != nil {
		return nil, err
	}
	return &Decoder{dec: dec}, nil
}

func (e *Encoder) Encode(pcm []int16, data []byte) (int, error) {
	return e.enc.Encode(pcm, data)
}

func (d *Decoder) Decode(data []byte, pcm []int16) (int, error) {
	return d.dec.Decode(data, pcm)
}

func (d *Decoder) DecodeFloat32(data []byte, pcm []float32) (int, error) {
	return d.dec.DecodeFloat32(data, pcm)
}

func (e *Encoder) Reset() error {
	return e.enc.Reset()
}

func (e *Encoder) SetDTX(dtx bool) error {
	return e.enc.SetDTX(dtx)
}

func (e *Encoder) SetBitrate(bitrate int) error {
	return e.enc.SetBitrate(bitrate)
}

func (e *Encoder) SetComplexity(complexity int) error {
	return e.enc.SetComplexity(complexity)
}

func (e *Encoder) SetMaxBandwidth(maxBw Bandwidth) error {
	return e.enc.SetMaxBandwidth(maxBw)
}

func (e *Encoder) SetInBandFEC(fec bool) error {
	return e.enc.SetInBandFEC(fec)
}

func (e *Encoder) SetPacketLossPerc(lossPerc int) error {
	return e.enc.SetPacketLossPerc(lossPerc)
}

func (e *Encoder) SetVBR(vbr bool) error {
	return e.enc.SetVBR(vbr)
}

func (e *Encoder) SetVBRConstraint(constraint bool) error {
	return e.enc.SetVBRConstraint(constraint)
}
