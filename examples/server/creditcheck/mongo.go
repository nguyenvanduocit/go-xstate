package creditcheck

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// DatabaseName is the database initDbConnection uses.
const DatabaseName = "creditCheck"

// MongoStore is the Store backed by MongoDB collections machineStates, creditReports and
// creditProfiles (actorService.ts and machineLogicService.ts).
type MongoStore struct {
	client   *mongo.Client
	states   *mongo.Collection
	reports  *mongo.Collection
	profiles *mongo.Collection
}

// ConnectMongo mirrors initDbConnection: it connects to uri with the stable server API.
// machineLogicService.ts reads creditReports and creditProfiles, but actorService.ts only
// ever assigns machineStates, so in JS the two other collections stay undefined and every
// report and profile operation is a no-op; here all three collections are wired.
func ConnectMongo(ctx context.Context, uri string) (*MongoStore, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri).SetServerAPIOptions(options.ServerAPI(options.ServerAPIVersion1)))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, err
	}
	db := client.Database(DatabaseName)
	return &MongoStore{
		client:   client,
		states:   db.Collection("machineStates"),
		reports:  db.Collection("creditReports"),
		profiles: db.Collection("creditProfiles"),
	}, nil
}

// Close disconnects from MongoDB.
func (m *MongoStore) Close(ctx context.Context) error { return m.client.Disconnect(ctx) }

func (m *MongoStore) FindMachineState(ctx context.Context, workflowID string) (*MachineState, error) {
	var doc bson.M
	err := m.states.FindOne(ctx, bson.D{{Key: "workflowId", Value: workflowID}}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	persisted, _ := plainValue(doc["persistedState"]).(map[string]any)
	return &MachineState{WorkflowID: workflowID, PersistedState: persisted}, nil
}

func (m *MongoStore) SaveMachineState(ctx context.Context, state MachineState) error {
	res, err := m.states.ReplaceOne(ctx, bson.D{{Key: "workflowId", Value: state.WorkflowID}}, state, options.Replace().SetUpsert(true))
	if err != nil {
		return err
	}
	if !res.Acknowledged {
		return errors.New("write was not acknowledged")
	}
	return nil
}

func (m *MongoStore) FindCreditReport(ctx context.Context, ssn, bureauName string) (*CreditReport, error) {
	var report CreditReport
	err := m.reports.FindOne(ctx, bson.D{{Key: "ssn", Value: ssn}, {Key: "bureauName", Value: bureauName}}).Decode(&report)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &report, nil
}

func (m *MongoStore) SaveCreditReport(ctx context.Context, report CreditReport) error {
	_, err := m.reports.ReplaceOne(ctx, bson.D{{Key: "ssn", Value: report.SSN}, {Key: "bureauName", Value: report.BureauName}}, report, options.Replace().SetUpsert(true))
	return err
}

func (m *MongoStore) SaveCreditProfile(ctx context.Context, profile CreditProfile) error {
	_, err := m.profiles.ReplaceOne(ctx, bson.D{{Key: "ssn", Value: profile.SSN}}, profile, options.Replace().SetUpsert(true))
	return err
}

// plainValue converts decoded BSON (bson.D, bson.M, bson.A) into the map and slice types
// of encoding/json, which is what WithSnapshot restores from.
func plainValue(v any) any {
	switch x := v.(type) {
	case bson.D:
		out := make(map[string]any, len(x))
		for _, e := range x {
			out[e.Key] = plainValue(e.Value)
		}
		return out
	case bson.M:
		out := make(map[string]any, len(x))
		for k, e := range x {
			out[k] = plainValue(e)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, e := range x {
			out[k] = plainValue(e)
		}
		return out
	case bson.A:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = plainValue(e)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = plainValue(e)
		}
		return out
	case int32:
		return float64(x)
	case int64:
		return float64(x)
	}
	return v
}
