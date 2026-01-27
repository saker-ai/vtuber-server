package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config represents a config.
type Config struct {
	Level  string     `mapstructure:"level" yaml:"level"`
	Stdout bool       `mapstructure:"stdout" yaml:"stdout"`
	File   FileConfig `mapstructure:"file" yaml:"file"`
}

// FileConfig represents a fileConfig.
type FileConfig struct {
	Enabled    bool   `mapstructure:"enabled" yaml:"enabled"`
	Path       string `mapstructure:"path" yaml:"path"`
	Name       string `mapstructure:"name" yaml:"name"`
	MaxSizeMB  int    `mapstructure:"max_size_mb" yaml:"max_size_mb"`
	MaxBackups int    `mapstructure:"max_backups" yaml:"max_backups"`
	MaxAgeDays int    `mapstructure:"max_age_days" yaml:"max_age_days"`
	Compress   bool   `mapstructure:"compress" yaml:"compress"`
}

// New executes the new function.
func New(cfg Config) (*zap.Logger, error) {
	level := parseLevel(cfg.Level)
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
	}
	encoder := zapcore.NewJSONEncoder(encoderCfg)

	sinks, err := buildSinks(cfg)
	if err != nil {
		return nil, err
	}

	core := zapcore.NewCore(encoder, sinks, level)
	return zap.New(core, zap.AddCaller()), nil
}

func buildSinks(cfg Config) (zapcore.WriteSyncer, error) {
	var sinks []zapcore.WriteSyncer

	if cfg.Stdout {
		sinks = append(sinks, zapcore.AddSync(os.Stdout))
	}

	if cfg.File.Enabled {
		fileWriter, err := newFileWriter(cfg.File)
		if err != nil {
			return nil, err
		}
		sinks = append(sinks, zapcore.AddSync(fileWriter))
	}

	if len(sinks) == 0 {
		sinks = append(sinks, zapcore.AddSync(os.Stdout))
	}

	return zapcore.NewMultiWriteSyncer(sinks...), nil
}

func newFileWriter(fileCfg FileConfig) (*lumberjack.Logger, error) {
	dir := strings.TrimSpace(fileCfg.Path)
	if dir == "" {
		dir = "./logs"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create log directory %s: %w", dir, err)
	}

	filename := strings.TrimSpace(fileCfg.Name)
	if filename == "" {
		filename = "vtuber-server.log"
	}

	logger := &lumberjack.Logger{
		Filename:   filepath.Join(dir, filename),
		MaxSize:    fileCfg.MaxSizeMB,
		MaxBackups: fileCfg.MaxBackups,
		MaxAge:     fileCfg.MaxAgeDays,
		Compress:   fileCfg.Compress,
		LocalTime:  true,
	}
	if logger.MaxSize <= 0 {
		logger.MaxSize = 100
	}
	if logger.MaxBackups < 0 {
		logger.MaxBackups = 0
	}
	if logger.MaxAge < 0 {
		logger.MaxAge = 0
	}
	return logger, nil
}

func parseLevel(raw string) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return zapcore.DebugLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "info", "":
		return zapcore.InfoLevel
	default:
		return zapcore.InfoLevel
	}
}
