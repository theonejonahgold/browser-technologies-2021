package appRouter

import (
	"bt/db"
	"bt/db/models"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	joinProtocol wsProtocol = "join"
	hostProtocol wsProtocol = "host"
)

type wsProtocol string

const (
	joined          wsMessageTypes = "joined"
	participant     wsMessageTypes = "participant"
	start           wsMessageTypes = "start"
	countdown       wsMessageTypes = "countdown"
	open            wsMessageTypes = "open"
	answer          wsMessageTypes = "answer"
	answerAmt       wsMessageTypes = "answered"
	confirmedAnswer wsMessageTypes = "confirmed"
	results         wsMessageTypes = "results"
	next            wsMessageTypes = "next"
	finished        wsMessageTypes = "finished"
)

type wsMessageTypes string

type wsMessage struct {
	Type    wsMessageTypes     `json:"type"`
	Session primitive.ObjectID `json:"quizid"`
	SessID  string             `json:"sessid"`
}

type joinedMessage struct {
	wsMessage
}

type participantMessage struct {
	wsMessage
	Amount int `json:"amount"`
}

type countdownMessage struct {
	wsMessage
	CurrentQuestion models.Question `json:"question"`
}

type openMessage struct {
	wsMessage
	CurrentQuestion models.Question `json:"question"`
	TimeLimit       int             `json:"timeLimit"`
}

type answerMessage struct {
	wsMessage
	Answer primitive.ObjectID `json:"answer"`
}

type answeredMessage struct {
	wsMessage
	Amount int `json:"amount"`
}

type confirmedAnswerMessage struct {
	wsMessage
	OK bool `json:"ok"`
}

type resultsMessage struct {
	wsMessage
	Question models.Question `json:"question"`
	Last     bool            `json:"last"`
}

type finishedMessage struct {
	wsMessage
}

func wsCheck(c *fiber.Ctx) error {
	fmt.Println(c.Get("origin"))
	if websocket.IsWebSocketUpgrade(c) {
		sess, uuidSess, _ := store.Get(c)
		u, ok := sess.Get("user").(models.User)
		if !ok {
			u = uuidSess.Get("user").(models.User)
		}
		c.Locals("user", u)
		c.Locals("sessid", uuidSess.ID())
		return c.Next()
	}
	return c.SendStatus(403)
}

func ws(c *websocket.Conn) {
	protocol := wsProtocol(c.Subprotocol())
	switch protocol {
	case joinProtocol:
		joinWS(c)
	case hostProtocol:
		hostWS(c)
	default:
		c.Close()
	}
}

func joinWS(c *websocket.Conn) {
	defer c.Close()
	var (
		msg []byte
		err error
	)
	code := c.Query("session")
	sessID := c.Locals("sessid").(string)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan bson.Raw)
	go notifyJoinOnEvent(code, ch, ctx)
	go func() {
	channelLoop:
		for update := range ch {
			doc := update.Lookup("fullDocument")
			var s models.Session
			if err = doc.Unmarshal(&s); err != nil {
				fmt.Println(err)
			}
			uf := update.Lookup("updateDescription.updatedFields").String()
			if strings.Contains(uf, "state") {
				switch s.State {
				case models.QuestionCountdown:
					message := countdownMessage{}
					message.Type = countdown
					message.Session = s.ID
					message.SessID = sessID
					var q models.Question
					for _, v := range s.Questions {
						if v.ID == s.CurrentQuestion {
							q = *v
						}
					}
					message.CurrentQuestion = q
					if err = c.WriteJSON(message); err != nil {
						fmt.Printf("error while writing json: %v", err)
					}
				case models.QuestionOpen:
					message := openMessage{}
					message.Type = open
					message.Session = s.ID
					message.SessID = sessID
					message.TimeLimit = s.QuestionTimer
					var q models.Question
					for _, v := range s.Questions {
						if v.ID == s.CurrentQuestion {
							q = *v
						}
					}
					message.CurrentQuestion = q
					if err = c.WriteJSON(message); err != nil {
						fmt.Printf("error while writing json: %v", err)
					}
				case models.QuestionClosed:
					message := resultsMessage{}
					message.Type = results
					message.Session = s.ID
					message.SessID = sessID
					var q models.Question
					for _, v := range s.Questions {
						if v.ID == s.CurrentQuestion {
							q = *v
						}
					}
					message.Question = q
					if err = c.WriteJSON(message); err != nil {
						fmt.Printf("error while writing json: %v", err)
					}
				case models.Finished:
					message := finishedMessage{}
					message.Type = finished
					message.Session = s.ID
					message.SessID = sessID
					if err = c.WriteJSON(message); err != nil {
						fmt.Printf("error while writing json: %v", err)
					}
					break channelLoop
				}
			}
		}
	}()
	var sID struct {
		ID primitive.ObjectID `bson:"_id"`
	}
	if err := db.
		Database().
		Collection("sessions").
		FindOne(ctx, bson.M{"code": code}).
		Decode(&sID); err != nil {
		fmt.Printf("error while getting session code: %v", err)
		return
	}
	message := joinedMessage{}
	message.Type = joined
	message.Session = sID.ID
	message.SessID = sessID
	c.WriteJSON(message)
	for {
		if _, msg, err = c.ReadMessage(); err != nil {
			log.Printf("read error: %v", err)
			break
		}
		var message answerMessage
		if err = json.Unmarshal(msg, &message); err != nil {
			log.Printf("json parse error: %v", err)
			break
		}
		if sessID != message.SessID {
			c.WriteMessage(1, []byte("the session id you provided is incorrect. Closing connection"))
			break
		}
		if err = db.
			Database().
			Collection("sessions").
			FindOneAndUpdate(ctx, bson.M{
				"_id":                   message.Session,
				"participants.user":     c.Locals("user").(models.User).ID,
				"questions.answers._id": message.Answer,
			}, bson.M{
				"$push": bson.M{
					"questions.$.answers.$.participants": c.Locals("user").(models.User).ID,
				},
			}).
			Err(); err != nil {
			fmt.Printf("error while pushing answer into db: %v", err)
			confirmMessage := confirmedAnswerMessage{}
			confirmMessage.Type = confirmedAnswer
			confirmMessage.Session = sID.ID
			confirmMessage.SessID = sessID
			confirmMessage.OK = false
			c.WriteJSON(confirmMessage)
			break
		}
		confirmMessage := confirmedAnswerMessage{}
		confirmMessage.Type = confirmedAnswer
		confirmMessage.Session = sID.ID
		confirmMessage.SessID = sessID
		confirmMessage.OK = true
		c.WriteJSON(confirmMessage)
	}
}

