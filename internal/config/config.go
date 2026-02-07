package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	appdefaults "github.com/saker-ai/vtuber-server/config"

	"github.com/saker-ai/vtuber-server/internal/logger"
	"github.com/spf13/viper"
)

// SystemConfig represents a systemConfig.
type SystemConfig struct {
	Host                   string `mapstructure:"host"`
	Port                   int    `mapstructure:"port"`
	ConfigAltsDir          string `mapstructure:"config_alts_dir"`
	XiaoZhiBackendURL      string `mapstructure:"xiaozhi_backend_url"`
	XiaoZhiProtocolVersion int    `mapstructure:"xiaozhi_protocol_version"`
	XiaoZhiAudioFormat     string `mapstructure:"xiaozhi_audio_format"`
	XiaoZhiSampleRate      int    `mapstructure:"xiaozhi_sample_rate"`
	XiaoZhiChannels        int    `mapstructure:"xiaozhi_channels"`
	XiaoZhiFrameDuration   int    `mapstructure:"xiaozhi_frame_duration"`
	XiaoZhiListenMode      string `mapstructure:"xiaozhi_listen_mode"`
	XiaoZhiDeviceID        string `mapstructure:"xiaozhi_device_id"`
	XiaoZhiClientID        string `mapstructure:"xiaozhi_client_id"`
	XiaoZhiAccessToken     string `mapstructure:"xiaozhi_access_token"`
}

// CharacterConfig represents a characterConfig.
type CharacterConfig struct {
	ConfName        string `mapstructure:"conf_name"`
	ConfUID         string `mapstructure:"conf_uid"`
	Live2dModelName string `mapstructure:"live2d_model_name"`
	CharacterName   string `mapstructure:"character_name"`
	Avatar          string `mapstructure:"avatar"`
}

// Config represents a config.
type Config struct {
	RootDir                string          `mapstructure:"-"`
	HTTPAddr               string          `mapstructure:"http_addr"`
	XiaoZhiBackendURL      string          `mapstructure:"xiaozhi_backend_url"`
	XiaoZhiProtocolVersion int             `mapstructure:"xiaozhi_protocol_version"`
	XiaoZhiAudioFormat     string          `mapstructure:"xiaozhi_audio_format"`
	XiaoZhiSampleRate      int             `mapstructure:"xiaozhi_sample_rate"`
	XiaoZhiChannels        int             `mapstructure:"xiaozhi_channels"`
	XiaoZhiFrameDuration   int             `mapstructure:"xiaozhi_frame_duration"`
	XiaoZhiListenMode      string          `mapstructure:"xiaozhi_listen_mode"`
	XiaoZhiDeviceID        string          `mapstructure:"xiaozhi_device_id"`
	XiaoZhiClientID        string          `mapstructure:"xiaozhi_client_id"`
	XiaoZhiAccessToken     string          `mapstructure:"xiaozhi_access_token"`
	ConfigAltsDir          string          `mapstructure:"config_alts_dir"`
	ModelDictPath          string          `mapstructure:"model_dict_path"`
	ChatHistoryDir         string          `mapstructure:"chat_history_dir"`
	FrontendDir            string          `mapstructure:"frontend_dir"`
	Live2DModelsDir        string          `mapstructure:"live2d_models_dir"`
	BackgroundsDir         string          `mapstructure:"backgrounds_dir"`
	AvatarsDir             string          `mapstructure:"avatars_dir"`
	AssetsDir              string          `mapstructure:"assets_dir"`
	WebToolDir             string          `mapstructure:"web_tool_dir"`
	TLSCertPath            string          `mapstructure:"tls_cert_path"`
	TLSKeyPath             string          `mapstructure:"tls_key_path"`
	TLSRequired            bool            `mapstructure:"tls_required"`
	TLSDisable             bool            `mapstructure:"tls_disable"`
	SystemConfig           SystemConfig    `mapstructure:"system_config"`
	CharacterConfig        CharacterConfig `mapstructure:"character_config"`
	Log                    logger.Config   `mapstructure:"log"`
}

