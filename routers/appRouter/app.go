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
	router.Post("/session/delete/:id", deleteSession)
	router.Get("/session/:id", sessionPage)
	router.Get("/session/:id/question/create", newQuestionPage)
	router.Post("/session/:id/question/create", saveNewQuestion)
	router.Get("/session/:id/question/edit/:qid", editQuestionPage)
	router.Post("/session/:id/question/edit/:qid", editQuestion)
	router.Post("/session/:id/question/delete/:qid", deleteQuestion)
	router.Post("/session/:id/answer/delete/:aid", removeAnswerFromQuestion)
}

func isLoggedIn(c *fiber.Ctx) error {
	sess, err := store.Get(c)
	if err != nil {
		return err
	}
	user := sess.Get("user")
	if user == nil {
		return c.Redirect("/login")
	}
	sess.Save()
	return c.Next()
}

func appPage(c *fiber.Ctx) error {
	sess, _ := store.Get(c)
	u := sess.Get("user").(models.User)
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
		return c.Redirect("/login")
	}
	cl := db.Database().Collection("sessions")
	id := primitive.NewObjectID()
	ctx, stop := createCtx()
	defer stop()
	if _, err = cl.InsertOne(ctx, models.Session{
		ID:            id,
		Name:          si.Name,
		Owner:         u.ID,
		Participants:  []primitive.ObjectID{},
		QuestionTimer: 0,
		Questions:     []*models.Question{},
		Code:          fmt.Sprintf("%v-%v", u.Username, id.Hex()[len(id.Hex())-8:]),
	}); err != nil {
		return err
	}
	return c.Redirect(fmt.Sprintf("/app/session/%v", id.Hex()))
}

func deleteSession(c *fiber.Ctx) error {
	id := c.Params("id")
	objid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	sess, _ := store.Get(c)
	u := sess.Get("user").(models.User)
	ctx, stop := createCtx()
	defer stop()
	if db.Database().Collection("sessions").FindOneAndDelete(ctx, bson.M{
		"_id":   objid,
		"owner": u.ID,
	}).Err(); err != nil {
		return err
	}
	return c.Redirect("/app")
}

func sessionPage(c *fiber.Ctx) error {
	id := c.Params("id")
	sess, err := store.Get(c)
	if err != nil {
		return err
	}
	u, ok := sess.Get("user").(models.User)
	if !ok {
		return c.Redirect("/login")
	}
	objid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	var s models.Session
	ctx, stop := createCtx()
	if err = db.Database().Collection("sessions").FindOne(ctx, bson.M{"_id": objid, "owner": u.ID}).Decode(&s); err != nil {
		stop()
		return err
	}
	stop()
	return c.Render("pages/app/session/index", fiber.Map{
		"session": s,
		"id":      objid.Hex(),
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
		return c.Redirect("/login")
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
	return c.Render("pages/app/session/question/create", fiber.Map{
		"session": s,
	}, "layouts/app")
}

func saveNewQuestion(c *fiber.Ctx) error {
	qi := models.QuestionInput{}
	if err := c.BodyParser(&qi); err != nil {
		return err
	}
	q := models.Question{
		ID:      primitive.NewObjectID(),
		Title:   qi.Title,
		Answers: []*models.Answer{},
	}
	id := c.Params("id")
	objid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	sess, _ := store.Get(c)
	u := sess.Get("user").(models.User)
	ctx, stop := createCtx()
	defer stop()
	var s models.Session
	if err := db.
		Database().
		Collection("sessions").
		FindOneAndUpdate(ctx, bson.M{
			"owner": u.ID,
			"_id":   objid,
		}, bson.M{
			"$push": bson.M{
				"questions": q,
			},
		}).
		Decode(&s); err != nil && err != mongo.ErrNoDocuments {
		return err
	}
	return c.Redirect(fmt.Sprintf("/app/session/%v/question/edit/%v", objid.Hex(), q.ID.Hex()))
}

func editQuestionPage(c *fiber.Ctx) error {
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
	sess, _ := store.Get(c)
	u := sess.Get("user").(models.User)
	var s models.Session
	if err := db.
		Database().
		Collection("sessions").
		FindOne(ctx, bson.M{
			"owner": u.ID,
			"_id":   sobjid,
		}).
		Decode(&s); err != nil {
		return err
	}
	var q *models.Question
	for _, v := range s.Questions {
		if v.ID == qobjid {
			q = v
		}
	}
	return c.Render("pages/app/session/question/edit", fiber.Map{
		"question": q,
		"sid":      s.ID,
	}, "layouts/app")
}

func editQuestion(c *fiber.Ctx) error {
	id := c.Params("id")
	objid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	qid := c.Params("qid")
	qobjid, err := primitive.ObjectIDFromHex(qid)
	if err != nil {
		return err
	}
	data, err := processForm(string(c.Body()))
	if err != nil {
		return err
	}
	sess, _ := store.Get(c)
	u := sess.Get("user").(models.User)
	cl := db.Database().Collection("sessions")
	var s models.Session
	ctx, stop := createCtx()
	if err := cl.
		FindOne(ctx, bson.M{"owner": u.ID, "_id": objid}).
		Decode(&s); err != nil {
		stop()
		return err
	}
	stop()
	var q *models.Question
	for _, v := range s.Questions {
		if v.ID.Hex() == qobjid.Hex() {
			q = v
			break
		}
	}
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
	if err = cl.
		FindOneAndReplace(ctx, bson.M{
			"_id": s.ID,
		}, s).
		Err(); err != nil {
		stop()
		return err
	}
	stop()
	return c.Redirect(fmt.Sprintf("/app/session/%v/question/edit/%v", s.ID.Hex(), q.ID.Hex()))
}

func deleteQuestion(c *fiber.Ctx) error {
	id := c.Params("id")
	objid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	qid := c.Params("qid")
	qobjid, err := primitive.ObjectIDFromHex(qid)
	if err != nil {
		return err
	}
	sess, _ := store.Get(c)
	u := sess.Get("user").(models.User)
	ctx, stop := createCtx()
	if err := db.
		Database().
		Collection("sessions").
		FindOneAndUpdate(ctx, bson.M{
			"_id":           objid,
			"owner":         u.ID,
			"questions._id": qobjid,
		}, bson.M{
			"$pull": bson.M{
				"questions": bson.M{"_id": qobjid},
			},
		}).
		Err(); err != nil {
		stop()
		return c.Redirect("/login")
	}
	stop()
	return c.Redirect(fmt.Sprintf("/app/session/%v", objid.Hex()))
}

func removeAnswerFromQuestion(c *fiber.Ctx) error {
	sid := c.Params("id")
	objid, err := primitive.ObjectIDFromHex(sid)
	if err != nil {
		return err
	}
	aid := c.Params("aid")
	aobjid, err := primitive.ObjectIDFromHex(aid)
	if err != nil {
		return err
	}
	sess, _ := store.Get(c)
	u := sess.Get("user").(models.User)
	cl := db.Database().Collection("sessions")
	ctx, stop := createCtx()
	var s models.Session
	if err := cl.
		FindOneAndUpdate(ctx, bson.M{
			"_id":                   objid,
			"owner":                 u.ID,
			"questions.answers._id": aobjid,
		}, bson.M{
			"$pull": bson.M{
				"questions.$.answers": bson.M{
					"_id": aobjid,
				}},
		}).
		Decode(&s); err != nil {
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
