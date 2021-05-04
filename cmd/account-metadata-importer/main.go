package main

import (
	"context"
	"encoding/csv"
	"errors"
	"io"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/b-harvest/gravity-dex-backend/schema"
)

func main() {
	f, err := os.Open("accounts.csv")
	if err != nil {
		log.Fatalf("failed to open accounts.csv: %v", err)
	}
	defer f.Close()

	now := time.Now()
	rd := csv.NewReader(f)
	var docs []interface{}
	for {
		row, err := rd.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			log.Fatalf("failed to read row: %v", err)
		}
		if len(row) == 2 {
			docs = append(docs, schema.AccountMetadata{
				Address:   row[0],
				Username:  row[1],
				IsBlocked: false,
				CreatedAt: now,
			})
		}
	}

	mc, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost"))
	if err != nil {
		log.Fatalf("failed connect to mongodb: %v", err)
	}
	defer mc.Disconnect(context.Background())

	coll := mc.Database("gdex").Collection("accountMetadata")
	log.Printf("importing %d account metadata", len(docs))
	r, err := coll.InsertMany(context.Background(), docs)
	if err != nil {
		log.Fatalf("failed to insert account metadata: %v", err)
	}
	log.Printf("imported %d account metadata", len(r.InsertedIDs))
}
