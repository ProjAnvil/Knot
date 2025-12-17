package logger

import (
	"os"
	"time"

	"github.com/ProjAnvil/knot/backend/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log *zap.Logger

// InitLogger initializes the logger based on configuration
func InitLogger(cfg *config.Config) error {
	if !cfg.EnableLogging {
		// Use no-op logger if logging is disabled
		Log = zap.NewNop()
		return nil
	}

	// Ensure log directory exists
	logDir := config.GetLogDir()
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// Configure encoder
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	// Use local time with format: yyyy-MM-dd HH:mm:ss
	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Local().Format("2006-01-02 15:04:05"))
	}

	// Configure log rotation with lumberjack
	logPath := config.GetLogPath()
	logWriter := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    100,  // megabytes per file
		MaxBackups: 30,   // keep 30 old log files
		MaxAge:     30,   // days to retain old log files
		Compress:   true, // compress old files with gzip
	}

	// Create core
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(logWriter),
		zapcore.InfoLevel,
	)

	// Build logger
	Log = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	Log.Info("Logger initialized", zap.String("logPath", logPath))

	return nil
}

// Sync flushes any buffered log entries
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}
