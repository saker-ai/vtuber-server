package ws

import "context"
import "go.uber.org/zap"

type incomingHandler func(context.Context, incomingMessage)

func (s *session) dispatchIncoming(ctx context.Context, msg incomingMessage) {
	handlers := map[string]incomingHandler{
		"text-input":                 s.onTextInput,
		"interrupt-signal":           s.onInterruptSignal,
		"mic-audio-data":             s.onMicAudioData,
		"mic-audio-end":              s.onMicAudioEnd,
		"set-listen-mode":            s.onSetListenMode,
		"mcp-capture-response":       s.onMCPCaptureResponse,
		"frontend-playback-complete": s.onFrontendPlaybackComplete,
		"audio-play-start":           s.onNoop,
		"fetch-configs":              s.onFetchConfigs,
		"switch-config":              s.onSwitchConfig,
		"fetch-backgrounds":          s.onFetchBackgrounds,
		"request-init-config":        s.onRequestInitConfig,
		"fetch-history-list":         s.onFetchHistoryList,
		"fetch-and-set-history":      s.onFetchAndSetHistory,
		"create-new-history":         s.onCreateNewHistory,
		"delete-history":             s.onDeleteHistory,
		"request-group-info":         s.onRequestGroupInfo,
		"add-client-to-group":        s.onAddClientToGroup,
		"remove-client-from-group":   s.onRemoveClientFromGroup,
		"ai-speak-signal":            s.onAISpeakSignal,
		"heartbeat":                  s.onNoop,
	}

	if handler, ok := handlers[msg.Type]; ok {
		handler(ctx, msg)
		return
	}
	s.logger.Debug("ws unknown message type",
		zap.String("session_id", s.clientUID),
		zap.String("type", msg.Type),
	)
}

func (s *session) onTextInput(ctx context.Context, msg incomingMessage) {
	if msg.Text == "" {
		return
	}
	if err := s.xiaozhi.SendTextInput(ctx, msg.Text); err != nil {
		s.sendJSON(map[string]any{"type": "error", "message": err.Error()})
	}
}

func (s *session) onInterruptSignal(ctx context.Context, _ incomingMessage) {
	if err := s.xiaozhi.Abort(ctx); err != nil {
		s.sendJSON(map[string]any{"type": "error", "message": err.Error()})
	}
	s.stateMachine.OnInterrupt()
	s.endConversation()
}

func (s *session) onMicAudioData(ctx context.Context, msg incomingMessage) {
	if msg.AudioPCM != "" {
		s.handleMicAudioPCM(ctx, msg.AudioPCM, msg.AudioRate, msg.AudioCh)
		return
	}
	s.handleMicAudio(ctx, msg.Audio)
}

func (s *session) onMicAudioEnd(ctx context.Context, _ incomingMessage) {
	s.handleMicEnd(ctx)
}

func (s *session) onSetListenMode(_ context.Context, msg incomingMessage) {
	s.handleSetListenMode(msg.ListenMode)
}

func (s *session) onMCPCaptureResponse(_ context.Context, msg incomingMessage) {
	s.handleCaptureResponse(msg)
}

func (s *session) onFrontendPlaybackComplete(_ context.Context, _ incomingMessage) {
	s.sendJSON(map[string]any{"type": "force-new-message"})
}

func (s *session) onFetchConfigs(ctx context.Context, _ incomingMessage) {
	s.handleFetchConfigs(ctx)
}

func (s *session) onSwitchConfig(ctx context.Context, msg incomingMessage) {
	s.handleConfigSwitch(ctx, msg.File)
}

func (s *session) onFetchBackgrounds(ctx context.Context, _ incomingMessage) {
	s.handleFetchBackgrounds(ctx)
}

func (s *session) onRequestInitConfig(ctx context.Context, _ incomingMessage) {
	s.handleInitConfig(ctx)
}

func (s *session) onFetchHistoryList(ctx context.Context, _ incomingMessage) {
	s.handleHistoryList(ctx)
}

func (s *session) onFetchAndSetHistory(ctx context.Context, msg incomingMessage) {
	s.handleFetchHistory(ctx, msg.HistoryUID)
}

func (s *session) onCreateNewHistory(ctx context.Context, _ incomingMessage) {
	s.handleCreateHistory(ctx)
}

func (s *session) onDeleteHistory(ctx context.Context, msg incomingMessage) {
	s.handleDeleteHistory(ctx, msg.HistoryUID)
}

func (s *session) onRequestGroupInfo(ctx context.Context, _ incomingMessage) {
	s.handleGroupInfo(ctx)
}

func (s *session) onAddClientToGroup(ctx context.Context, msg incomingMessage) {
	s.handleAddToGroup(ctx, msg.InviteeUID)
}

func (s *session) onRemoveClientFromGroup(ctx context.Context, msg incomingMessage) {
	s.handleRemoveFromGroup(ctx, msg.TargetUID)
}

func (s *session) onAISpeakSignal(_ context.Context, _ incomingMessage) {
	s.sendJSON(map[string]any{"type": "error", "message": "proactive speak not supported in XiaoZhi mode"})
}

func (s *session) onNoop(_ context.Context, _ incomingMessage) {}
