package main

import (
	"bt/db"
	"bt/db/models"
	"bt/isosession"
	"bt/routers/appRouter"
	"bt/routers/userRouter"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/logger"
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
		objID, ok := id.(primitive.ObjectID)
		if !ok {
			return "No valid id passed"
		}
		return objID.Hex()
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
	engine.AddFunc("subOne", func(num int) int {
		return num - 1
	})
	engine.AddFunc("sessionState", func(state models.SessionState) string {
		switch state {
		case models.Creating:
			return "Open up session for participants"
		default:
			return "Continue session"
		}
	})
	engine.AddFunc("stateCreating", func(state models.SessionState) bool {
		return state == models.Creating
	})
	engine.AddFunc("stateFinished", func(state models.SessionState) bool {
		return state == models.Finished
	})
	engine.AddFunc("statePlaying", func(state models.SessionState) bool {
		return state != models.Creating && state != models.Finished
	})
	engine.AddFunc("currentQuestion", func(session models.Session) int {
		for k, v := range session.Questions {
			if v.ID == session.CurrentQuestion {
				return k + 1
			}
		}
		return 0
	})
	engine.AddFunc("totalAnswers", func(session models.Session) int {
		var totalAnswers int
		for _, q := range session.Questions {
			for _, a := range q.Answers {
				totalAnswers += len(a.Participants)
			}
		}
		return totalAnswers
	})
	engine.AddFunc("validIndex", func(idx int, length int) bool {
		return idx > -1 && idx < length
	})
	app := fiber.New(fiber.Config{
		Views: engine,
	})
	app.Use(compress.New(compress.Config{Level: compress.LevelBestCompression}))
	app.Use(logger.New(logger.ConfigDefault))
	app.Static("/static", "./static", fiber.Static{Compress: false})
	sessStore := isosession.NewStore()
	userRouter.NewRouter(app, sessStore)
	appRouter.NewRouter(app, sessStore)
	app.Get("/", index)
	log.Fatal(app.Listen(":" + port))
}

func index(c *fiber.Ctx) error {
	return c.Render("pages/index", nil, "layouts/main")
}
