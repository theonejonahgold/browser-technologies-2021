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
	sess, uuidSess, _ := store.Get(c)
	sObjID, ok := sess.Get("current-hosting").(primitive.ObjectID)
	if !ok {
		sObjID = uuidSess.Get("current-hosting").(primitive.ObjectID)
	}
	var s models.Session
	ctx, stop := createCtx()
	if err := db.
		Database().
		Collection("sessions").
		FindOne(ctx, bson.M{"_id": sObjID}).
		Decode(&s); err != nil {
		stop()
		return err
	}
	stop()
	switch s.State {
	case models.Waiting:
		return hostWaitingRoom(c)
	case models.QuestionCountdown:
		return hostCountdown(c)
	case models.QuestionOpen:
		return hostAnswerPage(c)
	case models.QuestionClosed:
		return hostQResults(c)
	case models.Finished:
		return c.Redirect(fmt.Sprintf("/app/quiz/%v/results?sessid=%v", s.ID.Hex(), uuidSess.ID()))
	default:
		return c.Redirect(fmt.Sprintf("/app?error=session_noexist?sessid=%v", uuidSess.ID()))
	}
}

func host(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Query("session"))
	if err != nil {
		return err
	}
	sess, uuidSess, _ := store.Get(c)
	defer sess.Save()
	if sID, ok := sess.Get("current-hosting").(primitive.ObjectID); ok && sID == id {
		return hostRouter(c)
	}
	if sID, ok := uuidSess.Get("current-hosting").(primitive.ObjectID); ok && sID == id {
		return hostRouter(c)
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
		FindOne(
			ctx,
			bson.M{"_id": id, "owner": u.ID}).
		Decode(&s); err != nil {
		stop()
		return err
	}
	stop()
	sess.Set("current-hosting", s.ID)
	uuidSess.Set("current-hosting", s.ID)
	if s.State != models.Creating && s.State != models.Waiting {
		return c.Redirect(fmt.Sprintf("/app/host?session=%v&sessid=%v", s.ID.Hex(), uuidSess.ID()))
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
	return c.Redirect(fmt.Sprintf("/app/host?session=%v&sessid=%v", s.ID.Hex(), uuidSess.ID()))
}

func hostWaitingRoom(c *fiber.Ctx) error {
	sess, uuidSess, _ := store.Get(c)
	sID, ok := sess.Get("current-hosting").(primitive.ObjectID)
	if !ok {
		sID = uuidSess.Get("current-hosting").(primitive.ObjectID)
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
	return c.Render("pages/app/host/waiting", fiber.Map{
		"session": s,
		"sessid":  uuidSess.ID(),
	}, "layouts/host")
}

func startSession(c *fiber.Ctx) error {
	sess, uuidSess, _ := store.Get(c)
	u, ok := sess.Get("user").(models.User)
	if !ok {
		u = uuidSess.Get("user").(models.User)
	}
	sID, ok := sess.Get("current-hosting").(primitive.ObjectID)
	if !ok {
		sID, ok = uuidSess.Get("current-hosting").(primitive.ObjectID)
		if !ok {
			return c.Redirect(fmt.Sprintf("/app?error=session_nostart&sessid=%v", uuidSess.ID()))
		}
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
	return c.Redirect(fmt.Sprintf("/app/host?session=%v&sessid=%v", s.ID.Hex(), uuidSess.ID()))
}

func hostCountdown(c *fiber.Ctx) error {
	sess, uuidSess, _ := store.Get(c)
	sObjID, ok := sess.Get("current-hosting").(primitive.ObjectID)
	if !ok {
		sObjID, ok = uuidSess.Get("current-hosting").(primitive.ObjectID)
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
	var last bool
	var q models.Question
	for i, v := range s.Questions {
		if s.CurrentQuestion == v.ID {
			last = i == len(s.Questions)-1
			q = *v
			break
		}
	}
	return c.Render("pages/app/host/countdown", fiber.Map{
		"session":  s,
		"question": q,
		"sessid":   uuidSess.ID(),
		"last":     last,
	}, "layouts/host")
}

func hostAnswerPage(c *fiber.Ctx) error {
	sess, uuidSess, _ := store.Get(c)
	sObjID, ok := sess.Get("current-hosting").(primitive.ObjectID)
	if !ok {
		sObjID = uuidSess.Get("current-hosting").(primitive.ObjectID)
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
		"sessid":   uuidSess.ID(),
	}, "layouts/host")
}

func hostQResults(c *fiber.Ctx) error {
	sess, uuidSess, _ := store.Get(c)
	sObjID, ok := sess.Get("current-hosting").(primitive.ObjectID)
	if !ok {
		sObjID = uuidSess.Get("current-hosting").(primitive.ObjectID)
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
	var last bool
	var currQ models.Question
	for i, q := range s.Questions {
		if q.ID == s.CurrentQuestion {
			last = i == len(s.Questions)-1
			currQ = *q
		}
	}
	return c.Render("pages/app/host/results", fiber.Map{
		"session":  s,
		"question": currQ,
		"sessid":   uuidSess.ID(),
		"last":     last,
	}, "layouts/host")
}

func nextQuestion(c *fiber.Ctx) error {
	sess, uuidSess, _ := store.Get(c)
	sID, ok := sess.Get("current-hosting").(primitive.ObjectID)
	if !ok {
		sID = uuidSess.Get("current-hosting").(primitive.ObjectID)
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
	return c.Redirect(fmt.Sprintf("/app/host?session=%v&sessid=%v", s.ID.Hex(), uuidSess.ID()))
}

func changeStateAfterCountdown(s models.Session) {
	time.Sleep(3 * time.Second)
	ctx, stop := createCtx()
	if err := db.
		Database().
		Collection("sessions").
		FindOneAndUpdate(ctx,
			bson.M{"_id": s.ID},
			bson.M{"$set": bson.M{"state": models.QuestionOpen}},
			options.FindOneAndUpdate().SetReturnDocument(options.After)).
		Err(); err != nil {
		fmt.Printf("error while updating countdown state: %v\n", err)
		stop()
	}
	stop()
	go changeStateAfterQTimeElapsed(s)
}

func changeStateAfterQTimeElapsed(s models.Session) {
	if s.QuestionTimer == 0 {
		return
	}
	time.Sleep(time.Duration(s.QuestionTimer) * time.Second)
	ctx, stop := createCtx()
	if err := db.
		Database().
		Collection("sessions").
		FindOneAndUpdate(
			ctx,
			bson.M{"_id": s.ID, "state": models.QuestionOpen, "current": s.CurrentQuestion},
			bson.M{"$set": bson.M{"state": models.QuestionClosed}}).
		Err(); err != nil {
		fmt.Printf("error while updating question state: %v\n", err)
		stop()
	}
	stop()
}
