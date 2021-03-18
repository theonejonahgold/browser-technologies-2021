package main

import (
	"bt/db"
	"bt/routers/appRouter"
	"bt/routers/userRouter"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/template/handlebars"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil && !strings.Contains(err.Error(), "no such file or directory") {
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
	engine.AddFunc("objectid", func(id interface{}) string {
		objid, ok := id.(primitive.ObjectID)
		if !ok {
			return "No valid id passed"
		}
		return objid.Hex()
	})
	engine.AddFunc("len", func(iter interface{}) int {
		switch reflect.TypeOf(iter).Kind() {
		case reflect.Slice:
			fallthrough
		case reflect.Map:
			fallthrough
		case reflect.Array:
			s := reflect.ValueOf(iter)
			return s.Len()
		}
		return 0
	})
	engine.AddFunc("addOne", func(num int) int {
		return num + 1
	})
	app := fiber.New(fiber.Config{
		Views: engine,
	})
	sessStore := session.New()
	app.Use(logger.New(logger.ConfigDefault))
	userRouter.NewRouter(app, sessStore)
	appRouter.NewRouter(app, sessStore)
	app.Get("/", index)
	log.Fatal(app.Listen(":" + port))
	if err := recover(); err != nil {
		fmt.Println(err)
	}
}

func index(c *fiber.Ctx) error {
	return c.Render("pages/index", nil, "layouts/main")
}
