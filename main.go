package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

func main() {
	log.Fatal(godotenv.Load())
	app := fiber.New()
	app.Get("/", index)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Fatal(app.Listen(fmt.Sprintf(":%v", port)))
}

func index(c *fiber.Ctx) error {
	return c.SendString("Hi there! ✌️")
}
