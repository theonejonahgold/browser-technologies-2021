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
	"go.mongodb.org/mongo-driver/mongo/options"
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
	Quiz models.Session `json:"quiz"`
	Host string         `json:"host"`
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
	AmtPart         int             `json:"participantAmount"`
}

type answerMessage struct {
	wsMessage
	Answer primitive.ObjectID `json:"answer"`
}

type answeredMessage struct {
	wsMessage
	Amount  int `json:"amount"`
	AmtPart int `json:"participantAmount"`
}

type confirmedAnswerMessage struct {
	wsMessage
	OK              bool            `json:"ok"`
	Answer          string          `json:"answer"`
	CurrentQuestion models.Question `json:"question"`
}

type resultsMessage struct {
	wsMessage
	Question models.Question `json:"question"`
	Last     bool            `json:"last"`
	AmtPart  int             `json:"participantAmount"`
}

type finishedMessage struct {
	wsMessage
}

func wsCheck(c *fiber.Ctx) error {
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
		mt  int
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
				continue
			}
			switch s.State {
			case models.QuestionCountdown:
				message := countdownMessage{}
				message.Type = countdown
				message.Session = s.ID
				message.SessID = sessID
				for _, v := range s.Questions {
					if v.ID == s.CurrentQuestion {
						message.CurrentQuestion = *v
					}
				}
				if err = c.WriteJSON(message); err != nil {
					fmt.Printf("error while writing json: %v\n", err)
				}
			case models.QuestionOpen:
				message := openMessage{}
				message.Type = open
				message.Session = s.ID
				message.SessID = sessID
				message.TimeLimit = s.QuestionTimer
				for _, v := range s.Questions {
					if v.ID == s.CurrentQuestion {
						message.CurrentQuestion = *v
					}
				}
				if err = c.WriteJSON(message); err != nil {
					fmt.Printf("error while writing json: %v\n", err)
				}
			case models.QuestionClosed:
				message := resultsMessage{}
				message.Type = results
				message.Session = s.ID
				message.SessID = sessID
				message.AmtPart = len(s.Participants)
				for _, v := range s.Questions {
					if v.ID == s.CurrentQuestion {
						message.Question = *v
					}
				}
				if err = c.WriteJSON(message); err != nil {
					fmt.Printf("error while writing json: %v\n", err)
				}
			case models.Finished:
				message := finishedMessage{}
				message.Type = finished
				message.Session = s.ID
				message.SessID = sessID
				if err = c.WriteJSON(message); err != nil {
					fmt.Printf("error while writing json: %v\n", err)
				}
				break channelLoop
			}
		}
	}()
	var s models.Session
	if err := db.
		Database().
		Collection("sessions").
		FindOne(ctx, bson.M{"code": code}).
		Decode(&s); err != nil {
		fmt.Printf("error while getting session: %v\n", err)
		return
	}
	var host models.User
	if err := db.
		Database().
		Collection("users").
		FindOne(ctx, bson.M{"_id": s.Owner}).
		Decode(&host); err != nil {
		fmt.Printf("error while getting session host: %v\n", err)
		return
	}
	switch s.State {
	case models.Waiting:
		message := joinedMessage{}
		message.Type = joined
		message.Session = s.ID
		message.SessID = sessID
		message.Quiz = s
		message.Host = host.Username
		c.WriteJSON(message)
	case models.QuestionCountdown:
		message := countdownMessage{}
		message.Type = countdown
		message.Session = s.ID
		message.SessID = sessID
		for _, v := range s.Questions {
			if v.ID == s.CurrentQuestion {
				message.CurrentQuestion = *v
			}
		}
		if err = c.WriteJSON(message); err != nil {
			fmt.Printf("error while writing json: %v\n", err)
		}
	case models.QuestionOpen:
		message := openMessage{}
		message.Type = open
		message.Session = s.ID
		message.SessID = sessID
		message.TimeLimit = s.QuestionTimer
		message.AmtPart = len(s.Participants)
		for _, v := range s.Questions {
			if v.ID == s.CurrentQuestion {
				message.CurrentQuestion = *v
			}
		}
		if err = c.WriteJSON(message); err != nil {
			fmt.Printf("error while writing json: %v\n", err)
		}
	case models.QuestionClosed:
		message := resultsMessage{}
		message.Type = results
		message.Session = s.ID
		message.SessID = sessID
		for i, v := range s.Questions {
			if v.ID == s.CurrentQuestion {
				message.Question = *v
				if i == len(s.Questions)-1 {
					message.Last = true
				}
			}
		}
		if err = c.WriteJSON(message); err != nil {
			fmt.Printf("error while writing json: %v\n", err)
		}
	}
	for {
		if mt, msg, err = c.ReadMessage(); err != nil {
			log.Printf("read error: %v", err)
			break
		}
		if mt == websocket.CloseMessage {
			log.Printf("socket closed, disconnecting")
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
		fmt.Printf("%+v\n", message)
		if err = db.
			Database().
			Collection("sessions").
			FindOneAndUpdate(ctx, bson.M{
				"_id": message.Session,
			}, bson.M{
				"$push": bson.M{
					"questions.$[].answers.$[answer].participants": c.Locals("user").(models.User).ID,
				},
			}, options.FindOneAndUpdate().SetArrayFilters(options.ArrayFilters{
				Filters: []interface{}{
					bson.M{
						"answer._id": message.Answer,
					},
				},
			})).
			Err(); err != nil {
			fmt.Printf("error while pushing answer into db: %v\n", err)
			confirmMessage := confirmedAnswerMessage{}
			confirmMessage.Type = confirmedAnswer
			confirmMessage.Session = s.ID
			confirmMessage.SessID = sessID
			confirmMessage.OK = false
			c.WriteJSON(confirmMessage)
			break
		}
		confirmMessage := confirmedAnswerMessage{}
		confirmMessage.Type = confirmedAnswer
		confirmMessage.Session = s.ID
		confirmMessage.SessID = sessID
		confirmMessage.OK = true
		for _, sq := range s.Questions {
			if sq.ID == s.CurrentQuestion {
				confirmMessage.CurrentQuestion = *sq
			}
			for _, qa := range sq.Answers {
				if qa.ID == message.Answer {
					confirmMessage.Answer = qa.Title
				}
			}
		}
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
		for update := range ch {
			doc := update.Lookup("fullDocument")
			var s models.Session
			if err = doc.Unmarshal(&s); err != nil {
				fmt.Println(err)
				continue
			}
			uf := update.Lookup("updateDescription", "updatedFields").String()
			fmt.Println("updates", uf)
			if strings.Contains(uf, "state") {
				switch s.State {
				case models.QuestionCountdown:
					message := countdownMessage{}
					message.Type = countdown
					message.Session = s.ID
					message.SessID = sessID
					for _, v := range s.Questions {
						if v.ID == s.CurrentQuestion {
							message.CurrentQuestion = *v
						}
					}
					if err = c.WriteJSON(message); err != nil {
						fmt.Printf("error while writing json: %v\n", err)
					}
				case models.QuestionOpen:
					message := openMessage{}
					message.Type = open
					message.Session = s.ID
					message.SessID = sessID
					message.TimeLimit = s.QuestionTimer
					message.AmtPart = len(s.Participants)
					for _, v := range s.Questions {
						if v.ID == s.CurrentQuestion {
							message.CurrentQuestion = *v
						}
					}
					if err = c.WriteJSON(message); err != nil {
						fmt.Printf("error while writing json: %v\n", err)
					}
				case models.QuestionClosed:
					message := resultsMessage{}
					message.Type = results
					message.Session = s.ID
					message.SessID = sessID
					message.AmtPart = len(s.Participants)
					for i, v := range s.Questions {
						if v.ID == s.CurrentQuestion {
							message.Question = *v
							if i == len(s.Questions)-1 {
								message.Last = true
							}
						}
					}
					if err = c.WriteJSON(message); err != nil {
						fmt.Printf("error while writing json: %v\n", err)
					}
				case models.Finished:
					message := finishedMessage{}
					message.Type = finished
					message.Session = s.ID
					message.SessID = sessID
					if err = c.WriteJSON(message); err != nil {
						fmt.Printf("error while writing json: %v\n", err)
					}
					break channelLoop
				}
			}
			if strings.Contains(uf, "questions") && s.State == models.QuestionOpen {
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
				message.Type = answerAmt
				message.Session = s.ID
				message.SessID = sessID
				message.Amount = totalAns
				message.AmtPart = len(s.Participants)
				if err := c.WriteJSON(message); err != nil {
					fmt.Printf("writing error: %v\n", err)
				}
				if (totalAns != len(s.Participants) &&
					s.State == models.QuestionOpen) ||
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
					fmt.Printf("error while updating state: %v\n", err)
				}
			}
			if strings.Contains(uf, "participants") && s.State == models.Waiting {
				message := participantMessage{}
				message.Type = participant
				message.Session = s.ID
				message.SessID = sessID
				message.Amount = len(s.Participants)
				if err := c.WriteJSON(message); err != nil {
					fmt.Printf("writing error: %v\n", err)
				}
			}
		}
	}()
	var s models.Session
	if err := db.
		Database().
		Collection("sessions").
		FindOne(ctx, bson.M{"_id": id}).
		Decode(&s); err != nil {
		fmt.Printf("error while getting session: %v\n", err)
		return
	}
	var host models.User
	if err := db.
		Database().
		Collection("users").
		FindOne(ctx, bson.M{"_id": s.Owner}).
		Decode(&host); err != nil {
		fmt.Printf("error while getting session host: %v\n", err)
		return
	}
	switch s.State {
	case models.Waiting:
		message := joinedMessage{}
		message.Type = joined
		message.Session = s.ID
		message.SessID = sessID
		message.Quiz = s
		message.Host = host.Username
		c.WriteJSON(message)
	case models.QuestionCountdown:
		message := countdownMessage{}
		message.Type = countdown
		message.Session = s.ID
		message.SessID = sessID
		for _, v := range s.Questions {
			if v.ID == s.CurrentQuestion {
				message.CurrentQuestion = *v
			}
		}
		if err = c.WriteJSON(message); err != nil {
			fmt.Printf("error while writing json: %v\n", err)
		}
	case models.QuestionOpen:
		message := openMessage{}
		message.Type = open
		message.Session = s.ID
		message.SessID = sessID
		message.TimeLimit = s.QuestionTimer
		message.AmtPart = len(s.Participants)
		for _, v := range s.Questions {
			if v.ID == s.CurrentQuestion {
				message.CurrentQuestion = *v
			}
		}
		if err = c.WriteJSON(message); err != nil {
			fmt.Printf("error while writing json: %v\n", err)
		}
	case models.QuestionClosed:
		message := resultsMessage{}
		message.Type = results
		message.Session = s.ID
		message.SessID = sessID
		for i, v := range s.Questions {
			if v.ID == s.CurrentQuestion {
				message.Question = *v
				if i == len(s.Questions)-1 {
					message.Last = true
				}
			}
		}
		if err = c.WriteJSON(message); err != nil {
			fmt.Printf("error while writing json: %v\n", err)
		}
	}
