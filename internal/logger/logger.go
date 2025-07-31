package logger

import (
	"youtube-summarizer/pkg/types"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger implements the types.Logger interface using zap
type Logger struct {
	zap *zap.Logger
}

// New creates a new structured logger
func New(development bool) (*Logger, error) {
	var config zap.Config

	if development {
		config = zap.NewDevelopmentConfig()
		config.Development = true
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
		config.Encoding = "json"
	}

	// Set output paths
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	// Create logger
	zapLogger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return &Logger{zap: zapLogger}, nil
}

// NewWithFile creates a logger that also writes to a file
func NewWithFile(development bool, logFile string) (*Logger, error) {
	var config zap.Config

	if development {
		config = zap.NewDevelopmentConfig()
		config.Development = true
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
		config.Encoding = "json"
	}

	// Set output paths to include both stdout and file
	config.OutputPaths = []string{"stdout", logFile}
	config.ErrorOutputPaths = []string{"stderr", logFile}

	// Create logger
	zapLogger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return &Logger{zap: zapLogger}, nil
}

// Info logs an info message with optional fields
func (l *Logger) Info(msg string, fields ...interface{}) {
	l.zap.Info(msg, l.parseFields(fields...)...)
}

// Error logs an error message with optional fields
func (l *Logger) Error(msg string, err error, fields ...interface{}) {
	zapFields := []zap.Field{zap.Error(err)}
	zapFields = append(zapFields, l.parseFields(fields...)...)
	l.zap.Error(msg, zapFields...)
}

// Debug logs a debug message with optional fields
func (l *Logger) Debug(msg string, fields ...interface{}) {
	l.zap.Debug(msg, l.parseFields(fields...)...)
}

// Warn logs a warning message with optional fields
func (l *Logger) Warn(msg string, fields ...interface{}) {
	l.zap.Warn(msg, l.parseFields(fields...)...)
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return l.zap.Sync()
}

// parseFields converts interface{} pairs to zap.Field
func (l *Logger) parseFields(fields ...interface{}) []zap.Field {
	var zapFields []zap.Field

	// Fields should come in key, value pairs
	for i := 0; i < len(fields)-1; i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			continue
		}
		value := fields[i+1]
		zapFields = append(zapFields, zap.Any(key, value))
	}

	return zapFields
}

// WithFields creates a logger with preset fields
func (l *Logger) WithFields(fields ...interface{}) types.Logger {
	zapFields := l.parseFields(fields...)
	return &Logger{zap: l.zap.With(zapFields...)}
}

// Close closes the logger and flushes any remaining logs
func (l *Logger) Close() error {
	return l.zap.Sync()
}
