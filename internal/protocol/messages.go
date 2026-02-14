package protocol

// ClientCommand represents a command sent from web frontend to vtuber-server.
// It intentionally keeps wire-compatible field names with the current runtime.
type ClientCommand struct {
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
