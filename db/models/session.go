package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	Creating SessionState = iota
	Waiting
	QuestionCountdown
	QuestionOpen
	QuestionClosed
	Finished
)

type SessionState int8

type Session struct {
	ID              primitive.ObjectID `json:"-" bson:"_id"`
	Name            string             `json:"name" bson:"name"`
	Owner           primitive.ObjectID `json:"owner" bson:"owner"`
	QuestionTimer   int                `json:"-" bson:"questionTimer"`
	Participants    []*Participant     `json:"-" bson:"participants"`
	Questions       []*Question        `json:"questions" bson:"questions"`
	Code            string             `json:"code" bson:"code"`
	State           SessionState       `json:"state" bson:"state"`
	CurrentQuestion primitive.ObjectID `json:"current" bson:"current"`
}

type SessionInput struct {
	Name string `json:"name" bson:"name"`
}

type Participant struct {
	ID      primitive.ObjectID `json:"_id" bson:"_id"`
	User    primitive.ObjectID `json:"user" bson:"user"`
	Strikes int                `json:"strikes" bson:"strikes"`
}
