package logger

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/vnFuhung2903/vcs-report-service/pkg/env"
	"go.uber.org/zap"
)

type LoggerSuite struct {
	suite.Suite
	tempDir    string
	testLogger ILogger
	logBuffer  *bytes.Buffer
}

func (suite *LoggerSuite) SetupSuite() {
	tempDir, err := os.MkdirTemp("", "zap_test")
	suite.Require().NoError(err)
	suite.tempDir = tempDir
}

func (suite *LoggerSuite) TearDownSuite() {
	os.RemoveAll(suite.tempDir)
	os.Remove("./logs")
}

func (suite *LoggerSuite) SetupTest() {
	once = sync.Once{}
	suite.logBuffer = &bytes.Buffer{}
}

func (suite *LoggerSuite) TearDownTest() {
	if suite.testLogger != nil {
		suite.testLogger.Sync()
	}
}

func TestLoggerSuite(t *testing.T) {
	suite.Run(t, new(LoggerSuite))
}

func (suite *LoggerSuite) TestLoadLogger() {
	loggerEnv := env.LoggerEnv{
		Level:      "info",
		FilePath:   filepath.Join(suite.tempDir, "test.log"),
		MaxSize:    10,
		MaxAge:     7,
		MaxBackups: 3,
	}

	logger, err := LoadLogger(loggerEnv)

	suite.NoError(err)
	suite.NotNil(logger)
	suite.NotNil(logger.logger)
}

func (suite *LoggerSuite) TestLoadLoggerInvalidLevel() {
	loggerEnv := env.LoggerEnv{
		Level:      "invalid_level",
		FilePath:   filepath.Join(suite.tempDir, "test.log"),
		MaxSize:    10,
		MaxAge:     7,
		MaxBackups: 3,
	}

	logger, err := LoadLogger(loggerEnv)

	suite.Error(err)
	suite.Nil(logger)
	suite.Contains(err.Error(), "unrecognized level")
}

func (suite *LoggerSuite) TestInitLogger() {
	loggerEnv := env.LoggerEnv{
		Level:      "warn",
		FilePath:   filepath.Join(suite.tempDir, "init_test.log"),
		MaxSize:    20,
		MaxAge:     14,
		MaxBackups: 5,
	}

	logger, err := initLogger(loggerEnv)

	suite.NoError(err)
	suite.NotNil(logger)
}

func (suite *LoggerSuite) TestInitLoggerInvalidLogPath() {
	invalidPath := "/root/cannot_write_here.log"

	loggerEnv := env.LoggerEnv{
		Level:      "info",
		FilePath:   invalidPath,
		MaxSize:    10,
		MaxAge:     7,
		MaxBackups: 3,
	}

	logger, err := initLogger(loggerEnv)

	if err != nil {
		suite.Error(err)
		suite.Nil(logger)
	} else {
		suite.NoError(err)
		suite.NotNil(logger)
		suite.testLogger = logger
	}
}

func (suite *LoggerSuite) TestLoggerDebug() {
	loggerEnv := env.LoggerEnv{
		Level:      "debug",
		FilePath:   filepath.Join(suite.tempDir, "debug_test.log"),
		MaxSize:    10,
		MaxAge:     7,
		MaxBackups: 3,
	}

	logger, err := initLogger(loggerEnv)
	suite.Require().NoError(err)
	suite.testLogger = logger

	logger.Debug("Debug message", zap.String("key", "value"))

	logContent, err := os.ReadFile(loggerEnv.FilePath)
	suite.NoError(err)
	suite.Contains(string(logContent), "Debug message")
	suite.Contains(string(logContent), "debug")
}

func (suite *LoggerSuite) TestLoggerInfo() {
	loggerEnv := env.LoggerEnv{
		Level:      "info",
		FilePath:   filepath.Join(suite.tempDir, "info_test.log"),
		MaxSize:    10,
		MaxAge:     7,
		MaxBackups: 3,
	}

	logger, err := initLogger(loggerEnv)
	suite.Require().NoError(err)
	suite.testLogger = logger

	logger.Info("Info message", zap.Int("count", 42))

	logContent, err := os.ReadFile(loggerEnv.FilePath)
	suite.NoError(err)
	suite.Contains(string(logContent), "Info message")
	suite.Contains(string(logContent), "info")
	suite.Contains(string(logContent), "42")
}

