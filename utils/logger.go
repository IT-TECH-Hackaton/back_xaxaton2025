package utils

import (
	"bekend/logger"
	"go.uber.org/zap"
)

func GetLogger() *zap.Logger {
	return logger.GetLogger()
}

