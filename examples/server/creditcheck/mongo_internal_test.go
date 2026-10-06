package creditcheck

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// plainValue turns decoded BSON into what encoding/json produces, which is what
// WithSnapshot restores from. The Mongo adapter itself needs a database and is not run here.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/mongodb-credit-check-api/services/actorService.ts#L33
// Related JS trace: scripts/trace/mongodb-credit-check-api/server.ts.
func TestPlainValue(t *testing.T) {
	in := bson.D{
		{Key: "status", Value: "active"},
		{Key: "count", Value: int32(2)},
		{Key: "big", Value: int64(3)},
		{Key: "score", Value: 1.5},
		{Key: "children", Value: bson.M{"a": bson.D{{Key: "src", Value: "x"}}}},
		{Key: "list", Value: bson.A{int32(1), bson.D{{Key: "k", Value: true}}}},
	}
	require.Equal(t, map[string]any{
		"status": "active",
		"count":  float64(2),
		"big":    float64(3),
		"score":  1.5,
		"children": map[string]any{
			"a": map[string]any{"src": "x"},
		},
		"list": []any{float64(1), map[string]any{"k": true}},
	}, plainValue(in))
}
