package cli

import (
	"github.com/ruohao1/penta/internal/config"
)

type App struct {
	Config      *config.Config
}

func NewApp() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	
	return &App{
		Config:      cfg,
	}, nil
}

