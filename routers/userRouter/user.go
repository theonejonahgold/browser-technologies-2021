package userRouter

import (
	"bt/db"
	"bt/db/models"
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/matthewhartstonge/argon2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var store *session.Store

func NewRouter(app *fiber.App, sessStore *session.Store) {
	store = sessStore
	app.Get("/login", loginPage)
	app.Post("/login", loginUser)
	app.Get("/register", registerPage)
	app.Post("/register", registerUser)
}

func loginPage(c *fiber.Ctx) error {
	err := c.Query("error", "")
	errs := struct {
		Username string
		Password string
	}{
		Username: "",
		Password: "",
	}
	if strings.Contains(err, "password") {
		if strings.Contains(err, "invalid") {
			errs.Password = "Password is invalid"
		}
	}
	return c.Render("pages/login", fiber.Map{
		"errors": errs,
	}, "layouts/main")
}

func loginUser(c *fiber.Ctx) error {
	var ui models.UserInput
	err := c.BodyParser(&ui)
	if err != nil {
		return err
	}
	ctx, stop := context.WithTimeout(context.Background(), 10*time.Second)
	defer stop()
	cl := db.Database().Collection("users")
	var u models.User
	err = cl.
		FindOne(ctx, bson.M{
			"username": ui.Username,
		}).
		Decode(&u)
	if err != nil {
		return err
	}
	ok, err := argon2.VerifyEncoded([]byte(ui.Password), []byte(u.Password))
	if err != nil {
		return err
	}
	if !ok {
		return c.Redirect("/login?error=password_invalid")
	}
	sess, err := store.Get(c)
	if err != nil {
		return err
	}
	defer sess.Save()
	sess.Set("user", u)
	return c.Redirect("/app/")
}

func registerPage(c *fiber.Ctx) error {
	err := c.Query("error", "")
	errs := struct {
		Username string
		Password string
	}{
		Username: "",
		Password: "",
	}
	if strings.Contains(err, "username") {
		if strings.Contains(err, "exists") {
			errs.Username = "Username already exists"
		}
	}
	return c.Render("pages/register", fiber.Map{
		"errors": errs,
	}, "layouts/main")
}

func registerUser(c *fiber.Ctx) error {
	var u models.UserInput
	err := c.BodyParser(&u)
	if err != nil {
		return err
	}
	ctx, stop := createCtx()
	cl := db.Database().Collection("users")
	_, err = cl.FindOne(ctx, bson.M{
		"username": u.Username,
	}).DecodeBytes()
	if err != mongo.ErrNoDocuments {
		return c.Redirect("/register?error=username_exists")
	}
	stop()
	enc, err := models.HashPassword([]byte(u.Password))
	if err != nil {
		return err
	}
	u.Password = string(enc)

	ctx, stop = createCtx()
	defer stop()
	_, err = cl.InsertOne(ctx, u)
	if err != nil {
		return err
	}
	return c.Redirect("/login")
}

func createCtx(timeout ...int) (context.Context, context.CancelFunc) {
	to := 0
	if len(timeout) == 0 {
		to = 10
	} else {
		to = timeout[0]
	}

	return context.WithTimeout(context.Background(), time.Duration(to)*time.Second)
}
