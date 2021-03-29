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
	sess, uuidSess, _ := store.Get(c)
	sObjID, ok := sess.Get("current-joining").(primitive.ObjectID)
	if !ok {
		sObjID = uuidSess.Get("current-joining").(primitive.ObjectID)
	}
	u, ok := sess.Get("user").(models.User)
	if !ok {
		u = uuidSess.Get("user").(models.User)
	}
	var s models.Session
	ctx, stop := createCtx()
	if err := db.
		Database().
		Collection("sessions").
		FindOne(ctx, bson.M{"_id": sObjID, "participants.user": u.ID}).
		Decode(&s); err != nil {
		stop()
		return err
	}
	stop()
	switch s.State {
	case models.Waiting:
		return joinWaitingRoom(c)
	case models.QuestionCountdown:
		return joinCountdown(c)
	case models.QuestionOpen:
		return joinAnswerPage(c)
	case models.QuestionClosed:
		return joinQResultsPage(c)
	case models.Finished:
		return c.Redirect(fmt.Sprintf("/app/quiz/%v/results?sessid=%v", s.ID.Hex(), uuidSess.ID()))
	default:
		return c.Redirect(fmt.Sprintf("/app?error=session_noexist&sessid=%v", uuidSess.ID()))
	}
}

func join(c *fiber.Ctx) error {
	sess, uuidSess, _ := store.Get(c)
	defer sess.Save()
	if sess.Get("current-joining") != nil {
		return joinRouter(c)
	}
	if uuidSess.Get("current-joining") != nil {
		return joinRouter(c)
	}
	var joinForm struct {
		Session string `form:"session" json:"session"`
	}
	if err := c.QueryParser(&joinForm); err != nil {
		return err
	}
	u, ok := sess.Get("user").(models.User)
	if !ok {
		u = uuidSess.Get("user").(models.User)
	}
	var s models.Session
	cl := db.Database().Collection("sessions")
	if amt, err := cl.CountDocuments(context.Background(), bson.M{"code": joinForm.Session}); err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Redirect(fmt.Sprintf("/app?error=session_sameacc&sessid=%v", uuidSess.ID()))
		}
		return err
	} else if amt == 0 {
		return c.Redirect(fmt.Sprintf("/app?error=session_noexist&sessid=%v", uuidSess.ID()))
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
			return c.Redirect(fmt.Sprintf("/app?error=session_sameacc&sessid=%v", uuidSess.ID()))
		}
		return err
	}
	stop()
	if s.State == models.Creating {
		return c.Redirect(fmt.Sprintf("/app?error=session_noexist&sessid=%v", uuidSess.ID()))
	}
	joined := false
	for _, v := range s.Participants {
		if v.User == u.ID {
			joined = true
		}
	}
	if s.State != models.Waiting && !joined {
		return c.Redirect(fmt.Sprintf("/app?error=session_closed&sessid=%v", uuidSess.ID()))
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
	uuidSess.Set("current-joining", s.ID)
	return c.Redirect(fmt.Sprintf("/app/join?session=%v&sessid=%v", s.Code, uuidSess.ID()))
}

func joinWaitingRoom(c *fiber.Ctx) error {
	sess, uuidSess, _ := store.Get(c)
	sObjID, ok := sess.Get("current-joining").(primitive.ObjectID)
	if !ok {
		sObjID, ok = uuidSess.Get("current-joining").(primitive.ObjectID)
		if !ok {
			return c.Redirect(fmt.Sprintf("/app?sessid=%v", uuidSess.ID()))
		}
	}
	var s models.Session
	ctx, stop := createCtx()
	if err := db.
		Database().
		Collection("sessions").
		FindOne(ctx, bson.M{"_id": sObjID}).
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
		"sessid":  uuidSess.ID(),
	}, "layouts/join")
}

