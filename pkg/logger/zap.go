package logger

import (
	"os"
	"sync"

	"github.com/vnFuhung2903/vcs-report-service/pkg/env"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type ILogger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
	Sync() error
	With(fields ...zap.Field) ILogger
}

type Logger struct {
	logger *zap.Logger
}

var (
	once sync.Once
)

func LoadLogger(env env.LoggerEnv) (logger *Logger, err error) {
	once.Do(func() {
		logger, err = initLogger(env)
	})
	return logger, err
}

func initLogger(env env.LoggerEnv) (*Logger, error) {
	level, err := zapcore.ParseLevel(env.Level)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll("./logs", 0755); err != nil {
		return nil, err
	}

	writer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   env.FilePath,
		MaxSize:    env.MaxSize,
		MaxAge:     env.MaxAge,
		MaxBackups: env.MaxBackups,
		Compress:   true,
	})

	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
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

	fileCore := zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), writer, level)
	consoleCore := zapcore.NewCore(zapcore.NewConsoleEncoder(encoderCfg), zapcore.AddSync(os.Stdout), level)
	core := zapcore.NewTee(fileCore, consoleCore)

	logger := &Logger{logger: zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zapcore.ErrorLevel))}
	return logger, nil
}

func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.logger.Debug(msg, fields...)
}

func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.logger.Info(msg, fields...)
}

func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.logger.Warn(msg, fields...)
}

func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.logger.Error(msg, fields...)
}

func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.logger.Fatal(msg, fields...)
}

func (l *Logger) Sync() error {
	return l.logger.Sync()
}

func (l *Logger) With(fields ...zap.Field) ILogger {
	return &Logger{logger: l.logger.With(fields...)}
}
