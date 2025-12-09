package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func InitLogger(env string) {
	var config zap.Config

	if env == "production" {
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	var err error
	Logger, err = config.Build()
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}

	if env == "production" {
		Logger = Logger.With(
			zap.String("service", "bekend-api"),
		)
	}

	zap.ReplaceGlobals(Logger)
}

func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}

func GetLogger() *zap.Logger {
	if Logger == nil {
		InitLogger(os.Getenv("APP_ENV"))
	}
	return Logger
}

