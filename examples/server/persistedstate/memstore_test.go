package persistedstate_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"

	donut "github.com/nguyenvanduocit/go-xstate/examples/server/persistedstate"
)

// memStore is the in-memory fake of the `donut-maker.donuts` collection, matching the
// fake `mongodb` module of scripts/trace/mongodb-persisted-state/mongodb-persisted-state.stdout.ts:
// one document `{ _id: "doc-1", persistedState }`, updateOne upserts it and reports
// modifiedCount 0 when the new persistedState equals the stored one.
type memStore struct {
	doc         map[string]any
	connectErr  error
	findErr     error
	updateErr   error
	closed      bool
	updateCalls int
}

func (s *memStore) Connect(context.Context) error { return s.connectErr }

func (s *memStore) FindOne(context.Context) (map[string]any, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	if s.doc == nil {
		return nil, nil
	}
	return clone(s.doc), nil
}

func (s *memStore) UpdateOne(_ context.Context, persistedState map[string]any) (donut.UpdateResult, error) {
	s.updateCalls++
	if s.updateErr != nil {
		return donut.UpdateResult{}, s.updateErr
	}
	persistedState = clone(persistedState)
	if old, ok := s.doc["persistedState"]; ok {
		modified := int64(0)
		if !reflect.DeepEqual(old, any(persistedState)) {
			modified = 1
		}
		s.doc["persistedState"] = persistedState
		return donut.UpdateResult{Acknowledged: true, MatchedCount: 1, ModifiedCount: modified}, nil
	}
	s.doc = map[string]any{"_id": "doc-1", "persistedState": persistedState}
	return donut.UpdateResult{Acknowledged: true, UpsertedCount: 1, UpsertedID: "doc-1"}, nil
}

func (s *memStore) Close(context.Context) error { s.closed = true; return nil }

func clone(m map[string]any) map[string]any {
	raw, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		panic(err)
	}
	return out
}

var errConnect = errors.New("connect ECONNREFUSED")
