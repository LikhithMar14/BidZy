package handler

import (
	"net/http"
	"time"

	"github.com/LikhithMar14/BidZy/internal/service/auction"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)


type Application struct {
	Config *Config
	Logger *zap.SugaredLogger
	Version string
	HubManager *auction.HubManager
}

func NewApplication(cfg *Config, version string, logger *zap.SugaredLogger, hubManager *auction.HubManager) *Application {

	return &Application{
		Config: cfg,
		Logger: logger,
		Version: version,
		HubManager: hubManager,
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