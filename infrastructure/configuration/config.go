package configuration

import (
	"fmt"
	"my-project/infrastructure/logger"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	Database    Database    `json:"database"`
	TulusTech   TulusTech   `json:"tulusTech"`
	App         App         `json:"app"`
	GoogleSheet GoogleSheet `json:"googleSheet"`
	Data        Data        `json:"data"`
	Pubsub      Pubsub      `json:"pubsub"`
	ServiceBus  ServiceBus  `json:"serviceBus"`
	RedisClient RedisClient `json:"redisClient"`
	Logger      Logger      `json:"logger"`
}

type App struct {
	Port      int    `json:"port"`
	SecretKey string `json:"secretKey"`
}

type Database struct {
	Psql  Db `json:"psql"`
	MySql Db `json:"mysql"`
	Mongo Db `json:"mongo"`
}

type GoogleSheet struct {
	Type                       int    `json:"type"`
	SpreadsheetId              string `json:"spreadsheetId"`
	SpreadsheetColumnReadRange string `json:"spreadsheetColumnReadRange"`
	SpreadsheetName            string `json:"spreadsheetName"`
	SpreadsheetDescription     string `json:"spreadsheetDescription"`
}

type Db struct {
	Name     string `json:"string"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type TulusTech struct {
	Header Header `json:"header"`
	Host   string `json:"host"`
}

type Data struct {
	Source string `json:"source"`
}

type Header struct {
	Accept          string `json:"accept"`
	AcceptLanguage  string `json:"acceptLanguage"`
	Connection      string `json:"connection"`
	ContentType     string `json:"contentType"`
	Cookie          string `json:"cookie"`
	Origin          string `json:"origin"`
	Referer         string `json:"referer"`
	SecFetchDest    string `json:"secFetchDest"`
	SecFetchMode    string `json:"secFetchMode"`
	SectFetchSite   string `json:"secFetchSite"`
	UserAgent       string `json:"userAgent"`
	XRequestedWith  string `json:"xRequestedWith"`
	SecChUa         string `json:"secChUa"`
	SecChUaMobile   string `json:"secChUaMobile"`
	SecChUaPlatform string `json:"secChUaPlatform"`
}

type Pubsub struct {
	ProjectID string `json:"projectID"`
}

type ServiceBus struct {
	Namespace string `json:"namespace"`
}

type RedisClient struct {
	Host         string `json:"host"`
	Port         string `json:"port"`
	Password     string `json:"password"`
	DatabaseName string `json:"databaseName"`
	Username     string `json:"username"`
}

type Logger struct {
	Format string `json:"format"`
}

var C Config

func init() {
	LoadConfig()
	initDatabase(&C)
}

func LoadConfig() {
	name := getConfig()
	viper.SetConfigName(name)
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	viper.AddConfigPath("../")
	viper.AddConfigPath("../../")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			fmt.Println("Config file not found")
		} else {
			// Config file was found but another error was produced
			fmt.Println("An error occurred. ", err)
		}
	}

	logger.GetLogger().WithField("config", name).Info("Config set up successfully")
	// Config file found and successfully parsed
	if err := viper.Unmarshal(&C); err != nil {
		logger.GetLogger().WithField("error", err).Error("Viper unable to decode into struct")
	}
}

func getConfig() string {
	name := "config"
	env := os.Getenv("ENV")
	if env != "" {
		name = fmt.Sprintf("%s-%s", name, env)
	}
	return name
}

func initDatabase(C *Config) {
	logger.GetLogger().WithField("Database", C.Database.Psql).Info("Database configuration")
	if C.Database.Psql.Name == "" {
		C.Database.Psql.Name = os.Getenv("DB_NAME")
	}
	if C.Database.Psql.Host == "" {
		C.Database.Psql.Host = os.Getenv("DB_HOST")
	}
	if C.Database.Psql.Password == "" {
		C.Database.Psql.Password = os.Getenv("DB_PASSWORD")
	}
	if C.Database.Psql.Port == "" {
		C.Database.Psql.Port = os.Getenv("DB_PORT")
	}
}
