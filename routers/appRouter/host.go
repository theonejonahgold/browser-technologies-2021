package appRouter

import (
	"bt/db"
	"bt/db/models"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func hostRouter(c *fiber.Ctx) error {
	sess, _ := store.Get(c)
	sobjID := sess.Get("current-hosting").(primitive.ObjectID)
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
		return c.Redirect("/app/host/waiting")
	case models.QuestionCountdown:
		return c.Redirect("/app/host/countdown")
	case models.QuestionOpen:
		return c.Redirect("/app/host/answer")
	case models.QuestionClosed:
		return c.Redirect("/app/host/question-results")
	case models.Finished:
		return c.Redirect(fmt.Sprintf("/app/quiz/%v/results", s.ID.Hex()))
	default:
		return c.Redirect("/app?error=session_noexist")
	}
}

func host(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return err
	}
	sess, _ := store.Get(c)
	defer sess.Save()
	if sID, ok := sess.Get("current-hosting").(primitive.ObjectID); ok && sID == id {
		return hostRouter(c)
	}
	u, _ := sess.Get("user").(models.User)
	var s models.Session
	ctx, stop := createCtx()

	if err := db.
		Database().
		Collection("sessions").
		FindOne(
			ctx,
			bson.M{"_id": id, "owner": u.ID}).
		Decode(&s); err != nil {
		stop()
		return err
	}
	stop()
	sess.Set("current-hosting", s.ID)
	if s.State != models.Creating && s.State != models.Waiting {
		return c.Redirect(fmt.Sprintf("/app/host/%v", s.ID.Hex()))
	}
	ctx, stop = createCtx()
	if err := db.
		Database().
		Collection("sessions").
		FindOneAndUpdate(
			ctx,
			bson.M{"_id": id, "owner": u.ID},
			bson.M{"$set": bson.M{"state": models.Waiting}},
			options.FindOneAndUpdate().SetReturnDocument(options.After)).
		Decode(&s); err != nil {
		stop()
		return err
	}
	stop()
	return c.Redirect(fmt.Sprintf("/app/host/%v", s.ID.Hex()))
}

func hostWaitingRoom(c *fiber.Ctx) error {
	sess, _ := store.Get(c)
	sID := sess.Get("current-hosting").(primitive.ObjectID)
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
	return c.Render("pages/app/host/waiting", fiber.Map{
		"session": s,
	}, "layouts/host")
}

func startSession(c *fiber.Ctx) error {
	sess, _ := store.Get(c)
	u, _ := sess.Get("user").(models.User)
	sID, ok := sess.Get("current-hosting").(primitive.ObjectID)
	if !ok {
		return c.Redirect("/app?error=session_nostart")
	}
	var s models.Session
	ctx, stop := createCtx()
	defer stop()
	if err := db.
		Database().
		Collection("sessions").
		FindOne(
			ctx,
			bson.M{"_id": sID, "owner": u.ID}).
		Decode(&s); err != nil {
		stop()
		return err
	}
	stop()
	ctx, stop = createCtx()
	if err := db.
		Database().
		Collection("sessions").
		FindOneAndUpdate(ctx,
			bson.M{"_id": s.ID},
			bson.M{"$set": bson.M{"state": models.QuestionCountdown, "current": s.Questions[0].ID}},
			options.FindOneAndUpdate().SetReturnDocument(options.After)).
		Decode(&s); err != nil {
		stop()
		return err
	}
	stop()
	go changeStateAfterCountdown(s)
	return c.Redirect(fmt.Sprintf("/app/host/%v", s.ID.Hex()))
}

func hostCountdown(c *fiber.Ctx) error {
	sess, _ := store.Get(c)
	sobjID, ok := sess.Get("current-hosting").(primitive.ObjectID)
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
	var q models.Question
	for _, v := range s.Questions {
		if s.CurrentQuestion == v.ID {
			q = *v
			break
		}
	}
	return c.Render("pages/app/host/countdown", fiber.Map{
		"session":  s,
		"question": q,
	}, "layouts/host")
}

func hostAnswerPage(c *fiber.Ctx) error {
	sess, _ := store.Get(c)
	sobjID := sess.Get("current-hosting").(primitive.ObjectID)
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
	var currQ *models.Question
	var amtAns int
	for _, q := range s.Questions {
		if q.ID == s.CurrentQuestion {
			currQ = q
			for _, a := range q.Answers {
				amtAns += len(a.Participants)
			}
			break
		}
	}
	return c.Render("pages/app/host/answer", fiber.Map{
		"session":  s,
		"question": currQ,
		"amtAns":   amtAns,
	}, "layouts/host")
}

func hostQResults(c *fiber.Ctx) error {
	sess, _ := store.Get(c)
	sobjID := sess.Get("current-hosting").(primitive.ObjectID)
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
	var currQ models.Question
	for _, q := range s.Questions {
		if q.ID == s.CurrentQuestion {
			currQ = *q
		}
	}
	return c.Render("pages/app/host/results", fiber.Map{
		"session":  s,
		"question": currQ,
	}, "layouts/host")
}

func nextQuestion(c *fiber.Ctx) error {
	sess, _ := store.Get(c)
	sID := sess.Get("current-hosting").(primitive.ObjectID)
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
	state := s.State
	for k, v := range s.Questions {
		if v.ID == s.CurrentQuestion {
			if len(s.Questions) == k+1 {
				state = models.Finished
				break
			}
			s.CurrentQuestion = s.Questions[k+1].ID
			state = models.QuestionCountdown
			break
		}
	}
	ctx, stop = createCtx()
	if err := db.
		Database().
		Collection("sessions").
		FindOneAndUpdate(ctx,
			bson.M{"_id": s.ID},
			bson.M{"$set": bson.M{"state": state, "current": s.CurrentQuestion}},
			options.FindOneAndUpdate().SetReturnDocument(options.After)).
		Decode(&s); err != nil {
		stop()
		return err
	}
	stop()
	if state != models.Finished {
		go changeStateAfterCountdown(s)
	}
	return c.Redirect(fmt.Sprintf("/app/host/%v", s.ID.Hex()))
}

func changeStateAfterCountdown(s models.Session) {
	<-time.After(5 * time.Second)
	ctx, stop := createCtx()
	if err := db.
		Database().
		Collection("sessions").
		FindOneAndUpdate(ctx,
			bson.M{"_id": s.ID},
			bson.M{"$set": bson.M{"state": models.QuestionOpen}},
			options.FindOneAndUpdate().SetReturnDocument(options.After)).
		Err(); err != nil {
		fmt.Printf("error while updating countdown state: %v", err)
		stop()
	}
	stop()
}
