package ws

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	appconfig "github.com/saker-ai/vtuber-server/internal/config"
	"github.com/saker-ai/vtuber-server/internal/group"
	"github.com/saker-ai/vtuber-server/internal/storage"
	"github.com/saker-ai/vtuber-server/internal/xiaozhi"
	"github.com/saker-ai/vtuber-server/pkg/audio"
)

// Handler represents a handler.
type Handler struct {
	logger   *zap.Logger
	upgrader websocket.Upgrader
	config   appconfig.Config
	group    *group.Manager
	sessions map[string]*session
	mu       sync.Mutex
}

type incomingMessage struct {
	Type       string    `json:"type"`
	Text       string    `json:"text,omitempty"`
	File       string    `json:"file,omitempty"`
	Audio      []float64 `json:"audio,omitempty"`
	AudioPCM   string    `json:"audio_pcm,omitempty"`
	AudioRate  int       `json:"audio_sample_rate,omitempty"`
	AudioCh    int       `json:"audio_channels,omitempty"`
	ListenMode string    `json:"listen_mode,omitempty"`
	RequestID  string    `json:"request_id,omitempty"`
	Success    *bool     `json:"success,omitempty"`
	Image      string    `json:"image,omitempty"`
	MimeType   string    `json:"mime_type,omitempty"`
	Message    string    `json:"message,omitempty"`
	InviteeUID string    `json:"invitee_uid,omitempty"`
	TargetUID  string    `json:"target_uid,omitempty"`
	HistoryUID string    `json:"history_uid,omitempty"`
}

type session struct {
	conn             *websocket.Conn
	sendMu           sync.Mutex
	logger           *zap.Logger
	xiaozhi          *xiaozhi.Client
	handler          *Handler
	clientUID        string
	confName         string
	confUID          string
	live2dModelName  string
	characterName    string
	avatar           string
	historyUID       string
	llmText          string
	inConversation   bool
	ttsActive        bool
	displaySent      bool
	frameDuration    int
	listening        bool
	audioFormat      string
	unsupportedAudio bool
	sampleRate       int
	channels         int
	inputSampleRate  int
	inputChannels    int
	micPCMBuffer     []int16
	frameSamples     int
	ttsBuffer        []byte
	ttsSampleRate    int
	ttsChannels      int
	resampler        *audio.StreamResampler
	opusEncoder      *audio.OpusEncoder
	opusScratch      []int16
	pcmBytesScratch  []byte
	listenMode       string

	mcpMu          sync.Mutex
	mcpWaiters     map[string]chan captureResponse
	mcpVisionURL   string
	mcpVisionToken string
	deviceID       string
	clientID       string

	micChunkCount int
	micBytes      int
	lastMicLog    time.Time
	lastMicRate   int
	lastMicCh     int

	ttsChunkCount int
	ttsBytes      int
	lastTTSLog    time.Time
}

const ttsChunkDurationMs = 300

type captureResponse struct {
	Success  bool
	Image    string
	MimeType string
	Message  string
}

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	ID      json.RawMessage `json:"id"`
	Params  json.RawMessage `json:"params"`
}

type mcpTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type mcpResult struct {
	Content []mcpContent `json:"content"`
	IsError bool         `json:"isError"`
}

type mcpContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// NewHandler executes the newHandler function.
func NewHandler(logger *zap.Logger, cfg appconfig.Config) *Handler {
	return &Handler{
		logger:   logger,
		config:   cfg,
		group:    group.NewManager(),
		sessions: make(map[string]*session),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

// Handle executes the handle method.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Warn("ws upgrade failed", zap.Error(err))
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sessionID := fmt.Sprintf("%d", time.Now().UnixNano())
	xzCfg := xiaozhi.Config{
		BackendURL:      h.config.XiaoZhiBackendURL,
		ProtocolVersion: h.config.XiaoZhiProtocolVersion,
		AudioParams: xiaozhi.AudioParams{
			Format:        h.config.XiaoZhiAudioFormat,
			SampleRate:    h.config.XiaoZhiSampleRate,
			Channels:      h.config.XiaoZhiChannels,
			FrameDuration: h.config.XiaoZhiFrameDuration,
		},
		ListenMode:  h.config.XiaoZhiListenMode,
		DeviceID:    fallbackID(h.config.XiaoZhiDeviceID, "mio-device-"+sessionID),
		ClientID:    fallbackID(h.config.XiaoZhiClientID, "mio-client-"+sessionID),
		AccessToken: h.config.XiaoZhiAccessToken,
	}

	sess := &session{
		conn:            conn,
		logger:          h.logger,
		handler:         h,
		clientUID:       sessionID,
		confName:        h.config.CharacterConfig.ConfName,
		confUID:         h.config.CharacterConfig.ConfUID,
		live2dModelName: h.config.CharacterConfig.Live2dModelName,
		characterName:   h.config.CharacterConfig.CharacterName,
		avatar:          h.config.CharacterConfig.Avatar,
		frameDuration:   h.config.XiaoZhiFrameDuration,
		audioFormat:     h.config.XiaoZhiAudioFormat,
		sampleRate:      h.config.XiaoZhiSampleRate,
		channels:        h.config.XiaoZhiChannels,
		inputSampleRate: h.config.XiaoZhiSampleRate,
		inputChannels:   h.config.XiaoZhiChannels,
		frameSamples:    h.config.XiaoZhiSampleRate * h.config.XiaoZhiFrameDuration / 1000,
		listenMode:      h.config.XiaoZhiListenMode,
		mcpWaiters:      make(map[string]chan captureResponse),
		deviceID:        xzCfg.DeviceID,
		clientID:        xzCfg.ClientID,
	}
	if sess.audioFormat == "opus" {
		if enc, err := audio.AcquireOpusEncoder(sess.sampleRate, sess.channels, sess.frameDuration); err != nil {
			sess.logger.Warn("opus encoder init failed", zap.Error(err))
		} else {
			sess.opusEncoder = enc
		}
	}

	sess.logger.Info("ws session opened",
		zap.String("session_id", sess.clientUID),
		zap.String("device_id", sess.deviceID),
		zap.String("client_id", sess.clientID),
		zap.String("audio_format", sess.audioFormat),
		zap.Int("sample_rate", sess.sampleRate),
		zap.Int("channels", sess.channels),
		zap.Int("frame_duration", sess.frameDuration),
	)

	h.registerSession(sess)
	sess.sendModelAndConf()

	callbacks := xiaozhi.Callbacks{
		OnSTT: func(text string) {
			sess.logger.Debug("xiaozhi stt",
				zap.String("session_id", sess.clientUID),
				zap.Int("chars", len(text)),
			)
			sess.sendJSON(map[string]any{"type": "user-input-transcription", "text": text})
		},
		OnLLM: func(text string, state string) {
			sess.logger.Debug("xiaozhi llm",
				zap.String("session_id", sess.clientUID),
				zap.String("state", state),
				zap.Int("chars", len(text)),
			)
			sess.ensureConversation()
			sess.applyLLMText(text, state)
		},
		OnText: func(text string) {
			sess.logger.Debug("xiaozhi text",
				zap.String("session_id", sess.clientUID),
				zap.Int("chars", len(text)),
			)
			sess.ensureConversation()
			sess.llmText = text
			sess.sendJSON(map[string]any{"type": "full-text", "text": sess.llmText})
		},
		OnTTS: func(state string, text string) {
			sess.logger.Debug("xiaozhi tts",
				zap.String("session_id", sess.clientUID),
				zap.String("state", state),
				zap.Int("chars", len(text)),
			)
			sess.handleTTS(ctx, state, text)
		},
		OnMCP: func(payload json.RawMessage) {
			sess.logger.Debug("xiaozhi mcp",
				zap.String("session_id", sess.clientUID),
				zap.Int("bytes", len(payload)),
			)
			sess.handleMCP(ctx, payload)
		},
		OnGoodbye: func() {
			sess.sendJSON(map[string]any{"type": "error", "message": "xiaozhi backend disconnected"})
			sess.endConversation()
		},
		OnAudio: func(frame xiaozhi.AudioFrame) {
			sess.handleAudio(frame)
		},
		OnConnected: func() {
			// Re-establish local state after reconnect.
			sess.listening = false
			if sess.listenMode == "manual" {
				sess.logger.Info("xiaozhi reconnected, manual mode waits for mic trigger",
					zap.String("session_id", sess.clientUID),
				)
				return
			}
			if err := sess.xiaozhi.SendListenState(ctx, "start"); err != nil {
				sess.logger.Warn("xiaozhi listen start on reconnect failed",
					zap.String("session_id", sess.clientUID),
					zap.String("mode", sess.listenMode),
					zap.Error(err),
				)
				return
			}
			sess.listening = true
			sess.logger.Info("xiaozhi reconnected and listen primed",
				zap.String("session_id", sess.clientUID),
				zap.String("mode", sess.listenMode),
			)
		},
		OnDisconnected: func(err error) {
			sess.listening = false
			sess.logger.Warn("xiaozhi disconnected, reset local listen state",
				zap.String("session_id", sess.clientUID),
				zap.Error(err),
			)
		},
		OnError: func(err error) {
			sess.logger.Warn("xiaozhi error", zap.Error(err))
		},
	}

	sess.xiaozhi = xiaozhi.NewClient(xzCfg, callbacks, h.logger)
	sess.xiaozhi.Connect(ctx)

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			sess.logger.Debug("ws connection closed", zap.Error(err))
			break
		}
		var msg incomingMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			sess.sendJSON(map[string]any{"type": "error", "message": "invalid json"})
			continue
		}
		if msg.Type != "heartbeat" {
			sess.logger.Debug("ws incoming message",
				zap.String("session_id", sess.clientUID),
				zap.String("type", msg.Type),
			)
		}
		sess.handleIncoming(ctx, msg)
	}

	sess.xiaozhi.Close()
	if sess.resampler != nil {
		sess.resampler.Close()
		sess.resampler = nil
	}
	if sess.opusEncoder != nil {
		audio.ReleaseOpusEncoder(sess.opusEncoder)
		sess.opusEncoder = nil
	}
	sess.logger.Info("ws session closed", zap.String("session_id", sess.clientUID))
	h.unregisterSession(sess.clientUID)
}

