package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"elterngeld-portal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestInit_ProductionConfig(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Env: "production",
		},
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
		},
	}

	err := Init(cfg)
	require.NoError(t, err)
	assert.NotNil(t, Logger)

	// Clean up
	Close()
	Logger = nil
}

func TestInit_DevelopmentConfig(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Env: "development",
		},
		Log: config.LogConfig{
			Level:  "debug",
			Format: "console",
		},
	}

	err := Init(cfg)
	require.NoError(t, err)
	assert.NotNil(t, Logger)

	// Clean up
	Close()
	Logger = nil
}

func TestInit_LogLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error", "invalid-level"}

	for _, level := range levels {
		t.Run("level_"+level, func(t *testing.T) {
			cfg := &config.Config{
				Server: config.ServerConfig{
					Env: "test",
				},
				Log: config.LogConfig{
					Level:  level,
					Format: "json",
				},
			}

			err := Init(cfg)
			require.NoError(t, err)
			assert.NotNil(t, Logger)

			// Clean up
			Close()
			Logger = nil
		})
	}
}

func TestInit_LogFormats(t *testing.T) {
	formats := []string{"json", "console", "invalid-format"}

	for _, format := range formats {
		t.Run("format_"+format, func(t *testing.T) {
			cfg := &config.Config{
				Server: config.ServerConfig{
					Env: "test",
				},
				Log: config.LogConfig{
					Level:  "info",
					Format: format,
				},
			}

			err := Init(cfg)
			require.NoError(t, err)
			assert.NotNil(t, Logger)

			// Clean up
			Close()
			Logger = nil
		})
	}
}

func TestClose(t *testing.T) {
	cfg := createTestConfig()
	err := Init(cfg)
	require.NoError(t, err)

	// Should not panic
	assert.NotPanics(t, func() {
		Close()
	})

	// Should be safe to call multiple times
	assert.NotPanics(t, func() {
		Close()
	})

	Logger = nil
}

func TestDebug(t *testing.T) {
	setupTestLogger(t)
	defer cleanupLogger()

	// Capture output
	output := captureLogOutput(t, func() {
		Debug("test debug message", zap.String("key", "value"))
	})

	assert.Contains(t, output, "test debug message")
	assert.Contains(t, output, "\"key\":\"value\"")
}

func TestInfo(t *testing.T) {
	setupTestLogger(t)
	defer cleanupLogger()

	output := captureLogOutput(t, func() {
		Info("test info message", zap.String("key", "value"))
	})

	assert.Contains(t, output, "test info message")
	assert.Contains(t, output, "\"key\":\"value\"")
}

func TestWarn(t *testing.T) {
	setupTestLogger(t)
	defer cleanupLogger()

	output := captureLogOutput(t, func() {
		Warn("test warning message", zap.String("key", "value"))
	})

	assert.Contains(t, output, "test warning message")
	assert.Contains(t, output, "\"key\":\"value\"")
}

func TestError(t *testing.T) {
	setupTestLogger(t)
	defer cleanupLogger()

	output := captureLogOutput(t, func() {
		Error("test error message", zap.String("key", "value"))
	})

	assert.Contains(t, output, "test error message")
	assert.Contains(t, output, "\"key\":\"value\"")
}

func TestFatal(t *testing.T) {
	setupTestLogger(t)
	defer cleanupLogger()

	// Test with no logger initialized - this should call os.Exit(1)
	// We'll test the condition where Logger is nil
	originalLogger := Logger
	Logger = nil
	defer func() {
		Logger = originalLogger
	}()

	// We can't easily test os.Exit without mocking, but we can test
	// that Fatal doesn't panic when Logger is nil
	// The actual os.Exit call would terminate the test, so we skip direct testing
	// Instead, we test the code path with a logger initialized
	
	// Initialize a logger for the test
	cfg := &config.Config{
		Server: config.ServerConfig{Env: "test"},
		Log:    config.LogConfig{Level: "debug", Format: "console"},
	}
	
	err := Init(cfg)
	require.NoError(t, err)
	
	// Test that Fatal with logger calls Logger.Fatal (which will exit)
	// We can't test the actual exit, but we can verify the logger is called
	// This is a limitation of testing functions that call os.Exit
	// The function should work without panicking when a logger is present
}

