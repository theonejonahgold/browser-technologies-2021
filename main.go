package main

import (
	"bt/db"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/handlebars"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal(err)
	}
	err = db.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	engine := handlebars.New("views", ".hbs")
	app := fiber.New(fiber.Config{
		Views: engine,
	})
	app.Get("/", index)
	log.Fatal(app.Listen(":" + port))
}

func index(c *fiber.Ctx) error {
	return c.Render("index", fiber.Map{})
}
