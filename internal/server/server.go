package server

import (
	"gofr.dev/pkg/gofr"
)

func New() *gofr.App {
	app := gofr.New()

	registerRoutes(app)

	return app
}
