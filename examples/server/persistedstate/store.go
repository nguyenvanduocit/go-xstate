package persistedstate

import "context"

// UpdateResult carries the fields main.ts reads from the MongoDB driver's
// updateOne result. Fields are in alphabetical order so JSON output has sorted keys.
type UpdateResult struct {
	Acknowledged  bool  `json:"acknowledged"`
	MatchedCount  int64 `json:"matchedCount"`
	ModifiedCount int64 `json:"modifiedCount"`
	UpsertedCount int64 `json:"upsertedCount"`
	UpsertedID    any   `json:"upsertedId"`
}

// Store is the slice of the MongoDB `donut-maker.donuts` collection that main.ts uses.
// Documents are JSON-shaped values (map[string]any).
type Store interface {
	// Connect mirrors client.connect().
	Connect(ctx context.Context) error
	// FindOne mirrors donutCollection.findOne(); it returns nil when the collection is empty.
	FindOne(ctx context.Context) (map[string]any, error)
	// UpdateOne mirrors donutCollection.updateOne({ persistedState: { $exists: true } },
	// { $set: { persistedState } }, { upsert: true }).
	UpdateOne(ctx context.Context, persistedState map[string]any) (UpdateResult, error)
	// Close mirrors client.close().
	Close(ctx context.Context) error
}