infiniteLoop:
	for {
		if _, msg, err = c.ReadMessage(); err != nil {
			log.Printf("read error: %v\n", err)
			break
		}
		var recMessage wsMessage

		if err = json.Unmarshal(msg, &recMessage); err != nil {
			log.Printf("json parse error: %v\n", err)
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
				log.Printf("findOne error: %v\n", err)
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
						"$set": bson.M{
							"state":   models.QuestionCountdown,
							"current": q.ID,
						},
					},
					options.FindOneAndUpdate().SetReturnDocument(options.After)).
				Decode(&s); err != nil {
				log.Printf("findOne error: %v\n", err)
				break infiniteLoop
			}
			go changeStateAfterCountdown(s)
		case next:
			var s models.Session
			if err := db.
				Database().
				Collection("sessions").
				FindOne(
					ctx,
					bson.M{"_id": recMessage.Session}).
				Decode(&s); err != nil {
				log.Printf("findOne error: %v\n", err)
				break infiniteLoop
			}
			var state models.SessionState
			var currentQ primitive.ObjectID
			for i, q := range s.Questions {
				if q.ID == s.CurrentQuestion {
					if i == len(s.Questions)-1 {
						state = models.Finished
						break
					}
					currentQ = s.Questions[i+1].ID
					state = models.QuestionCountdown
				}
			}
			if err := db.
				Database().
				Collection("sessions").
				FindOneAndUpdate(
					ctx,
					bson.M{"_id": recMessage.Session},
					bson.M{
						"$set": bson.M{
							"state":   state,
							"current": currentQ,
						},
					}).
				Err(); err != nil {
				log.Printf("findOne error: %v\n", err)
				break infiniteLoop
			}
			go changeStateAfterCountdown(s)
		default:
			log.Printf("invalid message type provided: %v\n", recMessage.Type)
		}
	}
}