func (s *session) handleIncoming(ctx context.Context, msg incomingMessage) {
	switch msg.Type {
	case "text-input":
		text := msg.Text
		if text == "" {
			return
		}
		if err := s.xiaozhi.SendTextInput(ctx, text); err != nil {
			s.sendJSON(map[string]any{"type": "error", "message": err.Error()})
		}
	case "interrupt-signal":
		if err := s.xiaozhi.Abort(ctx); err != nil {
			s.sendJSON(map[string]any{"type": "error", "message": err.Error()})
		}
		s.endConversation()
	case "mic-audio-data":
		if msg.AudioPCM != "" {
			s.handleMicAudioPCM(ctx, msg.AudioPCM, msg.AudioRate, msg.AudioCh)
		} else {
			s.handleMicAudio(ctx, msg.Audio)
		}
	case "mic-audio-end":
		s.handleMicEnd(ctx)
	case "set-listen-mode":
		s.handleSetListenMode(msg.ListenMode)
	case "mcp-capture-response":
		s.handleCaptureResponse(msg)
	case "frontend-playback-complete":
		s.sendJSON(map[string]any{"type": "force-new-message"})
	case "fetch-configs":
		s.handleFetchConfigs(ctx)
	case "switch-config":
		s.handleConfigSwitch(ctx, msg.File)
	case "fetch-backgrounds":
		s.handleFetchBackgrounds(ctx)
	case "request-init-config":
		s.handleInitConfig(ctx)
	case "fetch-history-list":
		s.handleHistoryList(ctx)
	case "fetch-and-set-history":
		s.handleFetchHistory(ctx, msg.HistoryUID)
	case "create-new-history":
		s.handleCreateHistory(ctx)
	case "delete-history":
		s.handleDeleteHistory(ctx, msg.HistoryUID)
	case "request-group-info":
		s.handleGroupInfo(ctx)
	case "add-client-to-group":
		s.handleAddToGroup(ctx, msg.InviteeUID)
	case "remove-client-from-group":
		s.handleRemoveFromGroup(ctx, msg.TargetUID)
	case "ai-speak-signal":
		s.sendJSON(map[string]any{"type": "error", "message": "proactive speak not supported in XiaoZhi mode"})
	case "heartbeat":
		return
	default:
		s.logger.Debug("ws unknown message type",
			zap.String("session_id", s.clientUID),
			zap.String("type", msg.Type),
		)
		return
	}
}

func (s *session) handleMicAudio(ctx context.Context, samples []float64) {
	if len(samples) == 0 {
		return
	}
	tmp := float64ToFloat32(samples)
	pcm := audio.Float32SliceToInt16SliceInto(audio.AcquireInt16(len(tmp)), tmp)
	pcmBytes := audio.Int16SliceToBytesInto(audio.AcquireBytes(len(pcm)*2), pcm)
	s.handleMicPCMBytes(ctx, pcmBytes, s.sampleRate, s.channels)
	audio.ReleaseBytes(pcmBytes)
	audio.ReleaseInt16(pcm)
	audio.ReleaseFloat32(tmp)
}

func (s *session) handleMicAudioPCM(ctx context.Context, audioPCM string, sampleRate int, channels int) {
	if audioPCM == "" {
		return
	}
	pcm, err := base64.StdEncoding.DecodeString(audioPCM)
	if err != nil {
		s.logger.Warn("mic audio pcm decode failed", zap.Error(err))
		s.sendJSON(map[string]any{"type": "error", "message": "invalid mic audio pcm"})
		return
	}
	if len(pcm) == 0 {
		return
	}
	s.lastMicRate = sampleRate
	s.lastMicCh = channels
	s.handleMicPCMBytes(ctx, pcm, sampleRate, channels)
}

func (s *session) handleMicEnd(ctx context.Context) {
	s.flushMicBuffer(ctx, true)
	shouldStop := s.listenMode == "manual"
	if shouldStop && s.listening {
		if err := s.xiaozhi.SendListenState(ctx, "stop"); err == nil {
			s.logger.Info("xiaozhi listen stop", zap.String("session_id", s.clientUID))
		} else {
			s.logger.Warn("xiaozhi listen stop failed", zap.Error(err))
		}
		s.listening = false
	}
	s.logger.Debug("mic end state",
		zap.String("session_id", s.clientUID),
		zap.String("listen_mode", s.listenMode),
		zap.Bool("stopped", shouldStop),
		zap.Bool("listening", s.listening),
		zap.Int("llm_chars", len(s.llmText)),
	)
	s.logger.Info("mic audio end",
		zap.String("session_id", s.clientUID),
		zap.Int("chunks", s.micChunkCount),
		zap.Int("bytes", s.micBytes),
		zap.Int("input_rate", s.lastMicRate),
		zap.Int("input_channels", s.lastMicCh),
		zap.Int("target_rate", s.sampleRate),
		zap.Int("target_channels", s.channels),
	)
	s.micChunkCount = 0
	s.micBytes = 0
	s.ensureConversation()
	if s.llmText == "" {
		s.sendJSON(map[string]any{"type": "full-text", "text": "Thinking..."})
	}
}

func (s *session) handleSetListenMode(mode string) {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		return
	}
	switch mode {
	case "realtime", "auto", "manual":
		if s.listenMode != mode {
			s.logger.Info("listen mode updated",
				zap.String("session_id", s.clientUID),
				zap.String("mode", mode),
			)
		}
		s.listenMode = mode
		s.xiaozhi.SetListenMode(mode)
	default:
		s.logger.Warn("invalid listen mode",
			zap.String("session_id", s.clientUID),
			zap.String("mode", mode),
		)
	}
}

