package audio

import (
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/saker-ai/vtuber-server/pkg/audio/opusx"
)

type opusEncodeOptions struct {
	bitrate        int
	complexity     int
	vbr            *bool
	vbrConstraint  *bool
	fec            *bool
	dtx            *bool
	packetLossPerc int
	maxBandwidth   string
}

var (
	opusOptionsOnce sync.Once
	opusOptions     opusEncodeOptions
	opusLogOnce     sync.Once
)

func getOpusEncodeOptions() opusEncodeOptions {
	opusOptionsOnce.Do(func() {
		opusOptions = opusEncodeOptions{
			bitrate:        getenvInt("OPUS_BITRATE", 0),
			complexity:     getenvInt("OPUS_COMPLEXITY", 0),
			vbr:            getenvBoolPtr("OPUS_VBR"),
			vbrConstraint:  getenvBoolPtr("OPUS_VBR_CONSTRAINT"),
			fec:            getenvBoolPtr("OPUS_FEC"),
			dtx:            getenvBoolPtr("OPUS_DTX"),
			packetLossPerc: getenvInt("OPUS_PACKET_LOSS_PERC", 0),
			maxBandwidth:   strings.ToLower(strings.TrimSpace(os.Getenv("OPUS_MAX_BANDWIDTH"))),
		}
	})
	return opusOptions
}

func applyOpusEncoderOptions(enc *opusx.Encoder) {
	if enc == nil {
		return
	}
	opts := getOpusEncodeOptions()

	if opts.bitrate > 0 {
		_ = enc.SetBitrate(opts.bitrate)
	}
	if opts.complexity > 0 {
		_ = enc.SetComplexity(opts.complexity)
	}
	if opts.vbr != nil {
		_ = enc.SetVBR(*opts.vbr)
	}
	if opts.vbrConstraint != nil {
		_ = enc.SetVBRConstraint(*opts.vbrConstraint)
	}
	if opts.fec != nil {
		_ = enc.SetInBandFEC(*opts.fec)
	}
	if opts.dtx != nil {
		_ = enc.SetDTX(*opts.dtx)
	}
	if opts.packetLossPerc > 0 {
		_ = enc.SetPacketLossPerc(opts.packetLossPerc)
	}
	if bw := parseOpusBandwidth(opts.maxBandwidth); bw != nil {
		_ = enc.SetMaxBandwidth(*bw)
	}

	opusLogOnce.Do(func() {
		log.Printf(
			"Opus encoder options: bitrate=%d complexity=%d vbr=%s vbr_constraint=%s fec=%s dtx=%s packet_loss=%d max_bw=%s",
			opts.bitrate,
			opts.complexity,
			boolPtrString(opts.vbr),
			boolPtrString(opts.vbrConstraint),
			boolPtrString(opts.fec),
			boolPtrString(opts.dtx),
			opts.packetLossPerc,
			opts.maxBandwidth,
		)
	})
}

func LogOpusBackend() {
	opusLogOnce.Do(func() {
		log.Printf("Opus backend: %s", opusx.Backend())
	})
}

func parseOpusBandwidth(v string) *opusx.Bandwidth {
	switch v {
	case "", "auto":
		return nil
	case "narrowband", "nb":
		bw := opusx.Narrowband
		return &bw
	case "mediumband", "mb":
		bw := opusx.Mediumband
		return &bw
	case "wideband", "wb":
		bw := opusx.Wideband
		return &bw
	case "superwideband", "swb":
		bw := opusx.SuperWideband
		return &bw
	case "fullband", "fb":
		bw := opusx.Fullband
		return &bw
	default:
		return nil
	}
}

func getenvInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getenvBoolPtr(key string) *bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return nil
	}
	return &b
}

func boolPtrString(v *bool) string {
	if v == nil {
		return "unset"
	}
	if *v {
		return "true"
	}
	return "false"
}
