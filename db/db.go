package db

import (
	"context"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var db *mongo.Database

func Connect() error {
	uri := os.Getenv("MONGO_URI")
	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}
	ctx, stop := context.WithTimeout(context.Background(), 10*time.Second)
	defer stop()
	err = client.Connect(ctx)
	if err != nil {
		return err
	}
	db = client.Database("bt")
	return err
}

func Database() *mongo.Database {
	return db
}

func Close() error {
	ctx, stop := context.WithTimeout(context.Background(), 10*time.Second)
	defer stop()
	return db.Client().Disconnect(ctx)
}