func notifyJoinOnEvent(code string, c chan<- bson.Raw, ctx context.Context) {
	cs, err := db.
		Database().
		Collection("sessions").
		Watch(ctx, mongo.Pipeline{
			bson.D{{
				Key: "$match",
				Value: bson.D{{Key: "$and",
					Value: bson.A{
						bson.D{{
							Key: "fullDocument.code", Value: code,
						}},
						bson.D{{
							Key:   "updateDescription.updatedFields.state",
							Value: bson.D{{Key: "$exists", Value: true}},
						}},
						bson.D{{
							Key:   "operationType",
							Value: "update",
						}},
					},
				}},
			}},
		}, options.ChangeStream().SetFullDocument(options.UpdateLookup))
	if err != nil {
		fmt.Printf("watch error: %v\n", err)
		close(c)
		return
	}
	defer cs.Close(ctx)
	defer close(c)
	for {
		ok := cs.Next(ctx)
		if !ok {
			fmt.Println("not okay, breaking")
			break
		}
		c <- cs.Current
	}
}

func notifyHostOnEvent(id primitive.ObjectID, c chan<- bson.Raw, ctx context.Context) {
	cs, err := db.
		Database().
		Collection("sessions").
		Watch(ctx, mongo.Pipeline{
			bson.D{{
				Key: "$match",
				Value: bson.D{{Key: "$and",
					Value: bson.A{
						bson.D{{
							Key: "fullDocument._id", Value: id,
						}},
						bson.D{{
							Key:   "operationType",
							Value: "update",
						}},
					},
				}},
			}},
		}, options.ChangeStream().SetFullDocument(options.UpdateLookup))
	if err != nil {
		fmt.Printf("watch error: %v\n", err)
		close(c)
		return
	}
	defer cs.Close(ctx)
	defer close(c)
	for {
		ok := cs.Next(ctx)
		if !ok {
			fmt.Println("not okay, breaking")
			break
		}
		c <- cs.Current
	}
}
