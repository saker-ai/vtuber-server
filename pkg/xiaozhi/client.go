package xiaozhi

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	godepsopus "github.com/godeps/opus"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	xzcodec "github.com/saker-ai/vtuber-server/internal/transport/xiaozhi/codec"
)

const (
	opusMaxFrameDurationMs = 120
)

// AudioFrame represents a audioFrame.
type AudioFrame struct {
	PCM        []byte
	SampleRate int
	Channels   int
}

// Callbacks represents a callbacks.
type Callbacks struct {
	OnSTT          func(text string)
	OnLLM          func(text string, state string)
	OnText         func(text string)
	OnTTS          func(state string, text string)
	OnMCP          func(payload json.RawMessage)
	OnGoodbye      func()
	OnAudio        func(frame AudioFrame)
	OnConnected    func()
	OnDisconnected func(err error)
	OnError        func(err error)
}

// Client represents a client.
type Client struct {
	cfg       Config
	logger    *zap.Logger
	callbacks Callbacks

	mu sync.Mutex

	conn      *websocket.Conn
	closed    bool
	listen    string
	sessionID string

	protocolVersion int
	helloReady      bool

	downstream AudioParams
	decoder    *godepsopus.Decoder
	decoderSR  int
	decoderCH  int

	encoder *godepsopus.Encoder
	writeMu sync.Mutex
}

// NewClient executes the newClient function.
func NewClient(cfg Config, callbacks Callbacks, logger *zap.Logger) *Client {
	if logger == nil {
		logger = zap.NewNop()
	}

	cfg.ProtocolVersion = normalizeProtocolVersion(cfg.ProtocolVersion)
	cfg.AudioParams = normalizeAudioParams(cfg.AudioParams)
	cfg.ListenMode = normalizeListenMode(cfg.ListenMode)

	downstream := initialDownstreamAudio(cfg.AudioParams)

	var encoder *godepsopus.Encoder
	if cfg.AudioParams.Format == "opus" {
		enc, err := godepsopus.NewEncoder(cfg.AudioParams.SampleRate, cfg.AudioParams.Channels, godepsopus.AppAudio)
		if err != nil {
			logger.Warn("opus encoder init failed", zap.Error(err))
		} else {
			encoder = enc
		}
	}

	client := &Client{
		cfg:             cfg,
		logger:          logger,
		callbacks:       callbacks,
		listen:          cfg.ListenMode,
		protocolVersion: cfg.ProtocolVersion,
		downstream:      downstream,
		encoder:         encoder,
	}
	if downstream.Format == "opus" {
		if err := client.ensureDecoderLocked(downstream.SampleRate, downstream.Channels); err != nil {
			logger.Warn("opus decoder init failed", zap.Error(err))
		}
	}
	return client
}

// Connect executes the connect method.
func (c *Client) Connect(ctx context.Context) {
	go c.run(ctx)
}

// Close executes the close method.
func (c *Client) Close() {
	c.mu.Lock()
	c.closed = true
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
	c.mu.Unlock()
}

// SendTextInput executes the sendTextInput method.
func (c *Client) SendTextInput(ctx context.Context, text string) error {
	if err := c.waitHelloReady(ctx); err != nil {
		return err
	}
	payload := map[string]any{
		"type":      "listen",
		"state":     "detect",
		"mode":      c.listenMode(),
		"text":      text,
		"device_id": c.cfg.DeviceID,
	}
	c.attachSessionID(payload)
	return c.sendJSON(ctx, payload)
}

// SendListenState executes the sendListenState method.
func (c *Client) SendListenState(ctx context.Context, state string) error {
	if err := c.waitHelloReady(ctx); err != nil {
		return err
	}
	payload := map[string]any{
		"type":      "listen",
		"state":     state,
		"mode":      c.listenMode(),
		"device_id": c.cfg.DeviceID,
	}
	c.attachSessionID(payload)
	return c.sendJSON(ctx, payload)
}

// Abort executes the abort method.
func (c *Client) Abort(ctx context.Context) error {
	if err := c.waitHelloReady(ctx); err != nil {
		return err
	}
	payload := map[string]any{
		"type":   "abort",
		"reason": "user_interrupt",
	}
	c.attachSessionID(payload)
	return c.sendJSON(ctx, payload)
}

