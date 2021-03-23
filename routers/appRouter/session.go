package appRouter

import (
	"bt/db"
	"bt/db/models"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func createSessionPage(c *fiber.Ctx) error {
	_, uuidSess, _ := store.Get(c)
	return c.Render("pages/app/session/create", fiber.Map{
		"sessid": uuidSess.ID(),
	}, "layouts/app")
}

func saveSessionName(c *fiber.Ctx) error {
	var si models.SessionInput
	if err := c.BodyParser(&si); err != nil {
		return err
	}
	sess, uuidSess, err := store.Get(c)
	if err != nil {
		return err
	}
	defer sess.Save()
	u, ok := sess.Get("user").(models.User)
	if !ok {
		u, ok = uuidSess.Get("user").(models.User)
		if !ok {
			return c.Redirect(fmt.Sprintf("/login?sessid=%v", uuidSess.ID()))
		}
	}
	cl := db.Database().Collection("sessions")
	id := primitive.NewObjectID()
	ctx, stop := createCtx()
	defer stop()
	if _, err = cl.InsertOne(ctx, models.Session{
		ID:              id,
		Name:            si.Name,
		Owner:           u.ID,
		QuestionTimer:   0,
		Participants:    []*models.Participant{},
		Questions:       []*models.Question{},
		Code:            fmt.Sprintf("%v-%v", u.Username, id.Hex()[len(id.Hex())-8:]),
		State:           models.Creating,
		CurrentQuestion: [12]byte{},
	}); err != nil {
		return err
	}
	return c.Redirect(fmt.Sprintf("/app/quiz/%v?sessid=%v", id.Hex(), uuidSess.ID()))
}

func deleteSession(c *fiber.Ctx) error {
	id := c.Params("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	sess, uuidSess, _ := store.Get(c)
	u, ok := sess.Get("user").(models.User)
	if !ok {
		u = uuidSess.Get("user").(models.User)
	}
	ctx, stop := createCtx()
	defer stop()
	if db.Database().Collection("sessions").FindOneAndDelete(ctx, bson.M{
		"_id":   objID,
		"owner": u.ID,
	}).Err(); err != nil {
		return err
	}
	return c.Redirect(fmt.Sprintf("/app?sessid=%v", uuidSess.ID()))
}

func sessionPage(c *fiber.Ctx) error {
	id := c.Params("id")
	sess, uuidSess, _ := store.Get(c)
	u, ok := sess.Get("user").(models.User)
	if !ok {
		u = uuidSess.Get("user").(models.User)
	}
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	var s models.Session
	ctx, stop := createCtx()
	if err = db.Database().
		Collection("sessions").
		FindOne(ctx, bson.M{"_id": objID, "owner": u.ID}).
		Decode(&s); err != nil {
		stop()
		return err
	}
	stop()
	if s.State == models.Finished {
		return c.Redirect(fmt.Sprintf("/app/quiz/%v/results?sessid=%v", s.ID.Hex(), uuidSess.ID()))
	}
	if s.State != models.Creating {
		return c.Redirect(fmt.Sprintf("/app/host/%v?sessid=%v", s.ID.Hex(), uuidSess.ID()))
	}
	return c.Render("pages/app/session/index", fiber.Map{
		"session": s,
		"id":      objID.Hex(),
		"sessid":  uuidSess.ID(),
	}, "layouts/app")
}

func newQuestionPage(c *fiber.Ctx) error {
	id := c.Params("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	sess, uuidSess, err := store.Get(c)
	if err != nil {
		return err
	}
	u, ok := sess.Get("user").(models.User)
	if !ok {
		u, ok = uuidSess.Get("user").(models.User)
		if !ok {
			return c.Redirect("/login")
		}
	}
	ctx, stop := createCtx()
	defer stop()
	var s models.Session
	if err = db.
		Database().
		Collection("sessions").
		FindOne(ctx, bson.M{"owner": u.ID, "_id": objID}).
		Decode(&s); err == mongo.ErrNoDocuments {
		return c.Redirect(fmt.Sprintf("/app/sesssion/create?sessid=%v", uuidSess.ID()))
	} else if err != nil {
		return err
	}
	return c.Render("pages/app/session/question/create", fiber.Map{
		"session": s,
		"sessid":  uuidSess.ID(),
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
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	sess, uuidSess, _ := store.Get(c)
	u, ok := sess.Get("user").(models.User)
	if !ok {
		u = uuidSess.Get("user").(models.User)
	}
	ctx, stop := createCtx()
	defer stop()
	var s models.Session
	if err := db.
		Database().
		Collection("sessions").
		FindOneAndUpdate(ctx, bson.M{
			"owner": u.ID,
			"_id":   objID,
		}, bson.M{
			"$push": bson.M{
				"questions": q,
			},
		}).
		Decode(&s); err != nil && err != mongo.ErrNoDocuments {
		return err
	}
	return c.Redirect(fmt.Sprintf("/app/quiz/%v/question/edit/%v?sessid=%v", objID.Hex(), q.ID.Hex(), uuidSess.ID()))
}

func editQuestionPage(c *fiber.Ctx) error {
	sid := c.Params("id")
	qid := c.Params("qid")
	sObjID, err := primitive.ObjectIDFromHex(sid)
	if err != nil {
		return err
	}
	qObjID, err := primitive.ObjectIDFromHex(qid)
	if err != nil {
		return err
	}
	ctx, stop := createCtx()
	defer stop()
	sess, uuidSess, _ := store.Get(c)
	u, ok := sess.Get("user").(models.User)
	if !ok {
		u = uuidSess.Get("user").(models.User)
	}
	var s models.Session
	if err := db.
		Database().
		Collection("sessions").
		FindOne(ctx, bson.M{
			"owner": u.ID,
			"_id":   sObjID,
		}).
		Decode(&s); err != nil {
		return err
	}
	var q models.Question
	for _, v := range s.Questions {
		if v.ID == qObjID {
			q = *v
		}
	}
	return c.Render("pages/app/session/question/edit", fiber.Map{
		"question": q,
		"sid":      s.ID,
		"sessid":   uuidSess.ID(),
	}, "layouts/app")
}

func editQuestion(c *fiber.Ctx) error {
	id := c.Params("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	qid := c.Params("qid")
	qObjID, err := primitive.ObjectIDFromHex(qid)
	if err != nil {
		return err
	}
	data, err := processForm(string(c.Body()))
	if err != nil {
		return err
	}
	sess, uuidSess, _ := store.Get(c)
	u, ok := sess.Get("user").(models.User)
	if !ok {
		u = uuidSess.Get("user").(models.User)
	}
	cl := db.Database().Collection("sessions")
	var s models.Session
	ctx, stop := createCtx()
	if err := cl.
		FindOne(ctx, bson.M{"owner": u.ID, "_id": objID}).
		Decode(&s); err != nil {
		stop()
		return err
	}
	stop()
	var q *models.Question
	for _, v := range s.Questions {
		if v.ID.Hex() == qObjID.Hex() {
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
		FindOneAndReplace(ctx, bson.M{"_id": s.ID}, s).
		Err(); err != nil {
		stop()
		return err
	}
	stop()
	return c.Redirect(fmt.Sprintf("/app/quiz/%v/question/edit/%v?sessid=%v", s.ID.Hex(), q.ID.Hex(), uuidSess.ID()))
}

func deleteQuestion(c *fiber.Ctx) error {
	id := c.Params("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	qid := c.Params("qid")
	qObjID, err := primitive.ObjectIDFromHex(qid)
	if err != nil {
		return err
	}
	sess, uuidSess, _ := store.Get(c)
	u, ok := sess.Get("user").(models.User)
	if !ok {
		u = uuidSess.Get("user").(models.User)
	}
	ctx, stop := createCtx()
	if err := db.
		Database().
		Collection("sessions").
		FindOneAndUpdate(ctx, bson.M{
			"_id":           objID,
			"owner":         u.ID,
			"questions._id": qObjID,
		}, bson.M{
			"$pull": bson.M{
				"questions": bson.M{"_id": qObjID},
			},
		}).
		Err(); err != nil {
		stop()
		return c.Redirect("/login")
	}
	stop()
	return c.Redirect(fmt.Sprintf("/app/quiz/%v?sessid=%v", objID.Hex(), uuidSess.ID()))
}

func removeAnswerFromQuestion(c *fiber.Ctx) error {
	sid := c.Params("id")
	objID, err := primitive.ObjectIDFromHex(sid)
	if err != nil {
		return err
	}
	aid := c.Params("aid")
	aObjID, err := primitive.ObjectIDFromHex(aid)
	if err != nil {
		return err
	}
	sess, uuidSess, _ := store.Get(c)
	u, ok := sess.Get("user").(models.User)
	if !ok {
		u = uuidSess.Get("user").(models.User)
	}
	cl := db.Database().Collection("sessions")
	ctx, stop := createCtx()
	var s models.Session
	if err := cl.
		FindOneAndUpdate(ctx, bson.M{
			"_id":                   objID,
			"owner":                 u.ID,
			"questions.answers._id": aObjID,
		}, bson.M{
			"$pull": bson.M{
				"questions.$.answers": bson.M{
					"_id": aObjID,
				}},
		}).
		Decode(&s); err != nil {
		stop()
		return err
	}
	stop()
	return c.Redirect(fmt.Sprintf("/app/quiz/%v?sessid=%v", objID.Hex(), uuidSess.ID()))
}

func processForm(body string) (map[string][]string, error) {
	vals, err := url.ParseQuery(body)
	if err != nil {
		return nil, err
	}
	return vals, err
}