// Load executes the load function.
func Load() (Config, error) {
	rootDir, err := resolveRootDir()
	if err != nil {
		return Config{}, err
	}

	v := viper.New()
	v.SetConfigName("conf")
	v.SetConfigType("yaml")
	v.AddConfigPath(rootDir)

	if err := v.ReadConfig(bytes.NewReader(appdefaults.Default)); err != nil {
		return Config{}, fmt.Errorf("load embedded config: %w", err)
	}

	v.SetDefault("http_addr", "")
	v.SetDefault("xiaozhi_protocol_version", 1)
	v.SetDefault("xiaozhi_audio_format", "opus")
	v.SetDefault("xiaozhi_sample_rate", 16000)
	v.SetDefault("xiaozhi_channels", 1)
	v.SetDefault("xiaozhi_frame_duration", 20)
	v.SetDefault("xiaozhi_listen_mode", "auto")
	v.SetDefault("tls_required", false)
	v.SetDefault("tls_disable", false)
	v.SetDefault("tls_cert_path", "")
	v.SetDefault("tls_key_path", "")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.stdout", true)
	v.SetDefault("log.file.enabled", true)
	v.SetDefault("log.file.path", "./data/logs")
	v.SetDefault("log.file.name", "vtuber-server.log")
	v.SetDefault("log.file.max_size_mb", 100)
	v.SetDefault("log.file.max_backups", 5)
	v.SetDefault("log.file.max_age_days", 30)
	v.SetDefault("log.file.compress", true)

	v.SetEnvPrefix("mio")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.MergeInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return Config{}, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}

	cfg.RootDir = rootDir
	applySystemConfig(&cfg)
	deriveHTTPAddr(&cfg)
	derivePaths(&cfg)
	deriveCharacterConfig(&cfg)

	return cfg, nil
}

// LoadConfig executes the loadConfig function.
func LoadConfig(configPath string) (Config, error) {
	path := strings.TrimSpace(configPath)
	if path == "" {
		return Load()
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return Config{}, err
	}

	rootDir := strings.TrimSpace(os.Getenv("MIO_ROOT_DIR"))
	if rootDir == "" {
		rootDir = filepath.Dir(absPath)
		if filepath.Base(rootDir) == "config" {
			rootDir = filepath.Dir(rootDir)
		}
	}

	v := viper.New()
	v.SetConfigType("yaml")

	if err := v.ReadConfig(bytes.NewReader(appdefaults.Default)); err != nil {
		return Config{}, fmt.Errorf("load embedded config: %w", err)
	}

	v.SetDefault("http_addr", "")
	v.SetDefault("xiaozhi_protocol_version", 1)
	v.SetDefault("xiaozhi_audio_format", "opus")
	v.SetDefault("xiaozhi_sample_rate", 16000)
	v.SetDefault("xiaozhi_channels", 1)
	v.SetDefault("xiaozhi_frame_duration", 20)
	v.SetDefault("tls_required", false)
	v.SetDefault("tls_disable", false)
	v.SetDefault("tls_cert_path", "")
	v.SetDefault("tls_key_path", "")

	v.SetEnvPrefix("mio")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetConfigFile(absPath)
	if err := v.MergeInConfig(); err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}

	cfg.RootDir = rootDir
	applySystemConfig(&cfg)
	deriveHTTPAddr(&cfg)
	derivePaths(&cfg)
	deriveCharacterConfig(&cfg)

	return cfg, nil
}

func applySystemConfig(cfg *Config) {
	system := cfg.SystemConfig
	if cfg.XiaoZhiBackendURL == "" {
		cfg.XiaoZhiBackendURL = system.XiaoZhiBackendURL
	}
	if cfg.XiaoZhiProtocolVersion == 0 {
		cfg.XiaoZhiProtocolVersion = system.XiaoZhiProtocolVersion
	}
	if cfg.XiaoZhiAudioFormat == "" {
		cfg.XiaoZhiAudioFormat = system.XiaoZhiAudioFormat
	}
	if cfg.XiaoZhiSampleRate == 0 {
		cfg.XiaoZhiSampleRate = system.XiaoZhiSampleRate
	}
	if cfg.XiaoZhiChannels == 0 {
		cfg.XiaoZhiChannels = system.XiaoZhiChannels
	}
	if cfg.XiaoZhiFrameDuration == 0 {
		cfg.XiaoZhiFrameDuration = system.XiaoZhiFrameDuration
	}
	if cfg.XiaoZhiListenMode == "" {
		cfg.XiaoZhiListenMode = system.XiaoZhiListenMode
	}
	if cfg.XiaoZhiDeviceID == "" {
		cfg.XiaoZhiDeviceID = system.XiaoZhiDeviceID
	}
	if cfg.XiaoZhiClientID == "" {
		cfg.XiaoZhiClientID = system.XiaoZhiClientID
	}
	if cfg.XiaoZhiAccessToken == "" {
		cfg.XiaoZhiAccessToken = system.XiaoZhiAccessToken
	}
}

