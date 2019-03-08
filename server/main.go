package main

import (
	"flag"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"net/http"
	"os"

	"os/signal"
	"time"

	"github.com/dato-live/golazy/server/config"
	"github.com/dato-live/golazy/server/logs"
	"github.com/dato-live/golazy/server/store"

	//数据存储后端适配器
	_ "github.com/dato-live/golazy/server/adapter/mysql"
	_ "github.com/dato-live/golazy/server/adapter/sqlite"
)

var globals struct {
	sessionStore *SessionStore
	grpcServer   *grpc.Server
	configs      config.Config
}

var logger *zap.Logger

func main() {

	runType := flag.String("type", "server", "RPC server run type, one for server [server], one for init db [initdb].")
	reset := flag.Bool("reset", false, "reset the database")
	flag.Parse()

	configs := config.LoadConfig("conf.yaml")
	//检查配置参数是否合法
	configs.CheckConfig()

	globals.configs = configs
	logs.InitLogger(configs.LogFile, configs.LogLevel, configs.LogToConsole)
	logger = logs.GetLogger()
	if *runType == "initdb" {
		logger.Info("Init db model", zap.String("run_type", *runType), zap.Bool("reset", *reset))
		store.InitDb(configs, *reset)
		defer func() {
			store.Close()
			logger.Info("Closed database connections")
			logger.Info("All done, good bye")
		}()
	} else {
		//
		logger.Info("Run as RPC server model", zap.String("run_type", *runType), zap.Bool("reset", *reset))
		var err error

		err = store.Open(configs)
		if err != nil {
			logger.Fatal("Failed to connect to DB", zap.Error(err))
		}
		defer func() {
			store.Close()
			logger.Info("Closed database connections")
			logger.Info("All done, good bye")
		}()
		go store.DbClearLoop()

		globals.sessionStore = NewSessionStore(time.Duration(configs.IdleSessionTimeoutSecond)*time.Second + 15*time.Second)
		globals.grpcServer, err = serveGrpc(configs.GrpcListen)
		if err != nil {
			logger.Fatal("Grpc server start error", zap.Error(err))
		}

		go RetrySendMsgLoop()

		//http service
		mu := http.NewServeMux()
		mu.HandleFunc("/v0/channels", serveWebSocket)

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)
		<-c
		logger.Info("Ctrl+C or Killed signal received,Graceful Exiting...")
		globals.grpcServer.GracefulStop()
		logger.Info("Graceful Exited.")
	}
}
