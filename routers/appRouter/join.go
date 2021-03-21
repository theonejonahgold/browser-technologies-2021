package appRouter

import (
	"bt/db"
	"bt/db/models"
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func joinRouter(c *fiber.Ctx) error {
	sess, _ := store.Get(c)
	sobjID := sess.Get("current-joining").(primitive.ObjectID)
	var s models.Session
	ctx, stop := createCtx()
	if err := db.
		Database().
		Collection("sessions").
		FindOne(ctx, bson.M{"_id": sobjID}).
		Decode(&s); err != nil {
		stop()
		return err
	}
	stop()
	switch s.State {
	case models.Waiting:
		return c.Redirect("/app/join/waiting")
	case models.QuestionCountdown:
		return c.Redirect("/app/join/countdown")
	case models.QuestionOpen:
		return c.Redirect("/app/join/answer")
	case models.QuestionClosed:
		return c.Redirect("/app/join/question-results")
	case models.Finished:
		return c.Redirect(fmt.Sprintf("/app/quiz/%v/results", s.ID.Hex()))
	default:
		return c.Redirect("/app?error=session_noexist")
	}
}

func join(c *fiber.Ctx) error {
	sess, _ := store.Get(c)
	defer sess.Save()
	if sess.Get("current-joining") != nil {
		return joinRouter(c)
	}
	var joinForm struct {
		Session string `json:"session"`
	}
	if err := c.QueryParser(&joinForm); err != nil {
		return err
	}
	u := sess.Get("user").(models.User)
	var s models.Session
	cl := db.Database().Collection("sessions")
	if amt, err := cl.CountDocuments(context.Background(), bson.M{"code": joinForm.Session}); err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Redirect("/app?error=session_sameacc")
		}
		return err
	} else if amt == 0 {
		return c.Redirect("/app?error=session_noexist")
	}
	ctx, stop := createCtx()
	if err := cl.
		FindOne(ctx, bson.M{
			"code":  joinForm.Session,
			"owner": bson.M{"$ne": u.ID},
		}).
		Decode(&s); err != nil {
		stop()
		if err == mongo.ErrNoDocuments {
			return c.Redirect("/app?error=session_sameacc")
		}
		return err
	}
	stop()
	if s.State == models.Creating {
		return c.Redirect("/app?error=session_noexist")
	}
	joined := false
	for _, v := range s.Participants {
		if v.ID == u.ID {
			joined = true
		}
	}
	if s.State != models.Waiting && !joined {
		return c.Redirect("/app?error=session_closed")
	}
	ctx, stop = createCtx()
	if err := cl.
		FindOneAndUpdate(ctx, bson.M{
			"code":              joinForm.Session,
			"participants.user": bson.M{"$ne": u.ID},
		}, bson.M{
			"$push": bson.M{
				"participants": models.Participant{
					ID:      primitive.NewObjectID(),
					User:    u.ID,
					Strikes: 0,
				},
			},
		}, options.FindOneAndUpdate().SetReturnDocument(options.After)).
		Decode(&s); err != nil && err != mongo.ErrNoDocuments {
		stop()
		return err
	}
	stop()
	sess.Set("current-joining", s.ID)
	return c.Redirect(fmt.Sprintf("/app/join?session=%v", s.Code))
}

func joinWaitingRoom(c *fiber.Ctx) error {
	sess, _ := store.Get(c)
	sobjID, ok := sess.Get("current-joining").(primitive.ObjectID)
	if !ok {
		return c.Redirect("/app")
	}
	var s models.Session
	ctx, stop := createCtx()
	if err := db.
		Database().
		Collection("sessions").
		FindOne(ctx, bson.M{"_id": sobjID}).
		Decode(&s); err != nil {
		return err
	}
	var u models.User
	if err := db.
		Database().
		Collection("users").
		FindOne(ctx, bson.M{
			"_id": s.Owner,
		}).
		Decode(&u); err != nil {
		stop()
		return err
	}
	stop()
	return c.Render("pages/app/join/waiting", fiber.Map{
		"session": s,
		"user":    u,
	}, "layouts/join")
}