func (s *session) handleMicPCMBytes(ctx context.Context, pcm []byte, sampleRate int, channels int) {
	if len(pcm) == 0 {
		return
	}
	if !s.listening {
		if err := s.xiaozhi.SendListenState(ctx, "start"); err != nil {
			s.logger.Warn("xiaozhi listen start failed", zap.Error(err))
			return
		}
		s.logger.Info("xiaozhi listen start", zap.String("session_id", s.clientUID))
		s.listening = true
	}

	inputRate := sampleRate
	if inputRate <= 0 {
		inputRate = s.inputSampleRate
	}
	inputCh := channels
	if inputCh <= 0 {
		inputCh = s.inputChannels
	}
	if inputRate <= 0 {
		inputRate = s.sampleRate
	}
	if inputCh <= 0 {
		inputCh = s.channels
	}
	if inputRate != s.inputSampleRate || inputCh != s.inputChannels {
		s.inputSampleRate = inputRate
		s.inputChannels = inputCh
		if s.resampler != nil {
			s.resampler.Close()
			s.resampler = nil
		}
	}

	s.micChunkCount++
	s.micBytes += len(pcm)
	now := time.Now()
	if s.lastMicLog.IsZero() || now.Sub(s.lastMicLog) >= 2*time.Second {
		s.lastMicLog = now
		s.logger.Info("mic audio stats",
			zap.String("session_id", s.clientUID),
			zap.Int("chunks", s.micChunkCount),
			zap.Int("bytes", s.micBytes),
			zap.Int("input_rate", inputRate),
			zap.Int("input_channels", inputCh),
			zap.Int("target_rate", s.sampleRate),
			zap.Int("target_channels", s.channels),
			zap.String("format", s.audioFormat),
			zap.Bool("resampling", s.resampler != nil),
			zap.Bool("listening", s.listening),
		)
		s.micChunkCount = 0
		s.micBytes = 0
	}

	if s.audioFormat != "opus" && s.audioFormat != "pcm16" && s.audioFormat != "pcm" {
		if !s.unsupportedAudio {
			s.unsupportedAudio = true
			s.logger.Warn("unsupported mic audio format",
				zap.String("session_id", s.clientUID),
				zap.String("format", s.audioFormat),
			)
			s.sendJSON(map[string]any{"type": "error", "message": "unsupported xiaozhi_audio_format for mic input"})
		}
		return
	}

	if inputRate != s.sampleRate {
		if s.resampler == nil {
			res, err := audio.NewStreamResampler(inputRate, s.sampleRate)
			if err != nil {
				s.logger.Warn("resampler init failed", zap.Error(err))
			} else {
				s.resampler = res
			}
		}
	}

	if s.resampler != nil {
		scratch := audio.BytesToInt16SliceInto(s.opusScratch, pcm)
		s.opusScratch = scratch
		if err := s.resampler.AppendPCM(scratch); err != nil {
			s.logger.Warn("resampler append failed", zap.Error(err))
			return
		}
		s.processResampledFrames(ctx, false)
		return
	}

	s.appendPCMBuffer(pcm)
	s.processPCMFrames(ctx, false)
}

func (s *session) processResampledFrames(ctx context.Context, flush bool) {
	frameSize := s.frameSamples * s.channels
	if frameSize <= 0 {
		frameSize = 960 * maxInt(1, s.channels)
	}
	if flush {
		if err := s.resampler.Flush(); err != nil {
			s.logger.Warn("resampler flush failed", zap.Error(err))
		}
	}
	framesSent := 0
	for {
		frame, ok := s.resampler.PopFrame(frameSize)
		if !ok {
			break
		}
		s.sendPCMFrame(ctx, frame)
		audio.ReleaseInt16(frame)
		framesSent++
	}
	if flush {
		if frame := s.resampler.PopRemainderPadded(frameSize); frame != nil {
			s.sendPCMFrame(ctx, frame)
			audio.ReleaseInt16(frame)
			framesSent++
		}
	}
	if framesSent > 0 {
		s.logger.Debug("mic audio frames sent",
			zap.String("session_id", s.clientUID),
			zap.Int("frames", framesSent),
		)
	}
}

func (s *session) appendPCMBuffer(pcm []byte) {
	samples := audio.BytesToInt16SliceInto(s.opusScratch, pcm)
	s.opusScratch = samples
	s.micPCMBuffer = append(s.micPCMBuffer, samples...)
}

func (s *session) processPCMFrames(ctx context.Context, flush bool) {
	frameSize := s.frameSamples * s.channels
	if frameSize <= 0 {
		frameSize = 960 * maxInt(1, s.channels)
	}
	framesSent := 0
	for len(s.micPCMBuffer) >= frameSize {
		frame := s.micPCMBuffer[:frameSize]
		s.micPCMBuffer = s.micPCMBuffer[frameSize:]
		s.sendPCMFrame(ctx, frame)
		framesSent++
	}
	if flush && len(s.micPCMBuffer) > 0 {
		frame := s.micPCMBuffer
		s.micPCMBuffer = nil
		s.sendPCMFrame(ctx, frame)
		framesSent++
	}
	if framesSent > 0 {
		s.logger.Debug("mic audio frames sent",
			zap.String("session_id", s.clientUID),
			zap.Int("frames", framesSent),
		)
	}
}

func (s *session) sendPCMFrame(ctx context.Context, frame []int16) {
	if len(frame) == 0 {
		return
	}
	if s.audioFormat == "opus" {
		if s.opusEncoder != nil {
			frameBytes := audio.Int16SliceToBytesInto(s.pcmBytesScratch, frame)
			s.pcmBytesScratch = frameBytes
			encoded, err := s.opusEncoder.EncodeWithScratch(frameBytes, s.opusScratch)
			if err != nil {
				s.logger.Warn("opus encode failed", zap.Error(err))
				s.sendJSON(map[string]any{"type": "error", "message": err.Error()})
				return
			}
			if len(encoded) == 0 {
				return
			}
			if err := s.xiaozhi.SendAudio(ctx, encoded); err != nil {
				s.logger.Warn("xiaozhi send opus audio failed", zap.Error(err))
				s.sendJSON(map[string]any{"type": "error", "message": err.Error()})
			}
			return
		}
		tmp := audio.Int16SliceToFloat32Into(audio.AcquireFloat32(len(frame)), frame)
		encoded, err := s.xiaozhi.EncodeOpusFloat(tmp)
		audio.ReleaseFloat32(tmp)
		if err != nil {
			s.logger.Warn("opus encode failed", zap.Error(err))
			s.sendJSON(map[string]any{"type": "error", "message": err.Error()})
			return
		}
		if len(encoded) == 0 {
			return
		}
		if err := s.xiaozhi.SendAudio(ctx, encoded); err != nil {
			s.logger.Warn("xiaozhi send opus audio failed", zap.Error(err))
			s.sendJSON(map[string]any{"type": "error", "message": err.Error()})
		}
		return
	}

	pcmBytes := audio.Int16SliceToBytesInto(s.pcmBytesScratch, frame)
	s.pcmBytesScratch = pcmBytes
	if err := s.xiaozhi.SendAudio(ctx, pcmBytes); err != nil {
		s.logger.Warn("xiaozhi send audio failed", zap.Error(err))
		s.sendJSON(map[string]any{"type": "error", "message": err.Error()})
	}
}

