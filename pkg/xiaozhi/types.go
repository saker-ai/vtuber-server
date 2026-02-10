package xiaozhi

// AudioParams represents a audioParams.
type AudioParams struct {
	Format        string
	OutputFormat  string
	SampleRate    int
	Channels      int
	FrameDuration int
}

// Config represents a config.
type Config struct {
	BackendURL      string
	ProtocolVersion int
	AudioParams     AudioParams
	ListenMode      string
	DeviceID        string
	ClientID        string
	AccessToken     string
	FeatureAEC      bool
}