func joinCountdown(c *fiber.Ctx) error {
	sess, _ := store.Get(c)
	sobjID, ok := sess.Get("current-joining").(primitive.ObjectID)
	if !ok {
		return c.Redirect("/app")
	}
	var s models.Session
	ctx, stop := createCtx()
	defer stop()
	if err := db.
		Database().
		Collection("sessions").
		FindOne(ctx, bson.M{"_id": sobjID}).
		Decode(&s); err != nil {
		return err
	}
	var q *models.Question
	for _, v := range s.Questions {
		if s.CurrentQuestion == v.ID {
			q = v
			break
		}
	}
	return c.Render("pages/app/join/countdown", fiber.Map{
		"session":  s,
		"question": q,
	}, "layouts/join")
}

func joinAnswerPage(c *fiber.Ctx) error {
	sess, _ := store.Get(c)
	sobjID, ok := sess.Get("current-joining").(primitive.ObjectID)
	if !ok {
		return c.Redirect("/app")
	}
	var s models.Session
	ctx, stop := createCtx()
	defer stop()
	if err := db.
		Database().
		Collection("sessions").
		FindOne(ctx, bson.M{"_id": sobjID}).
		Decode(&s); err != nil {
		return err
	}
	u, _ := sess.Get("user").(models.User)
	var q *models.Question
	var answered bool
	for _, v := range s.Questions {
		if v.ID == s.CurrentQuestion {
			q = v
		}
	}
aLoop:
	for _, v := range q.Answers {
		for _, a := range v.Participants {
			if a == u.ID {
				answered = true
				break aLoop
			}
		}
	}
	return c.Render("pages/app/join/answer", fiber.Map{
		"question": q,
		"session":  s,
		"user":     u,
		"answered": answered,
	}, "layouts/join")
}

func joinAnswerQuestion(c *fiber.Ctx) error {
	var body struct {
		UserID string `json:"userid"`
		Answer string `json:"answer"`
	}
	if err := c.BodyParser(&body); err != nil {
		return err
	}
	uID, _ := primitive.ObjectIDFromHex(body.UserID)
	aID, _ := primitive.ObjectIDFromHex(body.Answer)
	sess, _ := store.Get(c)
	sID := sess.Get("current-joining").(primitive.ObjectID)
	ctx, stop := createCtx()
	var s models.Session
	defer stop()
	if err := db.
		Database().
		Collection("sessions").
		FindOne(ctx, bson.M{
			"_id":                   sID,
			"participants.user":     uID,
			"questions.answers._id": aID,
		}).
		Decode(&s); err != nil {
		stop()
		return err
	}
	stop()
	var idx int
	for k, q := range s.Questions {
		if q.ID == s.CurrentQuestion {
			idx = k
			for _, a := range q.Answers {
				if a.ID == aID {
					a.Participants = append(a.Participants, uID)
					break
				}
			}
			break
		}
	}
	var totalAns int
	for _, a := range s.Questions[idx].Answers {
		totalAns += len(a.Participants)
	}
	if totalAns == len(s.Participants) {
		s.State = models.QuestionClosed
	}
	ctx, stop = createCtx()
	if err := db.
		Database().
		Collection("sessions").
		FindOneAndUpdate(ctx, bson.M{
			"_id":               s.ID,
			"participants.user": uID,
		}, bson.M{
			"$set": bson.M{
				"questions": s.Questions,
				"state":     s.State,
			},
		}).
		Err(); err != nil {
		stop()
		return err
	}
	stop()
	return c.Redirect(fmt.Sprintf("/app/join?session=%v", s.Code))
}

func joinQResultsPage(c *fiber.Ctx) error {
	sess, _ := store.Get(c)
	sID := sess.Get("current-joining").(primitive.ObjectID)
	var s models.Session
	ctx, stop := createCtx()
	if err := db.
		Database().
		Collection("sessions").
		FindOne(ctx, bson.M{"_id": sID}).
		Decode(&s); err != nil {
		stop()
		return err
	}
	stop()
	var currQ models.Question
	for _, q := range s.Questions {
		if q.ID == s.CurrentQuestion {
			currQ = *q
		}
	}
	return c.Render("pages/app/join/results", fiber.Map{
		"session":  s,
		"question": currQ,
	}, "layouts/join")
}
