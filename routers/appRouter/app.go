package appRouter

import (
	"bt/db"
	"bt/db/models"
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var store *session.Store

func NewRouter(app *fiber.App, sessStore *session.Store) {
	store = sessStore
	router := app.Group("/app")
	router.Use(isLoggedIn)
	router.Get("/", appPage)
	router.Get("/session/create/", createSessionPage)
	router.Post("/session/create/", saveSessionName)
	router.Get("/session/:id", sessionPage)
	router.Get("/session/:id/question", newQuestionPage)
	router.Post("/session/:id/question", saveNewQuestion)
	router.Get("/session/:id/question/:qid", editQuestion)
	router.Post("/session/:id/question/:qid", updateQuestion)
	router.Post("/session/:id/question/:qid/:aid", removeAnswerFromQuestion)
}

func isLoggedIn(c *fiber.Ctx) error {
	sess, err := store.Get(c)
	if err != nil {
		return err
	}
	user := sess.Get("user")
	if user == nil {
		return c.Redirect("/user/login")
	}
	sess.Save()
	return c.Next()
}

func appPage(c *fiber.Ctx) error {
	sess, _ := store.Get(c)
	u, ok := sess.Get("user").(models.User)
	if !ok {
		return c.Redirect("/user/login")
	}
	ctx, stop := createCtx(20)
	defer stop()
	cur, err := db.Database().Collection("sessions").Find(ctx, bson.M{
		"owner": u.ID,
	})
	if err != nil {
		return err
	}
	ctx, stop = createCtx(20)
	defer stop()
	var s []models.Session
	if err = cur.All(ctx, &s); err != nil {
		return err
	}
	return c.Render("pages/app/index", fiber.Map{
		"sessions": s,
		"user":     u,
	}, "layouts/app")
}

func createSessionPage(c *fiber.Ctx) error {
	return c.Render("pages/app/session/create", nil, "layouts/app")
}

func saveSessionName(c *fiber.Ctx) error {
	var si models.SessionInput
	if err := c.BodyParser(&si); err != nil {
		return err
	}
	sess, err := store.Get(c)
	if err != nil {
		return err
	}
	defer sess.Save()
	u, ok := sess.Get("user").(models.User)
	if !ok {
		return c.Redirect("/user/login")
	}
	cl := db.Database().Collection("sessions")
	ctx, stop := createCtx()
	defer stop()
	res, err := cl.InsertOne(ctx, models.Session{
		ID:           primitive.NewObjectID(),
		Name:         si.Name,
		Owner:        u.ID,
		Participants: []primitive.ObjectID{},
	})
	if err != nil {
		return err
	}

	id, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return fmt.Errorf("couldn't convert result to objectid: %v", res.InsertedID)
	}
	return c.Redirect(fmt.Sprintf("/app/session/%v", id.Hex()))
}

func sessionPage(c *fiber.Ctx) error {
	id := c.Params("id")
	sess, err := store.Get(c)
	if err != nil {
		return err
	}
	u, ok := sess.Get("user").(models.User)
	if !ok {
		return c.Redirect("/user/login")
	}
	objid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	var s models.Session
	ctx, stop := createCtx()
	if err = db.Database().Collection("sessions").FindOne(ctx, bson.M{"_id": objid, "owner": u.ID}).Decode(&s); err != nil {
		return err
	}
	stop()
	ctx, stop = createCtx()
	cur, err := db.Database().Collection("questions").Find(ctx, bson.M{"session": objid})
	if err != nil {
		return err
	}
	stop()
	var qs []models.Question
	ctx, stop = createCtx()
	defer stop()
	err = cur.All(ctx, &qs)
	if err != nil {
		return err
	}
	return c.Render("pages/app/session/index", fiber.Map{
		"session":   s,
		"questions": qs,
		"id":        objid.Hex(),
	}, "layouts/app")
}

func newQuestionPage(c *fiber.Ctx) error {
	id := c.Params("id")
	objid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	sess, err := store.Get(c)
	if err != nil {
		return err
	}
	u, ok := sess.Get("user").(models.User)
	if !ok {
		return c.Redirect("/user/login")
	}
	ctx, stop := createCtx()
	defer stop()
	var s models.Session
	if err = db.
		Database().
		Collection("sessions").
		FindOne(ctx, bson.M{"owner": u.ID, "_id": objid}).
		Decode(&s); err == mongo.ErrNoDocuments {
		return c.Redirect("/app/sesssion/create")
	} else if err != nil {
		return err
	}
	return c.Render("pages/app/session/question/new", fiber.Map{
		"session": s,
	}, "layouts/app")
}

