package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

var (
	globalLogger *Logger
	once         sync.Once
)

// Logger 日志器
type Logger struct {
	*slog.Logger
	writer io.Writer
	file   *os.File
	level  slog.Level
	ctx    context.Context
}

// Config 日志配置
type Config struct {
	Level      string // debug, info, warn, error
	OutputPath string // 日志文件路径
	MaxSize    int64  // 单文件最大大小（字节）
	MaxBackups int    // 保留的旧日志文件数量
	MaxAge     int    // 保留天数
	Compress   bool   // 是否压缩旧文件
}

// NewLogger 创建日志器
func NewLogger(level string) *Logger {
	once.Do(func() {
		cfg := &Config{
			Level:      level,
			OutputPath: "./log/app.log",
			MaxSize:    100 * 1024 * 1024, // 100MB
			MaxBackups: 10,
			MaxAge:     30,
			Compress:   true,
		}
		globalLogger = newLogger(cfg)
	})
	return globalLogger
}

// NewLoggerWithConfig 使用配置创建日志器
func NewLoggerWithConfig(cfg *Config) *Logger {
	return newLogger(cfg)
}

func newLogger(cfg *Config) *Logger {
	// 确保日志目录存在
	logDir := filepath.Dir(cfg.OutputPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		slog.Error("failed to create log directory", "error", err)
		os.Exit(1)
	}

	// 打开日志文件
	file, err := os.OpenFile(cfg.OutputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		slog.Error("failed to open log file", "error", err)
		file = nil
	}

	// 创建多写入器（同时输出到文件和标准输出）
	var writer io.Writer
	if file != nil {
		writer = io.MultiWriter(os.Stdout, file)
	} else {
		writer = os.Stdout
	}

	// 解析日志级别
	logLevel := parseLevel(cfg.Level)

	// 创建 slog handler
	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level: logLevel,
	})

	// 创建 logger
	l := slog.New(handler)

	return &Logger{
		Logger: l,
		writer: writer,
		file:   file,
		level:  logLevel,
		ctx:    context.Background(),
	}
}

// parseLevel 解析日志级别
func parseLevel(level string) slog.Level {
	switch level {
	case "debug", "DEBUG":
		return slog.LevelDebug
	case "info", "INFO":
		return slog.LevelInfo
	case "warn", "WARN", "warning", "WARNING":
		return slog.LevelWarn
	case "error", "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithRequestID 添加请求ID字段
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{
		Logger: l.Logger.With("request_id", requestID),
		writer: l.writer,
		file:   l.file,
		level:  l.level,
		ctx:    l.ctx,
	}
}

// WithFields 添加多个字段
func (l *Logger) WithFields(key string, value interface{}, more ...interface{}) *Logger {
	args := append([]interface{}{key, value}, more...)
	return &Logger{
		Logger: l.Logger.With(args...),
		writer: l.writer,
		file:   l.file,
		level:  l.level,
		ctx:    l.ctx,
	}
}

// WithContext 设置上下文
func (l *Logger) WithContext(ctx context.Context) *Logger {
	return &Logger{
		Logger: l.Logger,
		writer: l.writer,
		file:   l.file,
		level:  l.level,
		ctx:    ctx,
	}
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level string) {
	l.level = parseLevel(level)
	handler := slog.NewJSONHandler(l.writer, &slog.HandlerOptions{
		Level: l.level,
	})
	l.Logger = slog.New(handler)
}

// GetLevel 获取当前日志级别
func (l *Logger) GetLevel() slog.Level {
	return l.level
}

// Sync 同步日志（刷新缓冲区）
func (l *Logger) Sync() error {
	return nil
}

// Close 关闭日志器
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// Debug 调试日志
func (l *Logger) Debug(msg string, fields ...interface{}) {
	l.Logger.Debug(msg, fields...)
}

// Info 信息日志
func (l *Logger) Info(msg string, fields ...interface{}) {
	l.Logger.Info(msg, fields...)
}

// Warn 警告日志
func (l *Logger) Warn(msg string, fields ...interface{}) {
	l.Logger.Warn(msg, fields...)
}

// Error 错误日志
func (l *Logger) Error(msg string, fields ...interface{}) {
	l.Logger.Error(msg, fields...)
}

// Fatal 致命错误日志
func (l *Logger) Fatal(msg string, fields ...interface{}) {
	l.Logger.Error(msg, fields...)
	os.Exit(1)
}

// Debugf 格式化调试日志
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Logger.Debug(fmt.Sprintf(format, args...))
}

// Infof 格式化信息日志
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Logger.Info(fmt.Sprintf(format, args...))
}

// Warnf 格式化警告日志
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Logger.Warn(fmt.Sprintf(format, args...))
}

// Errorf 格式化错误日志
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Logger.Error(fmt.Sprintf(format, args...))
}

// Fatalf 格式化致命错误日志
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Logger.Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}