// SendAudio executes the sendAudio method.
func (c *Client) SendAudio(ctx context.Context, audio []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.waitHelloReady(ctx); err != nil {
		return err
	}

	c.mu.Lock()
	conn := c.conn
	version := c.protocolVersion
	c.mu.Unlock()
	if conn == nil {
		return errors.New("xiaozhi connection not ready")
	}

	frame := xzcodec.Pack(version, audio)

	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if err := conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
		return err
	}
	return nil
}

// EncodeOpusFloat executes the encodeOpusFloat method.
func (c *Client) EncodeOpusFloat(pcm []float32) ([]byte, error) {
	if c.encoder == nil {
		return nil, errors.New("opus encoder is not initialized")
	}
	if len(pcm) == 0 {
		return nil, errors.New("empty pcm frame")
	}
	out := make([]byte, 4000)
	written, err := c.encoder.EncodeFloat32(pcm, out)
	if err != nil {
		return nil, err
	}
	return out[:written], nil
}

// SendMCP executes the sendMCP method.
func (c *Client) SendMCP(ctx context.Context, payload any) error {
	if err := c.waitHelloReady(ctx); err != nil {
		return err
	}
	wrapper := map[string]any{
		"type":    "mcp",
		"payload": payload,
	}
	c.attachSessionID(wrapper)
	return c.sendJSON(ctx, wrapper)
}

func (c *Client) sendJSON(ctx context.Context, payload any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return errors.New("xiaozhi connection not ready")
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if err := conn.WriteJSON(payload); err != nil {
		return err
	}
	return nil
}

func (c *Client) waitHelloReady(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		c.mu.Lock()
		connReady := c.conn != nil
		helloReady := c.helloReady
		c.mu.Unlock()

		if !connReady {
			return errors.New("xiaozhi connection not ready")
		}
		if helloReady {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return errors.New("xiaozhi hello not acknowledged")
		case <-ticker.C:
		}
	}
}

func (c *Client) run(ctx context.Context) {
	delay := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		if c.isClosed() {
			return
		}
		c.logger.Info("xiaozhi connecting",
			zap.String("backend_url", c.cfg.BackendURL),
			zap.String("device_id", c.cfg.DeviceID),
			zap.String("client_id", c.cfg.ClientID),
		)
		if err := c.connectOnce(ctx); err != nil {
			c.reportError(err)
			c.logger.Warn("xiaozhi connect failed", zap.Error(err))
			time.Sleep(delay)
			delay = nextBackoff(delay)
			continue
		}
		c.logger.Info("xiaozhi connected",
			zap.String("backend_url", c.cfg.BackendURL),
			zap.String("device_id", c.cfg.DeviceID),
			zap.String("client_id", c.cfg.ClientID),
			zap.Int("protocol_version", c.getProtocolVersion()),
		)
		delay = time.Second
		if err := c.readLoop(); err != nil {
			if c.callbacks.OnDisconnected != nil {
				c.callbacks.OnDisconnected(err)
			}
			c.reportError(err)
			c.logger.Warn("xiaozhi connection lost", zap.Error(err))
			time.Sleep(delay)
			delay = nextBackoff(delay)
			continue
		}
	}
}

func (c *Client) connectOnce(ctx context.Context) error {
	if c.cfg.BackendURL == "" {
		return errors.New("xiaozhi backend url is empty")
	}

	version := c.getProtocolVersion()
	headers := http.Header{}
	headers.Set("Protocol-Version", intToString(version))
	headers.Set("Client-Id", c.cfg.ClientID)
	headers.Set("Device-Id", c.cfg.DeviceID)
	if c.cfg.AccessToken != "" {
		headers.Set("Authorization", "Bearer "+c.cfg.AccessToken)
	}

	dialer := websocket.Dialer{}
	conn, _, err := dialer.DialContext(ctx, c.cfg.BackendURL, headers)
	if err != nil {
		return err
	}
	conn.SetPingHandler(func(appData string) error {
		c.writeMu.Lock()
		defer c.writeMu.Unlock()
		return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(5*time.Second))
	})

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		_ = conn.Close()
		return errors.New("client closed")
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
	c.conn = conn
	c.sessionID = ""
	c.helloReady = false
	c.downstream = initialDownstreamAudio(c.cfg.AudioParams)
	if c.downstream.Format == "opus" {
		if err := c.ensureDecoderLocked(c.downstream.SampleRate, c.downstream.Channels); err != nil {
			c.logger.Warn("opus decoder init failed", zap.Error(err))
		}
	} else {
		c.decoder = nil
		c.decoderSR = 0
		c.decoderCH = 0
	}
	c.mu.Unlock()

	return c.sendHello(ctx)
}

