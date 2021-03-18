package models

import (
	"github.com/matthewhartstonge/argon2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var argon argon2.Config = argon2.DefaultConfig()

type User struct {
	ID       primitive.ObjectID `json:"_id" bson:"_id"`
	Username string             `json:"username" bson:"username"`
	Password string             `json:"password" bson:"password"`
}

type UserInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func VerifyUserPassword(password []byte, hash []byte) (bool, error) {
	return argon2.VerifyEncoded(password, hash)
}

func HashPassword(password []byte) ([]byte, error) {
	return argon.HashEncoded(password)
}
