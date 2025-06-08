package handler

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr        string
	Db          DbConfig
	Env         string
	ApiUrl      string
	FrontendUrl string
}

type DbConfig struct {
	Addr         string
	MaxOpenConns int
	MaxIdleConns int
	MaxLifetime  time.Duration
}

func Load() *Config {


	
	maxOpenConns, err := strconv.Atoi(os.Getenv("DB_MAX_OPEN_CONNS"))
	if err != nil {
		fmt.Println("DB_MAX_OPEN_CONNS is not set")
		panic("invalid DB_MAX_OPEN_CONNS: " + err.Error())
	}

	maxIdleConns, err := strconv.Atoi(os.Getenv("DB_MAX_IDLE_CONNS"))
	if err != nil {
		panic("invalid DB_MAX_IDLE_CONNS: " + err.Error())
	}

	maxLifetime, err := time.ParseDuration(os.Getenv("DB_MAX_LIFETIME"))
	if err != nil {
		panic("invalid DB_MAX_LIFETIME: " + err.Error())
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &Config{
		Addr: port,
		Db: DbConfig{
			Addr:         os.Getenv("DB_ADDR"),
			MaxOpenConns: maxOpenConns,
			MaxIdleConns: maxIdleConns,
			MaxLifetime:  maxLifetime,
		},
		Env:         os.Getenv("ENV"),
		ApiUrl:      os.Getenv("API_URL"),
		FrontendUrl: os.Getenv("FRONTEND_URL"),
	}
}
