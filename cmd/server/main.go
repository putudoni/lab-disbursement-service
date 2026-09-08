package main

import (
	"lab-disbursement-service/internal/server"
)

func main() {
	app := server.New()
	app.Run()
}
