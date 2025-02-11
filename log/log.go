package log

import (
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/rs/zerolog"
)

var logger zerolog.Logger

func toLogLevel(logLevel string) zerolog.Level {
	// 根据命令行标志设置日志级别
	var level zerolog.Level
	switch logLevel {
	case "debug":
		level = zerolog.DebugLevel
	case "info":
		level = zerolog.InfoLevel
	case "warn":
		level = zerolog.WarnLevel
	case "error":
		level = zerolog.ErrorLevel
	case "fatal":
		level = zerolog.FatalLevel
	case "panic":
		level = zerolog.PanicLevel
	default:
		level = zerolog.DebugLevel // 默认设置为 DebugLevel
	}
	return level
}

func Init(log *lumberjack.Logger, logLevel string) {
	// 创建 zerolog 日志器并直接将 lumberjack 作为输出
	zerolog.SetGlobalLevel(toLogLevel(logLevel))
	logger = zerolog.New(log).Hook(&fileAndLineHook{}).With().Timestamp().Logger()
}

func Logger() *zerolog.Logger {
	return &logger
}
