package cli

import (
	"github.com/ruohao1/penta/internal/config"
	"github.com/ruohao1/penta/internal/storage/sqlite"
)

type App struct {
	Config *config.Config
	DB     *sqlite.DB
}
