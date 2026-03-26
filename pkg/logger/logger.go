package logger

import (
	"fmt"
	"io"
	"os"
	"path"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type LogConfig struct {
	Level      string
	Format     string
	OutputPath string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
}

func NewZapLogger(level string) (*zap.Logger, error) {
	return NewZapLoggerWithConfig(&LogConfig{
		Level:      level,
		Format:     "json",
		OutputPath: "stdout",
	})
}

func NewZapLoggerWithConfig(cfg *LogConfig) (*zap.Logger, error) {
	var zapLevel zapcore.Level
	switch cfg.Level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	case "fatal":
		zapLevel = zapcore.FatalLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	var encoder zapcore.Encoder
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	if cfg.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	var writeSyncer zapcore.WriteSyncer

	if cfg.OutputPath == "" || cfg.OutputPath == "stdout" {
		writeSyncer = zapcore.AddSync(os.Stdout)
	} else if cfg.OutputPath == "stderr" {
		writeSyncer = zapcore.AddSync(os.Stderr)
	} else {
		if cfg.MaxSize <= 0 {
			cfg.MaxSize = 100
		}
		if cfg.MaxBackups <= 0 {
			cfg.MaxBackups = 30
		}
		if cfg.MaxAge <= 0 {
			cfg.MaxAge = 7
		}

		dir := path.Dir(cfg.OutputPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		lumberJackLogger := &lumberjack.Logger{
			Filename:   cfg.OutputPath,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}
		writeSyncer = zapcore.AddSync(lumberJackLogger)
	}

	core := zapcore.NewCore(
		encoder,
		writeSyncer,
		zapLevel,
	)

	return zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1)), nil
}

func NewStdLogger() *zap.Logger {
	logger, _ := NewZapLogger("info")
	return logger
}

func NewHelper(logger *zap.Logger) *zap.SugaredLogger {
	return logger.Sugar()
}

var defaultLogger *zap.SugaredLogger
var defaultMu sync.RWMutex

func init() {
	defaultLogger = NewStdLogger().Sugar()
}

func Default() *zap.SugaredLogger {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultLogger
}

func SetDefault(logger *zap.SugaredLogger) {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	defaultLogger = logger
}

func NewConsoleLogger() *zap.Logger {
	logger, _ := NewZapLoggerWithConfig(&LogConfig{
		Level:      "info",
		Format:     "console",
		OutputPath: "stdout",
	})
	return logger
}

func GetLogger(level string) (*zap.Logger, error) {
	return NewZapLogger(level)
}

func GetLoggerWithConfig(cfg *LogConfig) (*zap.Logger, error) {
	return NewZapLoggerWithConfig(cfg)
}

func Infow(msg string, keysAndValues ...interface{}) {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	defaultLogger.Infow(msg, keysAndValues...)
}

func Errorw(msg string, keysAndValues ...interface{}) {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	defaultLogger.Errorw(msg, keysAndValues...)
}

func Debugw(msg string, keysAndValues ...interface{}) {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	defaultLogger.Debugw(msg, keysAndValues...)
}

func Warnw(msg string, keysAndValues ...interface{}) {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	defaultLogger.Warnw(msg, keysAndValues...)
}

func Fatalw(msg string, keysAndValues ...interface{}) {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	defaultLogger.Fatalw(msg, keysAndValues...)
}

func Sync() {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	if defaultLogger != nil {
		defaultLogger.Sync()
	}
}

type Logger interface {
	Infow(msg string, keysAndValues ...interface{})
	Errorw(msg string, keysAndValues ...interface{})
	Debugw(msg string, keysAndValues ...interface{})
	Warnw(msg string, keysAndValues ...interface{})
	Fatalw(msg string, keysAndValues ...interface{})
}

type NopLogger struct{}

func (l NopLogger) Infow(_ string, _ ...interface{})  {}
func (l NopLogger) Errorw(_ string, _ ...interface{}) {}
func (l NopLogger) Debugw(_ string, _ ...interface{}) {}
func (l NopLogger) Warnw(_ string, _ ...interface{})  {}
func (l NopLogger) Fatalw(_ string, _ ...interface{}) {}

var _ Logger = NopLogger{}

type multiLogger struct {
	loggers []Logger
}

func NewMultiLogger(loggers ...Logger) Logger {
	return &multiLogger{loggers: loggers}
}

func (l *multiLogger) Infow(msg string, keysAndValues ...interface{}) {
	for _, logger := range l.loggers {
		logger.Infow(msg, keysAndValues...)
	}
}

func (l *multiLogger) Errorw(msg string, keysAndValues ...interface{}) {
	for _, logger := range l.loggers {
		logger.Errorw(msg, keysAndValues...)
	}
}

func (l *multiLogger) Debugw(msg string, keysAndValues ...interface{}) {
	for _, logger := range l.loggers {
		logger.Debugw(msg, keysAndValues...)
	}
}

func (l *multiLogger) Warnw(msg string, keysAndValues ...interface{}) {
	for _, logger := range l.loggers {
		logger.Warnw(msg, keysAndValues...)
	}
}

func (l *multiLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	for _, logger := range l.loggers {
		logger.Fatalw(msg, keysAndValues...)
	}
}

var _ Logger = (*multiLogger)(nil)

type writerWrapper struct {
	io.Writer
	logger Logger
}

func (w *writerWrapper) Write(p []byte) (int, error) {
	w.logger.Infow("log", "message", string(p))
	return len(p), nil
}