func (s *session) flushMicBuffer(ctx context.Context, flush bool) {
	if s.resampler != nil {
		s.processResampledFrames(ctx, flush)
		return
	}
	s.processPCMFrames(ctx, flush)
}

func (s *session) handleCaptureResponse(msg incomingMessage) {
	if msg.RequestID == "" || msg.Success == nil {
		return
	}
	resp := captureResponse{
		Success:  *msg.Success,
		Image:    msg.Image,
		MimeType: msg.MimeType,
		Message:  msg.Message,
	}
	s.mcpMu.Lock()
	ch, ok := s.mcpWaiters[msg.RequestID]
	if ok {
		delete(s.mcpWaiters, msg.RequestID)
	}
	s.mcpMu.Unlock()
	if ok {
		ch <- resp
	}
}

func (s *session) handleFetchConfigs(ctx context.Context) {
	files, err := appconfig.ScanConfigFiles(s.handler.config.RootDir, s.handler.config.ConfigAltsDir)
	if err != nil {
		s.sendJSON(map[string]any{"type": "error", "message": err.Error()})
		return
	}
	s.sendJSON(map[string]any{"type": "config-files", "configs": files})
}

func (s *session) handleConfigSwitch(ctx context.Context, filename string) {
	if filename == "" {
		return
	}
	configPath := filename
	if filename != "conf.yaml" {
		configPath = filepath.Join(s.handler.config.ConfigAltsDir, filepath.Base(filename))
	} else {
		configPath = filepath.Join(s.handler.config.RootDir, "conf.yaml")
	}
	conf, err := appconfig.ReadCharacterConfig(configPath)
	if err != nil {
		s.sendJSON(map[string]any{"type": "error", "message": err.Error()})
		return
	}
	s.confName = conf.ConfName
	s.confUID = conf.ConfUID
	s.live2dModelName = conf.Live2dModelName
	s.characterName = conf.CharacterName
	s.avatar = conf.Avatar
	s.historyUID = ""

	s.sendModelAndConf()
	s.sendJSON(map[string]any{"type": "config-switched"})
}

func (s *session) handleFetchBackgrounds(ctx context.Context) {
	files := appconfig.ScanBackgrounds(s.handler.config.BackgroundsDir)
	s.sendJSON(map[string]any{"type": "background-files", "files": files})
}

func (s *session) handleInitConfig(ctx context.Context) {
	s.sendModelAndConf()
}

func (s *session) handleHistoryList(ctx context.Context) {
	histories := storage.GetHistoryList(s.handler.config.ChatHistoryDir, s.confUID)
	s.sendJSON(map[string]any{"type": "history-list", "histories": histories})
}

func (s *session) handleFetchHistory(ctx context.Context, historyUID string) {
	if historyUID == "" {
		return
	}
	messages, err := storage.GetHistory(s.handler.config.ChatHistoryDir, s.confUID, historyUID)
	if err != nil {
		s.sendJSON(map[string]any{"type": "error", "message": err.Error()})
		return
	}
	s.historyUID = historyUID
	s.sendJSON(map[string]any{"type": "history-data", "messages": messages})
}

func (s *session) handleCreateHistory(ctx context.Context) {
	historyUID, err := storage.CreateHistory(s.handler.config.ChatHistoryDir, s.confUID)
	if err != nil {
		s.sendJSON(map[string]any{"type": "error", "message": err.Error()})
		return
	}
	s.historyUID = historyUID
	s.sendJSON(map[string]any{"type": "new-history-created", "history_uid": historyUID})
}

func (s *session) handleDeleteHistory(ctx context.Context, historyUID string) {
	if historyUID == "" {
		return
	}
	success := storage.DeleteHistory(s.handler.config.ChatHistoryDir, s.confUID, historyUID)
	s.sendJSON(map[string]any{"type": "history-deleted", "success": success, "history_uid": historyUID})
	if success && s.historyUID == historyUID {
		s.historyUID = ""
	}
}

func (s *session) handleGroupInfo(ctx context.Context) {
	s.handler.sendGroupUpdate(s.clientUID)
}

func (s *session) handleAddToGroup(ctx context.Context, inviteeUID string) {
	if inviteeUID == "" {
		return
	}
	success, message, members := s.handler.group.AddClient(s.clientUID, inviteeUID)
	s.sendJSON(map[string]any{"type": "group-operation-result", "success": success, "message": message})
	if success {
		s.handler.broadcastGroupUpdate(members)
	}
}

func (s *session) handleRemoveFromGroup(ctx context.Context, targetUID string) {
	if targetUID == "" {
		return
	}
	success, message, members := s.handler.group.RemoveClientFromGroup(s.clientUID, targetUID)
	s.sendJSON(map[string]any{"type": "group-operation-result", "success": success, "message": message})
	if success {
		s.handler.broadcastGroupUpdate(members)
	}
}

func (s *session) ensureConversation() {
	if s.inConversation {
		return
	}
	s.inConversation = true
	s.llmText = ""
	s.ttsBuffer = nil
	s.ttsSampleRate = 0
	s.ttsChannels = 0
	s.sendJSON(map[string]any{"type": "control", "text": "conversation-chain-start"})
}

