//go:build mongodb

package persistedstate

import (
	"context"
	"encoding/json"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MongoStore is the Store backed by the `donuts` collection of the `donut-maker` database.
// It is compiled only with `-tags mongodb`: the module's go.sum does not carry the driver's
// transitive dependencies (run `go mod tidy` in examples/ before building with the tag).
type MongoStore struct {
	client     *mongo.Client
	collection *mongo.Collection
}

// NewMongoStore mirrors the MongoClient setup of main.ts (serverApi v1, donut-maker.donuts).
func NewMongoStore(uri string) (*MongoStore, error) {
	client, err := mongo.Connect(options.Client().
		ApplyURI(uri).
		SetServerAPIOptions(options.ServerAPI(options.ServerAPIVersion1)))
	if err != nil {
		return nil, err
	}
	return &MongoStore{client: client, collection: client.Database("donut-maker").Collection("donuts")}, nil
}

// Connect verifies the deployment is reachable (mongo.Connect itself connects lazily).
func (s *MongoStore) Connect(ctx context.Context) error { return s.client.Ping(ctx, nil) }

// FindOne returns the first document as JSON-shaped data (relaxed extended JSON), or nil.
func (s *MongoStore) FindOne(ctx context.Context) (map[string]any, error) {
	var raw bson.Raw
	err := s.collection.FindOne(ctx, bson.D{}).Decode(&raw)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	text, err := bson.MarshalExtJSON(raw, false, false)
	if err != nil {
		return nil, err
	}
	var doc map[string]any
	return doc, json.Unmarshal(text, &doc)
}

// UpdateOne upserts `{ $set: { persistedState } }` into the document that has a persistedState.
func (s *MongoStore) UpdateOne(ctx context.Context, persistedState map[string]any) (UpdateResult, error) {
	filter := bson.D{{Key: "persistedState", Value: bson.D{{Key: "$exists", Value: true}}}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "persistedState", Value: persistedState}}}}
	res, err := s.collection.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return UpdateResult{}, err
	}
	return UpdateResult{
		Acknowledged:  res.Acknowledged,
		MatchedCount:  res.MatchedCount,
		ModifiedCount: res.ModifiedCount,
		UpsertedCount: res.UpsertedCount,
		UpsertedID:    res.UpsertedID,
	}, nil
}

// Close mirrors client.close().
func (s *MongoStore) Close(ctx context.Context) error { return s.client.Disconnect(ctx) }
