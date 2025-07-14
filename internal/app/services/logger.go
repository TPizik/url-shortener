package services

import "go.uber.org/zap"

var Sugar *zap.SugaredLogger

func InitLogger() *zap.SugaredLogger {
	if Sugar == nil {
		logger, err := zap.NewDevelopment()
		if err != nil {
			panic(err)
		}
		Sugar = logger.Sugar()
	}
	return Sugar
}
