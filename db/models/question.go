package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Question struct {
	ID      primitive.ObjectID `json:"_id" bson:"_id"`
	Title   string             `json:"title" bson:"title"`
	Answers []*Answer          `json:"answers" bson:"answers"`
}

type QuestionInput struct {
	Title   string         `json:"title" bson:"title"`
	Answers []*AnswerInput `json:"answers" bson:"answers"`
}

type Answer struct {
	ID           primitive.ObjectID   `json:"_id" bson:"_id"`
	Title        string               `json:"title" bson:"title"`
	Participants []primitive.ObjectID `json:"participants" bson:"participants"`
}

type AnswerInput struct {
	Title string `json:"title" bson:"title"`
}
