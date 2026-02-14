package fsm

import "testing"

func TestMachineDefault(t *testing.T) {
	m := New()
	if got := m.State(); got != StateIdle {
		t.Fatalf("state=%s, want %s", got, StateIdle)
	}
	if got := m.Mode(); got != ModeAuto {
		t.Fatalf("mode=%s, want %s", got, ModeAuto)
	}
}

func TestMachineTransitionLifecycleAuto(t *testing.T) {
	m := New()
	m.OnListenStart()
	m.OnAudioCommit()
	m.OnConversationStart()
	m.OnTTSStart()
	m.OnTTSStop()

	if got := m.State(); got != StateListening {
		t.Fatalf("state=%s, want %s", got, StateListening)
	}
}

func TestMachineTransitionLifecycleManual(t *testing.T) {
	m := New()
	m.SetMode("manual")
	m.OnListenStart()
	m.OnTTSStart()
	m.OnTTSStop()

	if got := m.State(); got != StateIdle {
		t.Fatalf("state=%s, want %s", got, StateIdle)
	}
}

func TestMachineTransitionLifecycleRealtime(t *testing.T) {
	m := New()
	m.SetMode("realtime")
	m.OnListenStart()
	m.OnTTSStart()
	m.OnTTSStop()

	if got := m.State(); got != StateListening {
		t.Fatalf("state=%s, want %s", got, StateListening)
	}
}

func TestMachineInvalidForce(t *testing.T) {
	m := New()
	if err := m.Force(State("unknown")); err == nil {
		t.Fatal("Force(unknown) error=nil, want non-nil")
	}
}
