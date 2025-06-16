package handler

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/LikhithMar14/BidZy/internal/service/mail"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type Config struct {
	Addr        string
	Db          DbConfig
	Env         string
	ApiUrl      string
	FrontendUrl string
	JwtSecret   string
	SMTP        *mail.SMTPConfig
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

	smtpPort, err := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if err != nil {
		panic("invalid SMTP_PORT: " + err.Error())
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
		JwtSecret:   os.Getenv("JWT_SECRET"),
		SMTP:        &mail.SMTPConfig{
			Host: os.Getenv("SMTP_HOST"),
			Port: smtpPort,
			User: os.Getenv("SMTP_USER"),
			Pass: os.Getenv("SMTP_PASS"),
			From: os.Getenv("SMTP_FROM"),
			Secure: os.Getenv("SMTP_SECURE"),
		},
	}
}

var GoogleOauthConfig *oauth2.Config

func InitGoogleOauthConfig(googleClientID, googleClientSecret, googleRedirectURL string) *oauth2.Config {
	GoogleOauthConfig = &oauth2.Config{
		RedirectURL:  googleRedirectURL,
		ClientID:     googleClientID,
		ClientSecret: googleClientSecret,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
	return GoogleOauthConfig

}
