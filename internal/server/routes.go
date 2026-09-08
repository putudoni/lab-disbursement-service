package server

import (
	"gofr.dev/pkg/gofr"
)

func registerRoutes(app *gofr.App) {
	app.GET("/", func(c *gofr.Context) (interface{}, error) {
		return map[string]string{"service": "caffe-latte", "status": "ok"}, nil
	})
}