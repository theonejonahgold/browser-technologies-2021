package appRouter

import (
	"bt/db"
	"bt/db/models"
	"bt/isosession"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var store *isosession.IsoStore

func NewRouter(app *fiber.App, sessStore *isosession.IsoStore) {
	store = sessStore
	router := app.Group("/app")
	router.Use(isLoggedIn)
	router.Get("/", appPage)
	router.Get("/join", join)
	router.Get("/join/waiting", joinWaitingRoom)
	router.Get("/join/countdown", joinCountdown)
	router.Get("/join/answer", joinAnswerPage)
	router.Post("/join/answer", joinAnswerQuestion)
	router.Get("/join/question-results", joinQResultsPage)
	router.Get("/host/start", startSession)
	router.Get("/host/waiting", hostWaitingRoom)
	router.Get("/host/countdown", hostCountdown)
	router.Get("/host/answer", hostAnswerPage)
	router.Get("/host/question-results", hostQResults)
	router.Get("/host/next-question", nextQuestion)
	router.Get("/host/:id", host)
	router.Get("/quiz/create/", createSessionPage)
	router.Post("/quiz/create/", saveSessionName)
	router.Post("/quiz/edit/:id", editSession)
	router.Post("/quiz/delete/:id", deleteSession)
	router.Get("/quiz/:id", sessionPage)
	router.Post("/quiz/:id/order/:oldPos/:newPos", changeQuestionOrder)
	router.Get("/quiz/:id/results", sessionResults)
	router.Get("/quiz/:id/question/create", newQuestionPage)
	router.Post("/quiz/:id/question/create", saveNewQuestion)
	router.Get("/quiz/:id/question/edit/:qid", editQuestionPage)
	router.Post("/quiz/:id/question/edit/:qid", editQuestion)
	router.Post("/quiz/:id/question/delete/:qid", deleteQuestion)
	router.Post("/quiz/:id/answer/delete/:aid", removeAnswerFromQuestion)
}

func isLoggedIn(c *fiber.Ctx) error {
	sess, uuidSess, err := store.Get(c)
	if err != nil {
		return c.Redirect("/login")
	}
	user := sess.Get("user")
	if user == nil && uuidSess != nil {
		user = uuidSess.Get("user")
	}
	if user == nil {
		return c.Redirect("/login")
	}
	return c.Next()
}

func appPage(c *fiber.Ctx) error {
	errQ := c.Query("error", "")
	var errs struct{ Session string }
	if errQ != "" {
		if strings.Contains(errQ, "session") {
			if strings.Contains(errQ, "sameacc") {
				errs.Session = "You cannot join your own sessions"
			}
			if strings.Contains(errQ, "noexist") {
				errs.Session = "Session does not exist"
			}
			if strings.Contains(errQ, "closed") {
				errs.Session = "Session is not open for new participants"
			}
			if strings.Contains(errQ, "mismatch") {
				errs.Session = "Something went wrong trying to get the quiz session, please try again."
			}
			if strings.Contains(errQ, "nostart") {
				errs.Session = "No session to start, open up a session first to start."
			}
		}
	}
	sess, uuidSess, _ := store.Get(c)
	var u models.User
	u, ok := sess.Get("user").(models.User)
	if !ok {
		u = uuidSess.Get("user").(models.User)
	}
	ctx, stop := createCtx(20)
	cur, err := db.
		Database().
		Collection("sessions").
		Find(ctx, bson.M{"$or": bson.A{bson.M{"owner": u.ID}, bson.M{"participants.user": u.ID}}})
	if err != nil {
		stop()
		return err
	}
	stop()
	ctx, stop = createCtx(20)
	var s []models.Session
	if err = cur.All(ctx, &s); err != nil {
		stop()
		return err
	}
	stop()
	var os []models.Session
	var ds []models.Session
	var fs []models.Session
	var js []models.Session
	for _, v := range s {
		if v.Owner == u.ID {
			if v.State == models.Finished {
				fs = append(fs, v)
				continue
			} else if v.State != models.Creating {
				os = append(os, v)
				continue
			}
			ds = append(ds, v)
			continue
		}
		js = append(js, v)
	}
	return c.Render("pages/app/index", fiber.Map{
		"ongoingSessions":  os,
		"finishedSessions": fs,
		"draftSessions":    ds,
		"joinedSessions":   js,
		"user":             u,
		"errors":           errs,
		"sessid":           uuidSess.ID(),
	}, "layouts/app")
}

func sessionResults(c *fiber.Ctx) error {
	sess, uuidSess, _ := store.Get(c)
	var u models.User
	defer sess.Save()
	sess.Delete("current-joining")
	sess.Delete("current-hosting")
	u, ok := sess.Get("user").(models.User)
	uuidSess.Delete("current-joining")
	uuidSess.Delete("current-hosting")
	if !ok {
		u = uuidSess.Get("user").(models.User)
	}
	sID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Redirect(fmt.Sprintf("/app?error=session_noexist&sessid=%v", uuidSess.ID()))
	}
	var s models.Session
	ctx, stop := createCtx()
	if err := db.
		Database().
		Collection("sessions").
		FindOne(ctx, bson.M{
			"_id": sID,
			"$or": bson.A{
				bson.M{"participants.user": u.ID},
				bson.M{"owner": u.ID},
			}}).
		Decode(&s); err != nil {
		stop()
		if err == mongo.ErrNoDocuments {
			return c.Redirect(fmt.Sprintf("/app?error=session_noexist&sessid=%v", uuidSess.ID()))
		}
		return err
	}
	stop()
	return c.Render("pages/app/session/results", fiber.Map{
		"session": s,
		"sessid":  uuidSess.ID(),
	}, "layouts/app")
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
