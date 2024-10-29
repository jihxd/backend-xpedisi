package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/jihxd/ekspedisi/bootstrap"
)

func main() {
	app := fiber.New()

	// Panggil InitializeApp hanya dengan app
	bootstrap.InitializeApp(app)

	log.Fatal(app.Listen(":8080"))
}
