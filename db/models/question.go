package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Question struct {
	Session primitive.ObjectID `json:"session" bson:"session"`
	ID      primitive.ObjectID `json:"_id" bson:"_id"`
	Title   string             `json:"title" bson:"title"`
	Answers []*Answer          `json:"answers" bson:"answers"`
}

type QuestionInput struct {
	Title   string         `json:"title" bson:"title"`
	Answers []*AnswerInput `json:"answers" bson:"answers"`
}

type Answer struct {
	ID    primitive.ObjectID `json:"_id" bson:"_id"`
	Title string             `json:"title" bson:"title"`
}

type AnswerInput struct {
	Title string `json:"title" bson:"title"`
}