func saveNewQuestion(c *fiber.Ctx) error {
	var q models.Question
	if err := c.BodyParser(&q); err != nil {
		return err
	}
	id := c.Params("id")
	objid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	q.ID = primitive.NewObjectID()
	q.Session = objid
	ctx, stop := createCtx()
	defer stop()
	res, err := db.
		Database().
		Collection("questions").
		InsertOne(ctx, q)
	if err != nil {
		return err
	}
	return c.Redirect(fmt.Sprintf("/app/session/%v/question/%v", objid.Hex(), res.InsertedID.(primitive.ObjectID).Hex()))
}

func editQuestion(c *fiber.Ctx) error {
	sid := c.Params("id")
	qid := c.Params("qid")
	sobjid, err := primitive.ObjectIDFromHex(sid)
	if err != nil {
		return err
	}
	qobjid, err := primitive.ObjectIDFromHex(qid)
	if err != nil {
		return err
	}
	ctx, stop := createCtx()
	defer stop()
	var q models.Question
	if err := db.
		Database().
		Collection("questions").
		FindOne(ctx, bson.M{
			"_id":     qobjid,
			"session": sobjid,
		}).
		Decode(&q); err != nil {
		return err
	}
	return c.Render("pages/app/session/question/edit", fiber.Map{
		"question": q,
	}, "layouts/app")
}

func updateQuestion(c *fiber.Ctx) error {
	qid := c.Params("qid")
	objid, err := primitive.ObjectIDFromHex(qid)
	if err != nil {
		return err
	}
	data, err := processForm(string(c.Body()))
	if err != nil {
		return err
	}
	cl := db.Database().Collection("questions")
	var q models.Question
	ctx, stop := createCtx()
	if err := cl.
		FindOne(ctx, bson.M{
			"_id": objid,
		}).Decode(&q); err != nil {
		stop()
		return err
	}
	stop()
	for k, v := range data {
		if len(v) == 0 || v[0] == "" {
			continue
		}
		if k == "answer" {
			q.Answers = append(q.Answers, &models.Answer{
				ID:    primitive.NewObjectID(),
				Title: v[0],
			})
		} else if strings.Contains(k, "answer") {
			i, err := strconv.Atoi(strings.Split(k, "-")[1])
			if err != nil {
				return err
			}
			q.Answers[i].Title = v[0]
		} else if k == "title" {
			q.Title = v[0]
		}
	}
	ctx, stop = createCtx()
	var newQ models.Question
	if err = cl.FindOneAndUpdate(ctx, bson.M{
		"_id": objid,
	}, bson.M{
		"$set": q,
	}).Decode(&newQ); err != nil {
		stop()
		return err
	}
	stop()
	return c.Redirect(fmt.Sprintf("/app/session/%v/question/%v", newQ.Session.Hex(), newQ.ID.Hex()))
}

func removeAnswerFromQuestion(c *fiber.Ctx) error {
	sid := c.Params("id")
	qid := c.Params("qid")
	aid := c.Params("aid")
	objid, err := primitive.ObjectIDFromHex(sid)
	if err != nil {
		return err
	}
	qobjid, err := primitive.ObjectIDFromHex(qid)
	if err != nil {
		return err
	}
	aobjid, err := primitive.ObjectIDFromHex(aid)
	if err != nil {
		return err
	}
	ctx, stop := createCtx()
	var q models.Question
	if err := db.Database().Collection("questions").FindOneAndUpdate(ctx, bson.M{
		"_id": qobjid,
	}, bson.M{
		"$pull": bson.M{
			"answers": bson.M{
				"_id": aobjid,
			},
		},
	}).Decode(&q); err != nil {
		stop()
		return err
	}
	stop()
	return c.Redirect(fmt.Sprintf("/app/session/%v", objid.Hex()))
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

func processForm(body string) (map[string][]string, error) {
	vals, err := url.ParseQuery(body)
	if err != nil {
		return nil, err
	}
	return vals, err
}
