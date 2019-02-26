package logs

import (
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

var globalLogger *zap.Logger

func InitLogger(logPath string, logLevel string, showConsole bool) {
	hook := lumberjack.Logger{
		Filename:   logPath, // 日志文件路径
		MaxSize:    1024,    //日志文件分割大小，单位MB
		MaxBackups: 3,       //最多保留三个备份
		MaxAge:     30,      //最多保留30天，单位天
		Compress:   true,    //是否压缩
	}
	fileWriter := zapcore.AddSync(&hook)
	var level zapcore.Level
	switch logLevel {
	case "debug":
		level = zap.DebugLevel
	case "info":
		level = zap.InfoLevel
	case "warn":
		level = zap.WarnLevel
	case "error":
		level = zap.ErrorLevel
	case "fatal":
		level = zap.FatalLevel
	case "panic":
		level = zap.PanicLevel
	default:
		level = zap.InfoLevel
	}
	var multiCores []zapcore.Core

	if showConsole {
		consoleWriter := zapcore.Lock(os.Stdout)
		consoleEncoderConfig := zap.NewProductionEncoderConfig()
		consoleEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderConfig)
		consoleCore := zapcore.NewCore(consoleEncoder, consoleWriter, level)
		multiCores = append(multiCores, consoleCore)
	}

	fileEncoderConfig := zap.NewProductionEncoderConfig()
	fileEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	fileCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(fileEncoderConfig),
		fileWriter,
		level,
	)
	multiCores = append(multiCores, fileCore)

	core := zapcore.NewTee(multiCores...)

	globalLogger = zap.New(core).WithOptions(zap.AddCaller())
	globalLogger.Info("Logger init success")
}

func GetLogger() *zap.Logger {
	return globalLogger
}