func (s *session) endConversation() {
	if !s.inConversation {
		return
	}
	s.inConversation = false
	s.ttsActive = false
	s.displaySent = false
	s.llmText = ""
	s.ttsBuffer = nil
	s.ttsSampleRate = 0
	s.ttsChannels = 0
	s.sendJSON(map[string]any{"type": "control", "text": "conversation-chain-end"})
}

func (s *session) handleTTS(ctx context.Context, state string, text string) {
	switch state {
	case "sentence_start":
		if text == "" {
			return
		}
		s.ensureConversation()
		s.llmText += text
		s.sendJSON(map[string]any{"type": "full-text", "text": s.llmText})
	case "start":
		s.ensureConversation()
		s.ttsActive = true
		s.displaySent = false
		s.ttsBuffer = nil
		s.ttsSampleRate = 0
		s.ttsChannels = 0
		s.ttsChunkCount = 0
		s.ttsBytes = 0
		s.lastTTSLog = time.Now()
		s.logger.Info("tts start", zap.String("session_id", s.clientUID))
		if s.llmText == "" {
			s.sendJSON(map[string]any{"type": "full-text", "text": "Thinking..."})
		}
	case "stop":
		s.ttsActive = false
		s.flushTTSAudio(true)
		s.sendJSON(map[string]any{"type": "backend-synth-complete"})
		s.logger.Info("tts stop",
			zap.String("session_id", s.clientUID),
			zap.Int("chunks", s.ttsChunkCount),
			zap.Int("bytes", s.ttsBytes),
			zap.Int("sample_rate", s.ttsSampleRate),
			zap.Int("channels", s.ttsChannels),
		)
		// Match py-xiaozhi keep-listening behavior:
		// auto mode re-primes listening after each TTS turn instead of per mic segment.
		if s.listenMode == "auto" {
			if err := s.xiaozhi.SendListenState(ctx, "start"); err != nil {
				s.listening = false
				s.logger.Warn("xiaozhi listen start after tts stop failed",
					zap.String("session_id", s.clientUID),
					zap.Error(err),
				)
			} else {
				s.listening = true
				s.logger.Info("xiaozhi listen start after tts stop",
					zap.String("session_id", s.clientUID),
				)
			}
		}
		s.endConversation()
	}
}

func (s *session) applyLLMText(text string, state string) {
	if state == "stream" {
		s.llmText += text
	} else {
		s.llmText = text
	}
	s.sendJSON(map[string]any{"type": "full-text", "text": s.llmText})
}

func (s *session) handleAudio(frame xiaozhi.AudioFrame) {
	if !s.ttsActive {
		return
	}
	if len(frame.PCM) == 0 {
		return
	}
	if s.ttsSampleRate == 0 {
		s.ttsSampleRate = frame.SampleRate
		s.ttsChannels = frame.Channels
	} else if s.ttsSampleRate != frame.SampleRate || s.ttsChannels != frame.Channels {
		s.flushTTSAudio(true)
		s.ttsSampleRate = frame.SampleRate
		s.ttsChannels = frame.Channels
	}
	s.ttsBuffer = append(s.ttsBuffer, frame.PCM...)
	s.flushTTSAudio(false)
}

func (s *session) flushTTSAudio(final bool) {
	if len(s.ttsBuffer) == 0 {
		return
	}
	sampleRate := s.ttsSampleRate
	channels := s.ttsChannels
	if sampleRate <= 0 || channels <= 0 {
		return
	}
	chunkFrames := sampleRate * ttsChunkDurationMs / 1000
	if chunkFrames <= 0 {
		chunkFrames = (len(s.ttsBuffer) / 2) / channels
	}
	chunkBytes := chunkFrames * channels * 2
	if chunkBytes <= 0 {
		return
	}
	for len(s.ttsBuffer) >= chunkBytes {
		chunk := s.ttsBuffer[:chunkBytes]
		s.ttsBuffer = s.ttsBuffer[chunkBytes:]
		s.sendAudioChunk(chunk, sampleRate, channels)
	}
	if final && len(s.ttsBuffer) > 0 {
		chunk := s.ttsBuffer
		s.ttsBuffer = nil
		s.sendAudioChunk(chunk, sampleRate, channels)
	}
}

func (s *session) sendAudioChunk(pcm []byte, sampleRate int, channels int) {
	sliceLength := s.frameDuration
	if sampleRate > 0 && channels > 0 {
		frames := (len(pcm) / 2) / channels
		if frames > 0 {
			sliceLength = int(math.Round(float64(frames*1000) / float64(sampleRate)))
		}
	}
	if sliceLength <= 0 {
		sliceLength = s.frameDuration
	}
	volumes := computeVolumes(pcm, sampleRate, channels, sliceLength)
	payload := map[string]any{
		"type":              "audio",
		"audio_pcm":         base64.StdEncoding.EncodeToString(pcm),
		"audio_format":      "pcm16",
		"audio_sample_rate": sampleRate,
		"audio_channels":    channels,
		"volumes":           volumes,
		"slice_length":      sliceLength,
		"display_text":      s.buildDisplayText(),
		"actions":           nil,
		"forwarded":         false,
	}
	if s.displaySent {
		payload["display_text"] = nil
	}
	s.sendJSON(payload)
	s.displaySent = true
	s.ttsChunkCount++
	s.ttsBytes += len(pcm)
	now := time.Now()
	if s.lastTTSLog.IsZero() || now.Sub(s.lastTTSLog) >= 2*time.Second {
		s.lastTTSLog = now
		s.logger.Info("tts audio stats",
			zap.String("session_id", s.clientUID),
			zap.Int("chunks", s.ttsChunkCount),
			zap.Int("bytes", s.ttsBytes),
			zap.Int("sample_rate", sampleRate),
			zap.Int("channels", channels),
		)
		s.ttsChunkCount = 0
		s.ttsBytes = 0
	}
}