func hostWS(c *websocket.Conn) {
	defer c.Close()
	var (
		msg []byte
		err error
	)
	id, _ := primitive.ObjectIDFromHex(c.Query("session"))
	sessID := c.Locals("sessid").(string)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan bson.Raw)
	go notifyHostOnEvent(id, ch, ctx)
	go func() {
	channelLoop:
		for {
			for update := range ch {
				doc := update.Lookup("fullDocument")
				var s models.Session
				if err = doc.Unmarshal(&s); err != nil {
					fmt.Println(err)
				}
				uf := update.Lookup("updateDescription.updatedFields").String()
				if strings.Contains(uf, "state") {
					switch s.State {
					case models.QuestionCountdown:
						message := countdownMessage{}
						message.Type = countdown
						message.Session = s.ID
						message.SessID = sessID
						var q models.Question
						for _, v := range s.Questions {
							if v.ID == s.CurrentQuestion {
								q = *v
							}
						}
						message.CurrentQuestion = q
						if err = c.WriteJSON(message); err != nil {
							fmt.Printf("error while writing json: %v", err)
						}
					case models.QuestionOpen:
						message := openMessage{}
						message.Type = open
						message.Session = s.ID
						message.SessID = sessID
						message.TimeLimit = s.QuestionTimer
						var q models.Question
						for _, v := range s.Questions {
							if v.ID == s.CurrentQuestion {
								q = *v
							}
						}
						message.CurrentQuestion = q
						if err = c.WriteJSON(message); err != nil {
							fmt.Printf("error while writing json: %v", err)
						}
					case models.QuestionClosed:
						message := resultsMessage{}
						message.Type = results
						message.Session = s.ID
						message.SessID = sessID
						var q models.Question
						for i, v := range s.Questions {
							if v.ID == s.CurrentQuestion {
								q = *v
								if i == len(s.Questions)-1 {
									message.Last = true
								}
							}
						}
						message.Question = q
						if err = c.WriteJSON(message); err != nil {
							fmt.Printf("error while writing json: %v", err)
						}
					case models.Finished:
						message := finishedMessage{}
						message.Type = finished
						message.Session = s.ID
						message.SessID = sessID
						if err = c.WriteJSON(message); err != nil {
							fmt.Printf("error while writing json: %v", err)
						}
						break channelLoop
					}
				} else if strings.Contains(uf, "questions") {
					var totalAns int
					for _, v := range s.Questions {
						if v.ID == s.CurrentQuestion {
							for _, a := range v.Answers {
								totalAns += len(a.Participants)
							}
							break
						}
					}
					message := answeredMessage{}
					message.Type = joined
					message.Session = s.ID
					message.SessID = sessID
					message.Amount = totalAns
					if err := c.WriteJSON(message); err != nil {
						fmt.Printf("writing error: %v", err)
					}
					if totalAns != len(s.Participants) ||
						(totalAns == len(s.Participants) &&
							s.State == models.QuestionClosed) {
						continue
					}
					if err = db.
						Database().
						Collection("sessions").
						FindOneAndUpdate(
							ctx,
							bson.M{"_id": s.ID},
							bson.M{"$set": bson.M{"state": models.QuestionClosed}}).
						Err(); err != nil {
						fmt.Printf("error while updating state: %v", err)
					}
				} else if strings.Contains(uf, "participants") && s.State == models.Waiting {
					message := participantMessage{}
					message.Type = participant
					message.Session = s.ID
					message.SessID = sessID
					message.Amount = len(s.Participants)
					if err := c.WriteJSON(message); err != nil {
						fmt.Printf("writing error: %v", err)
					}
				}
			}
		}
	}()
