package handler

import (
	"net/http"
	"time"

	"github.com/LikhithMar14/BidZy/internal/service"
	"github.com/LikhithMar14/BidZy/internal/service/auction"
	"github.com/LikhithMar14/BidZy/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

type Service struct {
	AuthService *service.AuthService

}

type Application struct {
	Config *Config
	Logger *zap.SugaredLogger
	Version string
	HubManager *auction.HubManager
	Rdb *redis.Client
	Service *service.Service
}

func NewApplication(cfg *Config, version string, logger *zap.SugaredLogger, hubManager *auction.HubManager, rdb *redis.Client,store *store.Store, googleOauthClient *oauth2.Config) *Application {

	service := service.NewService(store, cfg.JwtSecret, googleOauthClient)
	return &Application{
		Config: cfg,
		Logger: logger,
		Version: version,
		HubManager: hubManager,
		Rdb: rdb,
		Service: service,
	}
}


func (app *Application) Server(mux *chi.Mux) error {
	server := &http.Server{
		Addr: app.Config.Addr,
		Handler: mux,
		ReadTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout: 10 * time.Second,
		ErrorLog: zap.NewStdLog(app.Logger.Desugar()),
	}

	app.Logger.Infow("server started", "addr", app.Config.Addr)

	if err := server.ListenAndServe(); err != nil {
		return err
	}
	return nil
}