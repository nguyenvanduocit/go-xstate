package creditcheck_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	api "github.com/nguyenvanduocit/go-xstate/examples/server/creditcheck"
)

// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/mongodb-credit-check-api/services/machineLogicService.ts#L16
// Related JS trace: scripts/trace/mongodb-credit-check-api/bureau-failure.ts.
func TestVerifyCredentials(t *testing.T) {
	svc := api.NewServices(api.NewMemoryStore(), api.Env{}, nil)
	valid := func() map[string]any {
		return map[string]any{"type": "Submit", "SSN": "123456789", "firstName": "Gavin", "lastName": "Bauman"}
	}
	with := func(key string, v any) map[string]any {
		m := valid()
		if v == nil {
			delete(m, key)
		} else {
			m[key] = v
		}
		return m
	}
	cases := []struct {
		name  string
		event map[string]any
		bad   string // field named by the error; empty when valid
	}{
		{"valid", valid(), ""},
		{"name of 255 characters", with("firstName", strings.Repeat("a", 255)), ""},
		{"name of 256 characters", with("firstName", strings.Repeat("a", 256)), "firstName"},
		{"emoji counts as two UTF-16 units", with("lastName", strings.Repeat("😀", 127)), ""},
		{"emoji overflow", with("lastName", strings.Repeat("😀", 128)), "lastName"},
		{"empty firstName", with("firstName", ""), "firstName"},
		{"missing lastName", with("lastName", nil), "lastName"},
		{"non-string lastName", with("lastName", 123.0), "lastName"},
		{"SSN of 8", with("SSN", "12345678"), "SSN"},
		{"SSN of 10", with("SSN", "1234567890"), "SSN"},
		{"non-string SSN", with("SSN", 123456789.0), "SSN"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := svc.VerifyCredentials(context.Background(), c.event)
			if c.bad == "" {
				require.NoError(t, err)
				require.Equal(t, api.UserCredential{
					FirstName: c.event["firstName"].(string),
					LastName:  c.event["lastName"].(string),
					SSN:       "123456789",
				}, got)
				return
			}
			require.EqualError(t, err, "Invalid Credentials. Details: invalid "+c.bad)
		})
	}
}

// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/mongodb-credit-check-api/services/machineLogicService.ts#L29
// Related JS trace: scripts/trace/mongodb-credit-check-api/bureau-failure.ts.
func TestDetermineMiddleScoreSortsAsStrings(t *testing.T) {
	svc := api.NewServices(api.NewMemoryStore(), api.Env{}, nil)
	scores := []int{9, 10, 100}
	// JS: [9, 10, 100].sort()[1] === 100 (default sort compares decimal strings)
	got, err := svc.DetermineMiddleScore(context.Background(), scores)
	require.NoError(t, err)
	require.Equal(t, 100, got)
	require.Equal(t, []int{9, 10, 100}, scores, "input is not reordered")
}

// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/mongodb-credit-check-api/services/machineLogicService.ts#L82
// Related JS trace: scripts/trace/mongodb-credit-check-api/bureau-failure.ts.
func TestGenerateInterestRate(t *testing.T) {
	svc := api.NewServices(api.NewMemoryStore(), api.Env{
		Sleep:  func(context.Context, time.Duration) error { return nil },
		Random: func() float64 { return 0 },
	}, nil)
	for score, want := range map[int]float64{850: 3.5, 701: 3.5, 700: 5, 601: 5, 600: 200, 300: 200} {
		got, err := svc.GenerateInterestRate(context.Background(), score)
		require.NoError(t, err)
		require.Equal(t, want, got, "score %d", score)
	}
}

// range({min, max}) is Math.floor(random * (max - min) + min); the bureau sleeps first, then scores.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/mongodb-credit-check-api/services/machineLogicService.ts#L59
// Related JS trace: scripts/trace/mongodb-credit-check-api/bureau-failure.ts.
func TestCheckBureauUsesRandomForSleepThenScore(t *testing.T) {
	randoms := []float64{0.5, 0.5, 0, 0.999999}
	var slept []time.Duration
	svc := api.NewServices(api.NewMemoryStore(), api.Env{
		Sleep:  func(_ context.Context, d time.Duration) error { slept = append(slept, d); return nil },
		Random: func() float64 { r := randoms[0]; randoms = randoms[1:]; return r },
	}, nil)

	score, err := svc.CheckBureau(context.Background(), api.BureauQuery{SSN: "123456789", BureauName: api.GavUnion})
	require.NoError(t, err)
	require.Equal(t, 575, score)
	require.Equal(t, []time.Duration{5500 * time.Millisecond}, slept)

	// the rate service sleeps with the next random and ignores the rest
	_, err = svc.GenerateInterestRate(context.Background(), 650)
	require.NoError(t, err)
	require.Equal(t, 1000*time.Millisecond, slept[1])

	score, err = svc.CheckBureau(context.Background(), api.BureauQuery{SSN: "123456789", BureauName: "Unknown"})
	require.NoError(t, err)
	require.Zero(t, score, "unknown bureau: JS returns undefined without sleeping")
	require.Len(t, slept, 2)
	require.Equal(t, []float64{0.999999}, randoms)
}

// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/mongodb-credit-check-api/services/machineLogicService.ts#L125
// Related JS trace: scripts/trace/mongodb-credit-check-api/bureau-failure.ts.
func TestDefaultSleepStopsWithContext(t *testing.T) {
	svc := api.NewServices(api.NewMemoryStore(), api.Env{Random: func() float64 { return 0.5 }}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := svc.CheckBureau(ctx, api.BureauQuery{SSN: "123456789", BureauName: api.EquiGavin})
	require.ErrorIs(t, err, context.Canceled)
}

// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/mongodb-credit-check-api/services/machineLogicService.ts#L37
// Related JS trace: scripts/trace/mongodb-credit-check-api/bureau-failure.ts.
func TestReportsAndProfilesRoundTripThroughStore(t *testing.T) {
	store := api.NewMemoryStore()
	var lines []string
	svc := api.NewServices(store, api.Env{}, func(l string) { lines = append(lines, l) })

	report, err := svc.CheckReportsTable(context.Background(), api.BureauQuery{SSN: "123456789", BureauName: api.EquiGavin})
	require.NoError(t, err)
	require.Nil(t, report)

	require.NoError(t, svc.SaveCreditReport(api.CreditReport{SSN: "123456789", BureauName: api.EquiGavin, CreditScore: 700}))
	require.NoError(t, svc.SaveCreditReport(api.CreditReport{SSN: "123456789", BureauName: api.EquiGavin, CreditScore: 710}))
	report, err = svc.CheckReportsTable(context.Background(), api.BureauQuery{SSN: "123456789", BureauName: api.EquiGavin})
	require.NoError(t, err)
	require.Equal(t, &api.CreditReport{SSN: "123456789", BureauName: api.EquiGavin, CreditScore: 710}, report, "replaceOne upserts by ssn and bureau")
	require.Equal(t, []string{"Checking for an existing report....", "Checking for an existing report...."}, lines)

	profile := api.CreditProfile{SSN: "123456789", MiddleScore: 650}
	require.NoError(t, svc.SaveCreditProfile(profile))
	got, ok := store.CreditProfile("123456789")
	require.True(t, ok)
	require.Equal(t, profile, got)
}