infiniteLoop:
	for {
		if _, msg, err = c.ReadMessage(); err != nil {
			log.Printf("read error: %v", err)
			break
		}
		var recMessage wsMessage

		if err = json.Unmarshal(msg, &recMessage); err != nil {
			log.Printf("json parse error: %v", err)
			break
		}
		if sessID != recMessage.SessID {
			c.WriteMessage(1, []byte("the session id you provided is incorrect. Closing connection"))
			break
		}
		switch recMessage.Type {
		case start:
			var s models.Session
			if err := db.
				Database().
				Collection("sessions").
				FindOne(
					ctx,
					bson.M{"_id": recMessage.Session}).
				Decode(&s); err != nil {
				log.Printf("findOne error: %v", err)
				break infiniteLoop
			}
			q := s.Questions[0]
			if err := db.
				Database().
				Collection("sessions").
				FindOneAndUpdate(
					ctx,
					bson.M{"_id": recMessage.Session},
					bson.M{
						"state":   models.QuestionCountdown,
						"current": q.ID,
					}).
				Err(); err != nil {
				log.Printf("findOne error: %v", err)
				break infiniteLoop
			}
		case next:
			var s models.Session
			if err := db.
				Database().
				Collection("sessions").
				FindOne(
					ctx,
					bson.M{"_id": recMessage.Session}).
				Decode(&s); err != nil {
				log.Printf("findOne error: %v", err)
				break infiniteLoop
			}
			var state models.SessionState
			var currentQ models.Question
			for i, q := range s.Questions {
				if q.ID == s.CurrentQuestion {
					if i == len(s.Questions)-1 {
						state = models.Finished
						break
					}
					currentQ = *s.Questions[i+1]
				}
			}
			if err := db.
				Database().
				Collection("sessions").
				FindOneAndUpdate(
					ctx,
					bson.M{"_id": recMessage.Session},
					bson.M{
						"state":   state,
						"current": currentQ,
					}).
				Err(); err != nil {
				log.Printf("findOne error: %v", err)
				break infiniteLoop
			}
		default:
			log.Printf("invalid message type provided: %v", recMessage.Type)
		}
	}
}

func notifyJoinOnEvent(code string, c chan<- bson.Raw, ctx context.Context) {
	cs, err := db.
		Database().
		Collection("sessions").
		Watch(ctx, mongo.Pipeline{
			bson.D{{
				Key: "$match", Value: bson.D{{
					Key: "fullDocument.code", Value: code,
				}},
			}},
			bson.D{{
				Key:   "updateDescription.updatedFields",
				Value: bson.D{{Key: "$each", Value: bson.A{"state", "current"}}},
			}},
			bson.D{{
				Key:   "operationType",
				Value: "update",
			}},
		})
	if err != nil {
		close(c)
		return
	}
	go func() {
		<-ctx.Done()
		cs.Close(ctx)
	}()
	for {
		ok := cs.Next(ctx)
		if !ok {
			break
		}
		c <- cs.Current
	}
	cs.Close(ctx)
	close(c)
}

func notifyHostOnEvent(id primitive.ObjectID, c chan<- bson.Raw, ctx context.Context) {
	cs, err := db.
		Database().
		Collection("sessions").
		Watch(ctx, mongo.Pipeline{
			bson.D{{
				Key: "$match", Value: bson.D{{
					Key: "fullDocument._id", Value: id,
				}},
			}},
			bson.D{{
				Key:   "updateDescription.updatedFields",
				Value: bson.D{{Key: "$each", Value: bson.A{"state", "participants", "questions"}}},
			}},
			bson.D{{
				Key:   "operationType",
				Value: "update",
			}},
		})
	if err != nil {
		close(c)
		return
	}
	go func() {
		<-ctx.Done()
		cs.Close(ctx)
	}()
	for {
		ok := cs.Next(ctx)
		if !ok {
			break
		}
		c <- cs.Current
	}
	cs.Close(ctx)
	close(c)
}