func (suite *LoggerSuite) TestLoggerWarn() {
	loggerEnv := env.LoggerEnv{
		Level:      "warn",
		FilePath:   filepath.Join(suite.tempDir, "warn_test.log"),
		MaxSize:    10,
		MaxAge:     7,
		MaxBackups: 3,
	}

	logger, err := initLogger(loggerEnv)
	suite.Require().NoError(err)
	suite.testLogger = logger

	logger.Warn("Warning message", zap.Bool("important", true))

	logContent, err := os.ReadFile(loggerEnv.FilePath)
	suite.NoError(err)
	suite.Contains(string(logContent), "Warning message")
	suite.Contains(string(logContent), "warn")
}

func (suite *LoggerSuite) TestLoggerError() {
	loggerEnv := env.LoggerEnv{
		Level:      "error",
		FilePath:   filepath.Join(suite.tempDir, "error_test.log"),
		MaxSize:    10,
		MaxAge:     7,
		MaxBackups: 3,
	}

	logger, err := initLogger(loggerEnv)
	suite.Require().NoError(err)
	suite.testLogger = logger

	testErr := errors.New("test error for logging")
	logger.Error("Error message", zap.Error(testErr))

	logContent, err := os.ReadFile(loggerEnv.FilePath)
	suite.NoError(err)
	suite.Contains(string(logContent), "Error message")
	suite.Contains(string(logContent), "error")
}

func (suite *LoggerSuite) TestLoggerFatal() {
	if os.Getenv("BE_CRASHER") == "1" {
		loggerEnv := env.LoggerEnv{
			Level:      "error",
			FilePath:   os.Getenv("FATAL_LOG_PATH"),
			MaxSize:    10,
			MaxAge:     7,
			MaxBackups: 3,
		}

		logger, err := initLogger(loggerEnv)
		if err != nil {
			panic(err)
		}

		testErr := errors.New("test fatal for logging")
		logger.Fatal("Fatal message", zap.Error(testErr))
		return
	}

	logPath := filepath.Join(suite.tempDir, "fatal_test.log")

	cmd := exec.Command(os.Args[0], "-test.run=TestLoggerSuite/TestLoggerFatal")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1", "FATAL_LOG_PATH="+logPath)

	err := cmd.Run()

	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		logContent, readErr := os.ReadFile(logPath)
		suite.NoError(readErr)
		suite.Contains(string(logContent), "Fatal message")
		suite.Contains(string(logContent), "fatal")
	} else {
		suite.Fail("Expected subprocess to exit due to Fatal call")
	}
}

func (suite *LoggerSuite) TestLoggerWith() {
	loggerEnv := env.LoggerEnv{
		Level:      "info",
		FilePath:   filepath.Join(suite.tempDir, "with_test.log"),
		MaxSize:    10,
		MaxAge:     7,
		MaxBackups: 3,
	}

	logger, err := initLogger(loggerEnv)
	suite.Require().NoError(err)
	suite.testLogger = logger

	childLogger := logger.With(zap.String("module", "test"), zap.Int("version", 1))

	suite.Implements((*ILogger)(nil), childLogger)

	childLogger.Info("Message with context")

	logContent, err := os.ReadFile(loggerEnv.FilePath)
	suite.NoError(err)

	logStr := string(logContent)
	suite.Contains(logStr, "Message with context")
	suite.Contains(logStr, "module")
	suite.Contains(logStr, "test")
	suite.Contains(logStr, "version")
}

func (suite *LoggerSuite) TestLoggerSync() {
	loggerEnv := env.LoggerEnv{
		Level:      "info",
		FilePath:   filepath.Join(suite.tempDir, "sync_test.log"),
		MaxSize:    10,
		MaxAge:     7,
		MaxBackups: 3,
	}

	logger, err := initLogger(loggerEnv)
	suite.Require().NoError(err)
	suite.testLogger = logger

	logger.Info("Sync test message")

	err = logger.Sync()
	suite.Error(err)

	logContent, err := os.ReadFile(loggerEnv.FilePath)
	suite.NoError(err)
	suite.Contains(string(logContent), "Sync test message")
}