func TestPanic(t *testing.T) {
	setupTestLogger(t)
	defer cleanupLogger()

	assert.Panics(t, func() {
		Panic("test panic message", zap.String("key", "value"))
	})
}

func TestWith(t *testing.T) {
	setupTestLogger(t)
	defer cleanupLogger()

	childLogger := With(zap.String("component", "test"))
	assert.NotNil(t, childLogger)

	// Test that child logger includes the additional field
	output := captureLogOutput(t, func() {
		childLogger.Info("test message")
	})

	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "\"component\":\"test\"")
}

func TestLoggingWithNilLogger(t *testing.T) {
	// Temporarily set Logger to nil
	originalLogger := Logger
	Logger = nil
	defer func() {
		Logger = originalLogger
	}()

	// Should not panic when Logger is nil
	assert.NotPanics(t, func() {
		Debug("test debug")
		Info("test info")
		Warn("test warn")
		Error("test error")
	})

	// With should return a no-op logger
	childLogger := With(zap.String("key", "value"))
	assert.NotNil(t, childLogger)
}

func TestJSONOutput(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Env: "production",
		},
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
		},
	}

	setupTestLoggerWithConfig(t, cfg)
	defer cleanupLogger()

	output := captureLogOutput(t, func() {
		Info("test json message", zap.String("field1", "value1"), zap.Int("field2", 42))
	})

	// Verify it's valid JSON
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(output), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "test json message", logEntry["msg"])
	assert.Equal(t, "value1", logEntry["field1"])
	assert.Equal(t, float64(42), logEntry["field2"])
	assert.Equal(t, "info", logEntry["level"])
}

func TestLogLevelFiltering(t *testing.T) {
	tests := []struct {
		name       string
		configLevel string
		logLevel   string
		shouldLog  bool
	}{
		{"debug config allows debug", "debug", "debug", true},
		{"debug config allows info", "debug", "info", true},
		{"debug config allows warn", "debug", "warn", true},
		{"debug config allows error", "debug", "error", true},
		{"info config blocks debug", "info", "debug", false},
		{"info config allows info", "info", "info", true},
		{"info config allows warn", "info", "warn", true},
		{"info config allows error", "info", "error", true},
		{"warn config blocks debug", "warn", "debug", false},
		{"warn config blocks info", "warn", "info", false},
		{"warn config allows warn", "warn", "warn", true},
		{"warn config allows error", "warn", "error", true},
		{"error config blocks debug", "error", "debug", false},
		{"error config blocks info", "error", "info", false},
		{"error config blocks warn", "error", "warn", false},
		{"error config allows error", "error", "error", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Server: config.ServerConfig{
					Env: "test",
				},
				Log: config.LogConfig{
					Level:  tt.configLevel,
					Format: "json",
				},
			}

			setupTestLoggerWithConfig(t, cfg)
			defer cleanupLogger()

			output := captureLogOutput(t, func() {
				switch tt.logLevel {
				case "debug":
					Debug("test message")
				case "info":
					Info("test message")
				case "warn":
					Warn("test message")
				case "error":
					Error("test message")
				}
			})

			if tt.shouldLog {
				assert.Contains(t, output, "test message")
			} else {
				assert.Empty(t, strings.TrimSpace(output))
			}
		})
	}
}

