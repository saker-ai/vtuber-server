package fsm

import (
	"fmt"
	"strings"
	"sync"
)

// State describes the high-level conversation state for a client session.
type State string

const (
	StateIdle          State = "idle"
	StateListening     State = "listening"
	StateProcessingASR State = "processing_asr"
	StateProcessingLLM State = "processing_llm"
	StateSendingTTS    State = "sending_tts"
	StateInterrupted   State = "interrupted"
)

// Mode affects transition policy for listen and TTS stop behavior.
type Mode string

const (
	ModeAuto     Mode = "auto"
	ModeManual   Mode = "manual"
	ModeRealtime Mode = "realtime"
)

// Machine is a lightweight deterministic session state machine.
type Machine struct {
	mu    sync.RWMutex
	state State
	mode  Mode
}

// New creates a state machine with default idle/auto values.
func New() *Machine {
	return &Machine{
		state: StateIdle,
		mode:  ModeAuto,
	}
}

// State returns the current state.
func (m *Machine) State() State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

// Mode returns the current listen mode.
func (m *Machine) Mode() Mode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mode
}

// SetMode updates policy mode.
func (m *Machine) SetMode(mode string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case string(ModeManual):
		m.mode = ModeManual
	case string(ModeRealtime):
		m.mode = ModeRealtime
	default:
		m.mode = ModeAuto
	}
}

// OnListenStart moves session into listening.
func (m *Machine) OnListenStart() {
	m.transition(StateListening)
}

// OnAudioCommit marks upstream audio collected and awaiting llm.
func (m *Machine) OnAudioCommit() {
	m.transition(StateProcessingASR)
}

// OnConversationStart marks llm/text response processing.
func (m *Machine) OnConversationStart() {
	m.transition(StateProcessingLLM)
}

// OnTTSStart enters speaking state.
func (m *Machine) OnTTSStart() {
	m.transition(StateSendingTTS)
}

// OnTTSStop exits speaking state according to mode policy.
func (m *Machine) OnTTSStop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch m.mode {
	case ModeManual:
		m.state = StateIdle
	default:
		m.state = StateListening
	}
}

// OnInterrupt marks interruption.
func (m *Machine) OnInterrupt() {
	m.transition(StateInterrupted)
}

// Force sets state unconditionally.
func (m *Machine) Force(state State) error {
	switch state {
	case StateIdle, StateListening, StateProcessingASR, StateProcessingLLM, StateSendingTTS, StateInterrupted:
		m.transition(state)
		return nil
	default:
		return fmt.Errorf("invalid state: %s", state)
	}
}

func (m *Machine) transition(state State) {
	m.mu.Lock()
	m.state = state
	m.mu.Unlock()
}