func deriveHTTPAddr(cfg *Config) {
	if cfg.HTTPAddr != "" {
		return
	}
	host := cfg.SystemConfig.Host
	port := cfg.SystemConfig.Port
	if port == 0 {
		port = 8101
	}
	if host == "" {
		cfg.HTTPAddr = fmt.Sprintf(":%d", port)
		return
	}
	cfg.HTTPAddr = net.JoinHostPort(host, strconv.Itoa(port))
}

func resolveRootDir() (string, error) {
	if root := strings.TrimSpace(os.Getenv("MIO_ROOT_DIR")); root != "" {
		return filepath.Abs(root)
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := wd
	for i := 0; i < 6; i++ {
		if fileExists(filepath.Join(dir, "conf.yaml")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return wd, nil
}

func derivePaths(cfg *Config) {
	configAlts := cfg.ConfigAltsDir
	if configAlts == "" {
		configAlts = cfg.SystemConfig.ConfigAltsDir
	}
	cfg.ConfigAltsDir = resolvePath(cfg.RootDir, configAlts, "config_templates")
	cfg.ModelDictPath = resolvePath(cfg.RootDir, cfg.ModelDictPath, filepath.Join("webassets", "model_dict.json"))
	cfg.ChatHistoryDir = resolvePath(cfg.RootDir, cfg.ChatHistoryDir, filepath.Join("data", "vtuber", "chat"))
	cfg.FrontendDir = resolvePath(cfg.RootDir, cfg.FrontendDir, filepath.Join("webassets", "vtuber"))
	cfg.Live2DModelsDir = resolvePath(cfg.RootDir, cfg.Live2DModelsDir, filepath.Join("webassets", "live2d-models"))
	cfg.BackgroundsDir = resolvePath(cfg.RootDir, cfg.BackgroundsDir, filepath.Join("webassets", "backgrounds"))
	cfg.AvatarsDir = resolvePath(cfg.RootDir, cfg.AvatarsDir, "avatars")
	cfg.AssetsDir = resolvePath(cfg.RootDir, cfg.AssetsDir, "assets")
	cfg.WebToolDir = resolvePath(cfg.RootDir, cfg.WebToolDir, "web_tool")
	cfg.TLSCertPath = resolvePath(cfg.RootDir, cfg.TLSCertPath, filepath.Join("certs", "server.crt"))
	cfg.TLSKeyPath = resolvePath(cfg.RootDir, cfg.TLSKeyPath, filepath.Join("certs", "server.key"))
}

func deriveCharacterConfig(cfg *Config) {
	if cfg == nil {
		return
	}
	character := &cfg.CharacterConfig
	if character.Live2dModelName == "" {
		if modelName, err := loadFirstModelName(cfg.ModelDictPath); err == nil {
			character.Live2dModelName = modelName
		}
	}
	if character.ConfUID == "" {
		base := character.ConfName
		if base == "" {
			base = character.Live2dModelName
		}
		character.ConfUID = sanitizeConfUID(base)
	}
	if character.ConfName == "" {
		if character.Live2dModelName != "" {
			character.ConfName = character.Live2dModelName
		} else {
			character.ConfName = character.ConfUID
		}
	}
	if character.CharacterName == "" {
		character.CharacterName = character.ConfName
	}
}

func loadFirstModelName(modelDictPath string) (string, error) {
	data, err := os.ReadFile(modelDictPath)
	if err != nil {
		return "", err
	}
	var entries []modelInfoEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return "", err
	}
	for _, entry := range entries {
		if entry.Name != "" {
			return entry.Name, nil
		}
	}
	return "", fmt.Errorf("no model name found in %s", modelDictPath)
}

func sanitizeConfUID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "default"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_' || r == '-' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "._-")
	if out == "" {
		return "default"
	}
	return out
}

func resolvePath(rootDir string, configured string, fallback string) string {
	path := strings.TrimSpace(configured)
	if path == "" {
		path = fallback
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(rootDir, path)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
