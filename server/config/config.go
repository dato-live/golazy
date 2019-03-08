package config

import (
	"github.com/dato-live/golazy/server/store/types"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"path/filepath"
)

type MysqlConfig struct {
	DSN      string `yaml:"dsn"`
	Database string `yaml:"database"`
}

type SqliteConfig struct {
	Database string `yaml:"database"`
}

type AdapterConfig struct {
	AdapterType string       `yaml:"adapter_type"`
	Mysql       MysqlConfig  `yaml:"mysql"`
	Sqlite      SqliteConfig `yaml:"sqlite"`
}

type StoreConfig struct {
	Adapters AdapterConfig `yaml:"adapters"`
}

type Config struct {
	LogFile          string `yaml:"log_file"`
	LogLevel         string `yaml:"log_level"`
	LogToConsole     bool   `yaml:"log_to_console"`
	ShowSqlToConsole bool   `yaml:"show_sql_to_console"`
	GrpcListen       string `yaml:"grpc_listen"`
	MaxMessageSize   int64  `yaml:"max_message_size"`

	Store StoreConfig `yaml:"store"`

	IdleSessionTimeoutSecond    int    `yaml:"idle_session_timeout_second"`
	MaxRetryCount               int    `yaml:"max_retry_count"`
	RetrySecondInterval         int    `yaml:"retry_second_interval"`
	CleanDbMinuteInterval       int    `yaml:"clean_db_minute_interval"`
	MessageExpireMinuteInterval int    `yaml:"message_expire_minute_interval"`
	ApiKeySalt                  []byte `yaml:"api_key_salt"`
	TlsStrictMaxAge             string `yaml:"tls_strict_max_age"`
	TlsRedirectHTTP             string `yaml:"tls_redirect_http"`
}

func LoadConfig(configPath string) Config {
	fullPath, err := filepath.Abs(configPath)
	if err != nil {
		log.Fatalf("Get absolute config path failed:%v\n", err)
	}
	log.Printf("Loading config from file: %s\n", fullPath)
	var config Config
	yamlFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Read config file [%s] error:%v\n", fullPath, err)
	}
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatal("Parse config file [%s] error:%v\n", fullPath, err)
	}
	return config
}

func (c *Config) CheckConfig() {
	if c.MaxMessageSize <= 0 {
		c.MaxMessageSize = types.DefaultMaxMessageSize
	}

	if c.MaxRetryCount <= 0 {
		c.MaxRetryCount = types.DefaultMaxRetryCount
	}

	if c.CleanDbMinuteInterval <= 0 {
		c.CleanDbMinuteInterval = types.DefaultCleanDbMinuteInterval
	}

	if c.RetrySecondInterval <= 0 {
		c.RetrySecondInterval = types.DefaultRetrySecondInterval
	}

	if c.IdleSessionTimeoutSecond <= 30 {
		c.IdleSessionTimeoutSecond = types.DefaultIdleSessionTimeoutSecond
	}

	if c.MessageExpireMinuteInterval <= 0 {
		c.MessageExpireMinuteInterval = types.DefaultMessageExpireMinuteInterval
	}

}