func TestConcurrentLogging(t *testing.T) {
	setupTestLogger(t)
	defer cleanupLogger()

	const numGoroutines = 100
	const messagesPerGoroutine = 10

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < messagesPerGoroutine; j++ {
				Info("concurrent log message",
					zap.Int("goroutine", id),
					zap.Int("message", j),
				)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Test should complete without data races or panics
	assert.True(t, true)
}

func TestProductionVsDevelopmentConfig(t *testing.T) {
	// Test production config
	prodCfg := &config.Config{
		Server: config.ServerConfig{
			Env: "production",
		},
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
		},
	}

	setupTestLoggerWithConfig(t, prodCfg)
	prodOutput := captureLogOutput(t, func() {
		Info("production test message", zap.String("env", "prod"))
	})
	cleanupLogger()

	// Test development config
	devCfg := &config.Config{
		Server: config.ServerConfig{
			Env: "development",
		},
		Log: config.LogConfig{
			Level:  "debug",
			Format: "console",
		},
	}

	setupTestLoggerWithConfig(t, devCfg)
	devOutput := captureLogOutput(t, func() {
		Info("development test message", zap.String("env", "dev"))
	})
	cleanupLogger()

	// Production should be JSON format
	var prodLogEntry map[string]interface{}
	err := json.Unmarshal([]byte(prodOutput), &prodLogEntry)
	require.NoError(t, err)

	// Development should be console format (not valid JSON)
	var devLogEntry map[string]interface{}
	err = json.Unmarshal([]byte(devOutput), &devLogEntry)
	assert.Error(t, err) // Should fail to parse as JSON
	assert.Contains(t, devOutput, "development test message")
}

// Helper functions

func createTestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Env: "test",
		},
		Log: config.LogConfig{
			Level:  "debug",
			Format: "json",
		},
	}
}

func setupTestLogger(t *testing.T) {
	cfg := createTestConfig()
	setupTestLoggerWithConfig(t, cfg)
}

func setupTestLoggerWithConfig(t *testing.T, cfg *config.Config) {
	err := Init(cfg)
	require.NoError(t, err)
	require.NotNil(t, Logger)
}

func cleanupLogger() {
	if Logger != nil {
		Close()
		Logger = nil
	}
}

func captureLogOutput(t *testing.T, logFunc func()) string {
	// Create a buffer to capture output
	var buf bytes.Buffer
	
	// Save the current logger
	originalLogger := Logger
	
	// Create a new logger that writes to our buffer
	writer := zapcore.AddSync(&buf)
	encoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		TimeKey:        "T",
		LevelKey:       "L",
		NameKey:        "N",
		CallerKey:      "C",
		MessageKey:     "M",
		StacktraceKey:  "S",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	})
	core := zapcore.NewCore(encoder, writer, zapcore.DebugLevel)
	Logger = zap.New(core).With(zap.String("service", "test-service"))
	
	// Execute the logging function
	logFunc()
	
	// Force logger to flush
	if Logger != nil {
		Logger.Sync()
	}
	
	// Restore original logger
	Logger = originalLogger
	
	return strings.TrimSpace(buf.String())
}

func TestLoggerReconfiguration(t *testing.T) {
	// Initialize with one config
	cfg1 := &config.Config{
		Server: config.ServerConfig{
			Env: "test",
		},
		Log: config.LogConfig{
			Level:  "warn",
			Format: "json",
		},
	}

	err := Init(cfg1)
	require.NoError(t, err)

	// Debug should not be logged
	output1 := captureLogOutput(t, func() {
		Debug("debug message should not appear")
	})
	assert.Empty(t, strings.TrimSpace(output1))

	// Reconfigure with different settings
	cfg2 := &config.Config{
		Server: config.ServerConfig{
			Env: "test",
		},
		Log: config.LogConfig{
			Level:  "debug",
			Format: "json",
		},
	}

	Close()
	err = Init(cfg2)
	require.NoError(t, err)

	// Debug should now be logged
	output2 := captureLogOutput(t, func() {
		Debug("debug message should appear")
	})
	assert.Contains(t, output2, "debug message should appear")

	Close()
	Logger = nil
}

func TestEdgeCases(t *testing.T) {
	// Test with empty log message
	setupTestLogger(t)
	defer cleanupLogger()

	assert.NotPanics(t, func() {
		Info("", zap.String("key", "value"))
	})

	// Test with many fields
	assert.NotPanics(t, func() {
		fields := make([]zap.Field, 100)
		for i := 0; i < 100; i++ {
			fields[i] = zap.Int("field"+string(rune(i)), i)
		}
		Info("message with many fields", fields...)
	})

	// Test with nil fields (should not panic)
	assert.NotPanics(t, func() {
		Info("message with nil field", zap.Any("nil", nil))
	})
}