func (s *session) buildDisplayText() map[string]any {
	if s.llmText == "" {
		return nil
	}
	return map[string]any{
		"text":   s.llmText,
		"name":   "",
		"avatar": "",
	}
}

func (s *session) sendModelAndConf() {
	modelInfo, err := appconfig.LoadModelInfo(s.live2dModelName, s.handler.config.ModelDictPath)
	if err != nil {
		s.sendJSON(map[string]any{"type": "error", "message": err.Error()})
		return
	}
	payload := map[string]any{
		"type":       "set-model-and-conf",
		"model_info": modelInfo,
		"conf_name":  s.confName,
		"conf_uid":   s.confUID,
		"client_uid": s.clientUID,
	}
	s.sendJSON(payload)
}

func (s *session) handleMCP(ctx context.Context, payload json.RawMessage) {
	var req mcpRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		s.replyMCPError(ctx, req.ID, "invalid MCP payload")
		return
	}
	if req.JSONRPC != "2.0" {
		s.replyMCPError(ctx, req.ID, "invalid JSON-RPC version")
		return
	}
	if req.Method == "" || len(req.ID) == 0 {
		s.replyMCPError(ctx, req.ID, "missing method or id")
		return
	}

	switch req.Method {
	case "initialize":
		s.handleMCPInitialize(ctx, req)
	case "tools/list":
		s.handleMCPToolsList(ctx, req)
	case "tools/call":
		s.handleMCPToolsCall(ctx, req)
	default:
		s.replyMCPError(ctx, req.ID, "method not implemented")
	}
}

