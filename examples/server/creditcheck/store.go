package creditcheck

import (
	"context"
	"encoding/json"
	"sync"
)

// MachineState is a document of the machineStates collection.
type MachineState struct {
	WorkflowID     string         `json:"workflowId" bson:"workflowId"`
	PersistedState map[string]any `json:"persistedState" bson:"persistedState"`
}

// Store is the persistence the example needs from MongoDB: the machineStates collection
// (actorService.ts) and the creditReports and creditProfiles collections (machineLogicService.ts).
type Store interface {
	// FindMachineState returns nil when the workflow is unknown.
	FindMachineState(ctx context.Context, workflowID string) (*MachineState, error)
	// SaveMachineState upserts by workflowId.
	SaveMachineState(ctx context.Context, state MachineState) error
	// FindCreditReport returns nil when there is no report.
	FindCreditReport(ctx context.Context, ssn, bureauName string) (*CreditReport, error)
	// SaveCreditReport upserts by ssn and bureauName.
	SaveCreditReport(ctx context.Context, report CreditReport) error
	// SaveCreditProfile upserts by SSN.
	SaveCreditProfile(ctx context.Context, profile CreditProfile) error
}

// MemoryStore is an in-memory Store. Documents are kept as JSON, as a database would
// keep them, so a restored persisted state has the types that came out of JSON.
type MemoryStore struct {
	mu       sync.Mutex
	states   map[string][]byte
	reports  map[reportID][]byte
	profiles map[string]CreditProfile
}

// NewMemoryStore returns an empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		states:   map[string][]byte{},
		reports:  map[reportID][]byte{},
		profiles: map[string]CreditProfile{},
	}
}

func (m *MemoryStore) FindMachineState(_ context.Context, workflowID string) (*MachineState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	raw, ok := m.states[workflowID]
	if !ok {
		return nil, nil
	}
	var state MachineState
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (m *MemoryStore) SaveMachineState(_ context.Context, state MachineState) error {
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[state.WorkflowID] = raw
	return nil
}

// reportID is the replaceOne filter of saveCreditReport.
type reportID struct{ ssn, bureauName string }

func (m *MemoryStore) FindCreditReport(_ context.Context, ssn, bureauName string) (*CreditReport, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	raw, ok := m.reports[reportID{ssn, bureauName}]
	if !ok {
		return nil, nil
	}
	var report CreditReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return nil, err
	}
	return &report, nil
}

func (m *MemoryStore) SaveCreditReport(_ context.Context, report CreditReport) error {
	raw, err := json.Marshal(report)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reports[reportID{report.SSN, report.BureauName}] = raw
	return nil
}

func (m *MemoryStore) SaveCreditProfile(_ context.Context, profile CreditProfile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.profiles[profile.SSN] = profile
	return nil
}

// CreditProfile returns the profile saved for ssn.
func (m *MemoryStore) CreditProfile(ssn string) (CreditProfile, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.profiles[ssn]
	return p, ok
}

// CreditReport returns the report saved for ssn and bureauName.
func (m *MemoryStore) CreditReport(ssn, bureauName string) (CreditReport, bool) {
	r, err := m.FindCreditReport(context.Background(), ssn, bureauName)
	if err != nil || r == nil {
		return CreditReport{}, false
	}
	return *r, true
}
