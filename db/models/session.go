package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Session struct {
	ID           primitive.ObjectID   `json:"_id" bson:"_id"`
	Name         string               `json:"name" bson:"name"`
	Owner        primitive.ObjectID   `json:"owner" bson:"owner"`
	Participants []primitive.ObjectID `json:"participants" bson:"participants"`
}

type SessionInput struct {
	Name string `json:"name" bson:"name"`
}