func (s *session) handleMCPInitialize(ctx context.Context, req mcpRequest) {
	var params struct {
		Capabilities struct {
			Vision struct {
				URL   string `json:"url"`
				Token string `json:"token"`
			} `json:"vision"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(req.Params, &params); err == nil {
		s.mcpVisionURL = params.Capabilities.Vision.URL
		s.mcpVisionToken = params.Capabilities.Vision.Token
	}
	result := map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{"tools": map[string]any{}},
		"serverInfo": map[string]any{
			"name":    "mio-gateway",
			"version": "1.0",
		},
	}
	s.replyMCPResult(ctx, req.ID, result)
}

func (s *session) handleMCPToolsList(ctx context.Context, req mcpRequest) {
	tools := []mcpTool{
		{
			Name:        "take_photo",
			Description: "Capture a camera frame and analyze it.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"question": map[string]any{"type": "string", "default": ""}},
				"required":   []string{},
			},
		},
		{
			Name:        "take_screenshot",
			Description: "Capture a screen frame and analyze it.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"question": map[string]any{"type": "string", "default": ""},
					"display":  map[string]any{"type": "string", "default": ""},
				},
				"required": []string{},
			},
		},
	}
	result := map[string]any{"tools": tools}
	s.replyMCPResult(ctx, req.ID, result)
}

func (s *session) handleMCPToolsCall(ctx context.Context, req mcpRequest) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.replyMCPError(ctx, req.ID, "invalid tool call params")
		return
	}
	if params.Name == "" {
		s.replyMCPError(ctx, req.ID, "missing tool name")
		return
	}

	toolID := mcpIDString(req.ID)
	s.sendToolStatus(toolID, params.Name, "running", "")

	question := getStringArg(params.Arguments, "question")
	switch params.Name {
	case "take_photo":
		result, status, content := s.captureAndAnalyze(ctx, "camera", question, "")
		s.sendToolStatus(toolID, params.Name, status, content)
		s.replyMCPResult(ctx, req.ID, result)
	case "take_screenshot":
		display := getStringArg(params.Arguments, "display")
		result, status, content := s.captureAndAnalyze(ctx, "screen", question, display)
		s.sendToolStatus(toolID, params.Name, status, content)
		s.replyMCPResult(ctx, req.ID, result)
	default:
		s.replyMCPError(ctx, req.ID, "unknown tool")
	}
}

func (s *session) captureAndAnalyze(ctx context.Context, source string, question string, display string) (any, string, string) {
	id := newRequestID()
	ch := make(chan captureResponse, 1)
	s.mcpMu.Lock()
	s.mcpWaiters[id] = ch
	s.mcpMu.Unlock()

	s.sendJSON(map[string]any{
		"type":       "mcp-capture-request",
		"request_id": id,
		"source":     source,
		"question":   question,
		"display":    display,
	})

	resp, err := s.waitForCapture(ch, 30*time.Second)
	if err != nil {
		return mcpErrorResult(err.Error()), "error", err.Error()
	}
	if !resp.Success {
		return mcpErrorResult(resp.Message), "error", resp.Message
	}
	if s.mcpVisionURL == "" {
		return mcpErrorResult("vision service is not configured"), "error", "vision service is not configured"
	}

	imageBytes, err := decodeCaptureImage(resp.Image)
	if err != nil {
		return mcpErrorResult(err.Error()), "error", err.Error()
	}
	mimeType := resp.MimeType
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	result, err := s.callVision(ctx, imageBytes, mimeType, question)
	if err != nil {
		return mcpErrorResult(err.Error()), "error", err.Error()
	}

	return result, "completed", ""
}

func (s *session) callVision(ctx context.Context, image []byte, mimeType string, question string) (any, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("question", question); err != nil {
		return nil, err
	}
	fileWriter, err := writer.CreateFormFile("file", "capture.jpg")
	if err != nil {
		return nil, err
	}
	if _, err := fileWriter.Write(image); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.mcpVisionURL, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Device-Id", s.deviceID)
	req.Header.Set("Client-Id", s.clientID)
	if s.mcpVisionToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.mcpVisionToken)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(string(payload))
	}

	return parseVisionResponse(payload), nil
}

func parseVisionResponse(payload []byte) any {
	if json.Valid(payload) {
		var parsed any
		if err := json.Unmarshal(payload, &parsed); err == nil {
			return parsed
		}
	}
	return mcpResult{
		Content: []mcpContent{{Type: "text", Text: string(payload)}},
		IsError: false,
	}
}

func (s *session) waitForCapture(ch chan captureResponse, timeout time.Duration) (captureResponse, error) {
	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(timeout):
		return captureResponse{}, errors.New("capture timeout")
	}
}

func (s *session) replyMCPResult(ctx context.Context, id json.RawMessage, result any) {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}
	_ = s.xiaozhi.SendMCP(ctx, payload)
}

func (s *session) replyMCPError(ctx context.Context, id json.RawMessage, message string) {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error":   map[string]any{"message": message},
	}
	_ = s.xiaozhi.SendMCP(ctx, payload)
}

func newRequestID() string {
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

func (s *session) sendToolStatus(toolID string, toolName string, status string, content string) {
	if status == "" {
		return
	}
	payload := map[string]any{
		"type":      "tool_call_status",
		"tool_id":   toolID,
		"tool_name": toolName,
		"status":    status,
		"content":   content,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	s.sendJSON(payload)
}

func mcpIDString(id json.RawMessage) string {
	if len(id) == 0 {
		return ""
	}
	var str string
	if err := json.Unmarshal(id, &str); err == nil {
		return str
	}
	var num float64
	if err := json.Unmarshal(id, &num); err == nil {
		return fmt.Sprintf("%v", num)
	}
	return string(id)
}

func getStringArg(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	value, ok := args[key]
	if !ok {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func decodeCaptureImage(data string) ([]byte, error) {
	if data == "" {
		return nil, errors.New("empty capture image")
	}
	if strings.HasPrefix(data, "data:") {
		parts := strings.SplitN(data, ",", 2)
		if len(parts) != 2 {
			return nil, errors.New("invalid data url")
		}
		return base64.StdEncoding.DecodeString(parts[1])
	}
	return base64.StdEncoding.DecodeString(data)
}

func mcpErrorResult(message string) mcpResult {
	if message == "" {
		message = "capture failed"
	}
	return mcpResult{
		Content: []mcpContent{{Type: "text", Text: message}},
		IsError: true,
	}
}

func (s *session) sendJSON(payload any) {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	if err := s.conn.WriteJSON(payload); err != nil {
		s.logger.Debug("ws send failed", zap.Error(err))
	}
}

func fallbackID(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func computeVolumes(pcm []byte, sampleRate int, channels int, frameDuration int) []float64 {
	if len(pcm) == 0 || sampleRate <= 0 || channels <= 0 {
		return nil
	}
	samples := len(pcm) / 2
	if samples == 0 {
		return nil
	}
	frames := samples / channels
	if frames == 0 {
		return nil
	}
	chunkSize := sampleRate * frameDuration / 1000
	if chunkSize <= 0 {
		chunkSize = frames
	}

	volumes := make([]float64, 0, (frames+chunkSize-1)/chunkSize)
	for start := 0; start < frames; start += chunkSize {
		end := start + chunkSize
		if end > frames {
			end = frames
		}
		rms := rmsPCM(pcm, channels, start, end)
		volumes = append(volumes, rms)
	}

	maxVolume := 0.0
	for _, v := range volumes {
		if v > maxVolume {
			maxVolume = v
		}
	}
	if maxVolume == 0 {
		for i := range volumes {
			volumes[i] = 0
		}
		return volumes
	}
	for i := range volumes {
		volumes[i] = volumes[i] / maxVolume
	}
	return volumes
}

func rmsPCM(pcm []byte, channels int, startFrame int, endFrame int) float64 {
	if startFrame >= endFrame {
		return 0
	}
	sum := 0.0
	count := 0
	for frame := startFrame; frame < endFrame; frame++ {
		for ch := 0; ch < channels; ch++ {
			idx := (frame*channels + ch) * 2
			if idx+2 > len(pcm) {
				return finalizeRMS(sum, count)
			}
			sample := int16(binary.LittleEndian.Uint16(pcm[idx : idx+2]))
			value := float64(sample)
			sum += value * value
			count++
		}
	}
	return finalizeRMS(sum, count)
}

func finalizeRMS(sum float64, count int) float64 {
	if count == 0 {
		return 0
	}
	return math.Sqrt(sum / float64(count))
}

func (h *Handler) registerSession(sess *session) {
	h.mu.Lock()
	h.sessions[sess.clientUID] = sess
	h.mu.Unlock()
	h.group.RegisterClient(sess.clientUID)
}

func (h *Handler) unregisterSession(clientUID string) {
	h.mu.Lock()
	delete(h.sessions, clientUID)
	h.mu.Unlock()
	affected := h.group.RemoveClient(clientUID)
	h.broadcastGroupUpdate(affected)
}

func (h *Handler) sendGroupUpdate(clientUID string) {
	h.mu.Lock()
	sess := h.sessions[clientUID]
	h.mu.Unlock()
	if sess == nil {
		return
	}
	members := h.group.GetGroupMembers(clientUID)
	isOwner := h.group.IsOwner(clientUID)
	sess.sendJSON(map[string]any{"type": "group-update", "members": members, "is_owner": isOwner})
}

func (h *Handler) broadcastGroupUpdate(memberIDs []string) {
	for _, memberID := range memberIDs {
		h.sendGroupUpdate(memberID)
	}
}

func float64ToFloat32(samples []float64) []float32 {
	if len(samples) == 0 {
		return nil
	}
	out := make([]float32, len(samples))
	for i, sample := range samples {
		out[i] = float32(sample)
	}
	return out
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
