package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	SessionStateWaiting = iota
	SessionStateQuestionStarting
	SessionStateQuestionStarted
	SessionStateQuestionResults
	SessionStateFinished
)

type Session struct {
	ID              primitive.ObjectID   `json:"_id" bson:"_id"`
	Name            string               `json:"name" bson:"name"`
	Owner           primitive.ObjectID   `json:"owner" bson:"owner"`
	QuestionTimer   time.Duration        `json:"question-timer" bson:"questionTimer"`
	Participants    []primitive.ObjectID `json:"participants" bson:"participants"`
	Questions       []*Question          `json:"questions" bson:"questions"`
	Code            string               `json:"code" bson:"code"`
	State           int                  `json:"state" bson:"state"`
	CurrentQuestion int                  `json:"currentQuestion" bson:"currentQuestion"`
}

type SessionInput struct {
	Name          string        `json:"name" bson:"name"`
	QuestionTimer time.Duration `json:"question-timer" bson:"questionTimer"`
}