func (c *Client) sendHello(ctx context.Context) error {
	audioParams := map[string]any{
		"format":         c.cfg.AudioParams.Format,
		"sample_rate":    c.cfg.AudioParams.SampleRate,
		"channels":       c.cfg.AudioParams.Channels,
		"frame_duration": c.cfg.AudioParams.FrameDuration,
	}
	if c.cfg.AudioParams.OutputFormat != "" {
		audioParams["output_format"] = c.cfg.AudioParams.OutputFormat
	}

	payload := map[string]any{
		"type":      "hello",
		"device_id": c.cfg.DeviceID,
		"version":   c.getProtocolVersion(),
		"features": map[string]any{
			"mcp": true,
			"aec": c.cfg.FeatureAEC,
		},
		"transport":    "websocket",
		"audio_params": audioParams,
	}
	return c.sendJSON(ctx, payload)
}

func (c *Client) listenMode() string {
	c.mu.Lock()
	mode := c.listen
	c.mu.Unlock()
	if mode == "" {
		return "auto"
	}
	return mode
}

// SetListenMode updates the listen mode sent to Xiaozhi.
func (c *Client) SetListenMode(mode string) {
	mode = normalizeListenMode(mode)
	c.mu.Lock()
	c.listen = mode
	c.mu.Unlock()
}

func (c *Client) readLoop() error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return errors.New("xiaozhi connection not ready")
	}

	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			c.mu.Lock()
			if c.conn == conn {
				_ = c.conn.Close()
				c.conn = nil
			}
			c.mu.Unlock()
			return err
		}

		switch msgType {
		case websocket.TextMessage:
			c.handleTextMessage(data)
		case websocket.BinaryMessage:
			payload, kind, decodeErr := c.decodeBinaryPayload(data)
			if decodeErr != nil {
				c.reportError(decodeErr)
				continue
			}
			if len(payload) == 0 {
				continue
			}
			if kind == xzcodec.PayloadKindCommand {
				c.handleTextMessage(payload)
				continue
			}
			c.handleBinaryFrame(payload)
		}
	}
}

func (c *Client) decodeBinaryPayload(frame []byte) ([]byte, xzcodec.PayloadKind, error) {
	return xzcodec.Decode(c.getProtocolVersion(), frame)
}

func (c *Client) handleTextMessage(data []byte) {
	var envelope struct {
		Type      string `json:"type"`
		SessionID string `json:"session_id,omitempty"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		c.reportError(err)
		return
	}
	if envelope.SessionID != "" {
		c.setSessionID(envelope.SessionID)
	}

	switch envelope.Type {
	case "hello":
		c.handleHelloMessage(data)
		return
	}

	var payload struct {
		Type      string          `json:"type"`
		Text      string          `json:"text"`
		State     string          `json:"state"`
		RawMCP    json.RawMessage `json:"payload"`
		SessionID string          `json:"session_id,omitempty"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		c.reportError(err)
		return
	}
	if payload.SessionID != "" {
		c.setSessionID(payload.SessionID)
	}

	switch payload.Type {
	case "stt":
		if payload.Text != "" && c.callbacks.OnSTT != nil {
			c.callbacks.OnSTT(payload.Text)
		}
	case "llm":
		if payload.Text != "" && c.callbacks.OnLLM != nil {
			c.callbacks.OnLLM(payload.Text, payload.State)
		}
	case "text":
		if payload.Text != "" && c.callbacks.OnText != nil {
			c.callbacks.OnText(payload.Text)
		}
	case "tts":
		if c.callbacks.OnTTS != nil {
			c.callbacks.OnTTS(payload.State, payload.Text)
		}
	case "mcp":
		if c.callbacks.OnMCP != nil {
			c.callbacks.OnMCP(payload.RawMCP)
		}
	case "goodbye":
		if c.callbacks.OnGoodbye != nil {
			c.callbacks.OnGoodbye()
		}
	}
}