func joinCountdown(c *fiber.Ctx) error {
	sess, uuidSess, _ := store.Get(c)
	sObjID, ok := sess.Get("current-joining").(primitive.ObjectID)
	if !ok {
		sObjID, ok = uuidSess.Get("current-joining").(primitive.ObjectID)
		if !ok {
			return c.Redirect(fmt.Sprintf("/app?sessid=%v", uuidSess.ID()))
		}
	}
	var s models.Session
	ctx, stop := createCtx()
	defer stop()
	if err := db.
		Database().
		Collection("sessions").
		FindOne(ctx, bson.M{"_id": sObjID}).
		Decode(&s); err != nil {
		return err
	}
	var q *models.Question
	var last bool
	for i, v := range s.Questions {
		if s.CurrentQuestion == v.ID {
			last = i == len(s.Questions)-1
			q = v
			break
		}
	}
	return c.Render("pages/app/join/countdown", fiber.Map{
		"session":  s,
		"question": q,
		"sessid":   uuidSess.ID(),
		"last":     last,
	}, "layouts/join")
}

func joinAnswerPage(c *fiber.Ctx) error {
	sess, uuidSess, _ := store.Get(c)
	sObjID, ok := sess.Get("current-joining").(primitive.ObjectID)
	if !ok {
		sObjID, ok = uuidSess.Get("current-joining").(primitive.ObjectID)
		if !ok {
			return c.Redirect(fmt.Sprintf("/app?sessid=%v", uuidSess.ID()))
		}
	}
	var s models.Session
	ctx, stop := createCtx()
	defer stop()
	if err := db.
		Database().
		Collection("sessions").
		FindOne(ctx, bson.M{"_id": sObjID}).
		Decode(&s); err != nil {
		return err
	}
	if s.State != models.QuestionOpen {
		return c.Redirect(fmt.Sprintf("/app/join?session=%v&sessid=%v", s.ID.Hex(), uuidSess.ID()))
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
		"sessid":   uuidSess.ID(),
	}, "layouts/join")
}

func joinAnswerQuestion(c *fiber.Ctx) error {
	var body struct {
		Answer string `json:"answer"`
	}
	if err := c.BodyParser(&body); err != nil {
		return err
	}
	aID, _ := primitive.ObjectIDFromHex(body.Answer)
	sess, uuidSess, _ := store.Get(c)
	sID, ok := sess.Get("current-joining").(primitive.ObjectID)
	if !ok {
		sID = uuidSess.Get("current-joining").(primitive.ObjectID)
	}
	u, ok := sess.Get("user").(models.User)
	if !ok {
		u = uuidSess.Get("user").(models.User)
	}
	ctx, stop := createCtx()
	var s models.Session
	defer stop()
	if err := db.
		Database().
		Collection("sessions").
		FindOne(ctx, bson.M{
			"_id":                   sID,
			"participants.user":     u.ID,
			"questions.answers._id": aID,
		}).
		Decode(&s); err != nil {
		stop()
		return err
	}
	stop()
	if s.State != models.QuestionOpen {
		return c.Redirect(fmt.Sprintf("/app/join?session=%v&sessid=%v", s.ID.Hex(), uuidSess.ID()))
	}
	var idx int
	for k, q := range s.Questions {
		if q.ID == s.CurrentQuestion {
			idx = k
			for _, a := range q.Answers {
				if a.ID == aID {
					a.Participants = append(a.Participants, u.ID)
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
			"participants.user": u.ID,
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
	return c.Redirect(fmt.Sprintf("/app/join?session=%v&sessid=%v", s.Code, uuidSess.ID()))
}

func joinQResultsPage(c *fiber.Ctx) error {
	sess, uuidSess, _ := store.Get(c)
	sID, ok := sess.Get("current-joining").(primitive.ObjectID)
	if !ok {
		sID = uuidSess.Get("current-joining").(primitive.ObjectID)
	}
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
	var last bool
	var currQ models.Question
	for i, q := range s.Questions {
		if q.ID == s.CurrentQuestion {
			last = i == len(s.Questions)-1
			currQ = *q
		}
	}
	return c.Render("pages/app/join/results", fiber.Map{
		"session":  s,
		"question": currQ,
		"sessid":   uuidSess.ID(),
		"last":     last,
	}, "layouts/join")
}