func (c *Client) handleHelloMessage(data []byte) {
	var payload struct {
		Type      string `json:"type"`
		SessionID string `json:"session_id,omitempty"`
		Version   int    `json:"version,omitempty"`
		Audio     struct {
			Format       string `json:"format,omitempty"`
			OutputFormat string `json:"output_format,omitempty"`
			SampleRate   int    `json:"sample_rate,omitempty"`
			Channels     int    `json:"channels,omitempty"`
			FrameMs      int    `json:"frame_duration,omitempty"`
		} `json:"audio_params,omitempty"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		c.reportError(err)
		return
	}

	if payload.SessionID != "" {
		c.setSessionID(payload.SessionID)
	}
	if payload.Version > 0 {
		c.setProtocolVersion(payload.Version)
	}

	if payload.Audio.Format != "" || payload.Audio.OutputFormat != "" || payload.Audio.SampleRate > 0 || payload.Audio.Channels > 0 || payload.Audio.FrameMs > 0 {
		c.updateDownstreamAudio(payload.Audio.Format, payload.Audio.OutputFormat, payload.Audio.SampleRate, payload.Audio.Channels, payload.Audio.FrameMs)
	}

	format, sampleRate, channels, frameMs := c.downstreamSnapshot()
	c.logger.Info("xiaozhi hello acknowledged",
		zap.String("session_id", c.getSessionID()),
		zap.Int("protocol_version", c.getProtocolVersion()),
		zap.String("downstream_format", format),
		zap.Int("downstream_sample_rate", sampleRate),
		zap.Int("downstream_channels", channels),
		zap.Int("downstream_frame_duration", frameMs),
	)

	if c.markHelloReady() && c.callbacks.OnConnected != nil {
		c.callbacks.OnConnected()
	}
}

func (c *Client) handleBinaryFrame(frame []byte) {
	if len(frame) == 0 || c.callbacks.OnAudio == nil {
		return
	}

	format, sampleRate, channels, _ := c.downstreamSnapshot()
	switch format {
	case "opus":
		pcm, err := c.decodeOpus(frame, sampleRate, channels)
		if err != nil {
			c.reportError(err)
			return
		}
		if len(pcm) == 0 {
			return
		}
		c.callbacks.OnAudio(AudioFrame{PCM: pcm, SampleRate: sampleRate, Channels: channels})
	case "pcm_s16le", "pcm16", "pcm":
		c.callbacks.OnAudio(AudioFrame{PCM: frame, SampleRate: sampleRate, Channels: channels})
	case "wav":
		pcm, sr, ch, err := decodeWAVFrame(frame, sampleRate, channels)
		if err != nil {
			c.reportError(err)
			return
		}
		c.callbacks.OnAudio(AudioFrame{PCM: pcm, SampleRate: sr, Channels: ch})
	default:
		c.reportError(errors.New("unsupported xiaozhi audio format: " + format))
	}
}

func (c *Client) decodeOpus(frame []byte, sampleRate int, channels int) ([]byte, error) {
	if sampleRate <= 0 {
		sampleRate = 16000
	}
	if channels <= 0 {
		channels = 1
	}

	c.mu.Lock()
	if err := c.ensureDecoderLocked(sampleRate, channels); err != nil {
		c.mu.Unlock()
		return nil, err
	}
	decoder := c.decoder
	c.mu.Unlock()
	if decoder == nil {
		return nil, errors.New("opus decoder is not initialized")
	}

	maxSamples := sampleRate * opusMaxFrameDurationMs / 1000
	if maxSamples <= 0 {
		maxSamples = 5760
	}
	frameSamples := maxSamples * channels
	pcm := make([]int16, frameSamples)
	samplesDecoded, err := decoder.Decode(frame, pcm)
	if err != nil {
		return nil, err
	}
	if samplesDecoded <= 0 {
		return nil, nil
	}
	pcm = pcm[:samplesDecoded*channels]
	return pcm16ToBytes(pcm), nil
}

func decodeWAVFrame(frame []byte, fallbackSampleRate int, fallbackChannels int) ([]byte, int, int, error) {
	if len(frame) < 12 || string(frame[0:4]) != "RIFF" || string(frame[8:12]) != "WAVE" {
		return nil, 0, 0, errors.New("invalid wav frame")
	}

	sampleRate := fallbackSampleRate
	if sampleRate <= 0 {
		sampleRate = 16000
	}
	channels := fallbackChannels
	if channels <= 0 {
		channels = 1
	}
	bitsPerSample := 16

	offset := 12
	dataOffset := -1
	dataSize := 0
	for offset+8 <= len(frame) {
		chunkID := string(frame[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(frame[offset+4 : offset+8]))
		offset += 8
		if chunkSize < 0 {
			return nil, 0, 0, errors.New("invalid wav chunk size")
		}
		if offset+chunkSize > len(frame) {
			chunkSize = len(frame) - offset
		}

		switch chunkID {
		case "fmt ":
			if chunkSize >= 16 {
				channels = int(binary.LittleEndian.Uint16(frame[offset+2 : offset+4]))
				sampleRate = int(binary.LittleEndian.Uint32(frame[offset+4 : offset+8]))
				bitsPerSample = int(binary.LittleEndian.Uint16(frame[offset+14 : offset+16]))
			}
		case "data":
			dataOffset = offset
			dataSize = chunkSize
		}

		offset += chunkSize
		if chunkSize%2 == 1 {
			offset++
		}
	}

	if dataOffset < 0 || dataSize <= 0 || dataOffset+dataSize > len(frame) {
		return nil, 0, 0, errors.New("wav data chunk not found")
	}
	if bitsPerSample != 16 {
		return nil, 0, 0, errors.New("unsupported wav bits per sample")
	}
	return frame[dataOffset : dataOffset+dataSize], sampleRate, channels, nil
}

func pcm16ToBytes(pcm []int16) []byte {
	if len(pcm) == 0 {
		return nil
	}
	out := make([]byte, len(pcm)*2)
	for i, sample := range pcm {
		binary.LittleEndian.PutUint16(out[i*2:], uint16(sample))
	}
	return out
}

func (c *Client) attachSessionID(payload map[string]any) {
	if payload == nil {
		return
	}
	sessionID := c.getSessionID()
	if sessionID == "" {
		return
	}
	payload["session_id"] = sessionID
}

func (c *Client) getSessionID() string {
	c.mu.Lock()
	sessionID := c.sessionID
	c.mu.Unlock()
	return sessionID
}

func (c *Client) setSessionID(sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	c.mu.Lock()
	c.sessionID = sessionID
	c.mu.Unlock()
}

func (c *Client) getProtocolVersion() int {
	c.mu.Lock()
	version := c.protocolVersion
	c.mu.Unlock()
	return version
}

func (c *Client) setProtocolVersion(version int) {
	normalized := normalizeProtocolVersion(version)
	c.mu.Lock()
	changed := c.protocolVersion != normalized
	c.protocolVersion = normalized
	c.mu.Unlock()
	if changed {
		c.logger.Info("xiaozhi negotiated protocol version updated", zap.Int("protocol_version", normalized))
	}
}

func (c *Client) markHelloReady() bool {
	c.mu.Lock()
	if c.helloReady {
		c.mu.Unlock()
		return false
	}
	c.helloReady = true
	c.mu.Unlock()
	return true
}

func (c *Client) updateDownstreamAudio(format string, outputFormat string, sampleRate int, channels int, frameDuration int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	resolvedFormat := normalizeAudioFormat(outputFormat)
	if resolvedFormat == "" {
		resolvedFormat = normalizeAudioFormat(format)
	}
	if resolvedFormat == "" {
		resolvedFormat = c.downstream.Format
	}
	if resolvedFormat == "" {
		resolvedFormat = "opus"
	}

	if sampleRate <= 0 {
		sampleRate = c.downstream.SampleRate
	}
	if sampleRate <= 0 {
		sampleRate = 16000
	}
	if channels <= 0 {
		channels = c.downstream.Channels
	}
	if channels <= 0 {
		channels = 1
	}
	if frameDuration <= 0 {
		frameDuration = c.downstream.FrameDuration
	}
	if frameDuration <= 0 {
		frameDuration = 20
	}

	c.downstream.Format = resolvedFormat
	c.downstream.SampleRate = sampleRate
	c.downstream.Channels = channels
	c.downstream.FrameDuration = frameDuration

	if resolvedFormat == "opus" {
		if err := c.ensureDecoderLocked(sampleRate, channels); err != nil {
			c.logger.Warn("opus decoder re-init failed", zap.Error(err))
		}
	} else {
		c.decoder = nil
		c.decoderSR = 0
		c.decoderCH = 0
	}
}

func (c *Client) downstreamSnapshot() (format string, sampleRate int, channels int, frameDuration int) {
	c.mu.Lock()
	format = c.downstream.Format
	sampleRate = c.downstream.SampleRate
	channels = c.downstream.Channels
	frameDuration = c.downstream.FrameDuration
	c.mu.Unlock()
	if format == "" {
		format = "opus"
	}
	if sampleRate <= 0 {
		sampleRate = 16000
	}
	if channels <= 0 {
		channels = 1
	}
	if frameDuration <= 0 {
		frameDuration = 20
	}
	return format, sampleRate, channels, frameDuration
}

func (c *Client) ensureDecoderLocked(sampleRate int, channels int) error {
	if sampleRate <= 0 {
		sampleRate = 16000
	}
	if channels <= 0 {
		channels = 1
	}
	if c.decoder != nil && c.decoderSR == sampleRate && c.decoderCH == channels {
		return nil
	}
	decoder, err := godepsopus.NewDecoder(sampleRate, channels)
	if err != nil {
		return err
	}
	c.decoder = decoder
	c.decoderSR = sampleRate
	c.decoderCH = channels
	return nil
}

func normalizeAudioParams(params AudioParams) AudioParams {
	params.Format = normalizeAudioFormat(params.Format)
	if params.Format == "" {
		params.Format = "opus"
	}
	params.OutputFormat = normalizeAudioFormat(params.OutputFormat)
	if params.SampleRate <= 0 {
		params.SampleRate = 16000
	}
	if params.Channels <= 0 {
		params.Channels = 1
	}
	if params.FrameDuration <= 0 {
		params.FrameDuration = 20
	}
	return params
}

func initialDownstreamAudio(params AudioParams) AudioParams {
	downstream := normalizeAudioParams(params)
	if downstream.OutputFormat != "" {
		downstream.Format = downstream.OutputFormat
	}
	downstream.OutputFormat = ""
	return downstream
}

func normalizeListenMode(mode string) string {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case "manual", "realtime", "auto":
		return strings.TrimSpace(strings.ToLower(mode))
	default:
		return "auto"
	}
}

func normalizeAudioFormat(format string) string {
	switch strings.TrimSpace(strings.ToLower(format)) {
	case "":
		return ""
	case "opus":
		return "opus"
	case "pcm", "pcm16", "pcm_s16le":
		return "pcm_s16le"
	case "wav":
		return "wav"
	default:
		return strings.TrimSpace(strings.ToLower(format))
	}
}

func normalizeProtocolVersion(version int) int {
	return xzcodec.NormalizeVersion(version)
}

func (c *Client) reportError(err error) {
	if c.callbacks.OnError != nil {
		c.callbacks.OnError(err)
	}
}

func (c *Client) isClosed() bool {
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()
	return closed
}

func nextBackoff(delay time.Duration) time.Duration {
	if delay >= 30*time.Second {
		return 30 * time.Second
	}
	return delay * 2
}

func intToString(value int) string {
	if value == 0 {
		return "0"
	}
	buf := [20]byte{}
	i := len(buf)
	for value > 0 {
		i--
		buf[i] = byte('0' + value%10)
		value /= 10
	}
	return string(buf[i:])
}
