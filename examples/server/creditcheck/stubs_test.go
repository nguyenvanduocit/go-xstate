package creditcheck_test

import (
	"context"
	"errors"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"

	api "github.com/nguyenvanduocit/go-xstate/examples/server/creditcheck"
)

// stubServices is the Go side of scripts/trace/mongodb-credit-check-api/lib/stub-services.ts:
// the same SSN-keyed scenarios and the same delays (ms after the call):
// verifyCredentials 20, checkReportsTable 40, checkBureau 100/200/300, determineMiddleScore 40,
// generateInterestRate 60. Credential validation and the middle score are the real services.
func stubServices(log func(string)) api.Services {
	return stubServicesWithWait(log, func(ctx context.Context, ms int) error {
		t := time.NewTimer(time.Duration(ms) * time.Millisecond)
		defer t.Stop()
		select {
		case <-t.C:
			return nil
		case <-ctx.Done():
			return context.Cause(ctx)
		}
	})
}

func stubServicesWithWait(log func(string), wait func(context.Context, int) error) api.Services {
	real := api.NewServices(api.NewMemoryStore(), api.Env{}, nil)
	stored := map[[2]string]int{
		{"888888888", api.EquiGavin}: 690,
		{"888888888", api.GavUnion}:  710,
		{"888888888", api.Gavperian}: 640,
	}
	bureauDelay := map[string]int{api.EquiGavin: 100, api.GavUnion: 200, api.Gavperian: 300}
	bureauScore := map[string]int{api.EquiGavin: 720, api.GavUnion: 580, api.Gavperian: 650}
	bureauFail := map[string][]string{
		"999999999": {api.EquiGavin, api.GavUnion},
		"999999998": {api.Gavperian},
	}

	svc := api.Services{Log: log}
	svc.VerifyCredentials = func(ctx context.Context, event map[string]any) (api.UserCredential, error) {
		if err := wait(ctx, 20); err != nil {
			return api.UserCredential{}, err
		}
		return real.VerifyCredentials(ctx, event)
	}
	svc.DetermineMiddleScore = func(ctx context.Context, scores []int) (int, error) {
		if err := wait(ctx, 40); err != nil {
			return 0, err
		}
		for _, s := range scores {
			if s == 555 {
				return 0, errors.New("score service unavailable")
			}
		}
		return real.DetermineMiddleScore(ctx, scores)
	}
	svc.CheckReportsTable = func(ctx context.Context, q api.BureauQuery) (*api.CreditReport, error) {
		if err := wait(ctx, 40); err != nil {
			return nil, err
		}
		if q.SSN == "777777777" {
			return nil, errors.New("reports table unavailable")
		}
		if score, ok := stored[[2]string{q.SSN, q.BureauName}]; ok {
			return &api.CreditReport{SSN: q.SSN, BureauName: q.BureauName, CreditScore: score}, nil
		}
		return nil, nil
	}
	svc.CheckBureau = func(ctx context.Context, q api.BureauQuery) (int, error) {
		if err := wait(ctx, bureauDelay[q.BureauName]); err != nil {
			return 0, err
		}
		for _, failing := range bureauFail[q.SSN] {
			if failing == q.BureauName {
				return 0, errors.New(q.BureauName + " unavailable")
			}
		}
		switch q.SSN {
		case "666666666":
			return 666, nil
		case "555555555":
			return 555, nil
		}
		return bureauScore[q.BureauName], nil
	}
	svc.GenerateInterestRate = func(ctx context.Context, creditScore int) (float64, error) {
		if err := wait(ctx, 60); err != nil {
			return 0, err
		}
		if creditScore == 666 {
			return 0, errors.New("rate service unavailable")
		}
		switch {
		case creditScore > 700:
			return 3.5, nil
		case creditScore > 600:
			return 5, nil
		}
		return 200, nil
	}
	svc.SaveCreditReport = func(api.CreditReport) error { return nil }
	svc.SaveCreditProfile = func(api.CreditProfile) error { return nil }
	return svc
}

// traceMachine replaces asynchronous service calls with synchronous timer callbacks.
// Each callback uses the same service result and sends the invocation's completion event.
func traceMachine(clock xs.Clock) *xs.StateMachine[api.CreditProfile] {
	svc := stubServicesWithWait(nil, func(context.Context, int) error { return nil })
	service := func(delay func(any) int, call func(any) (any, error)) xs.ActorLogic {
		return xs.FromCallback(func(a xs.CallbackArgs) func() {
			timer := clock.SetTimeout(func() {
				output, err := call(a.Input)
				if err != nil {
					a.SendBack(xs.ErrorActorEvent{ActorID: a.Self.ID(), Error: err})
				} else {
					a.SendBack(xs.DoneActorEvent{ActorID: a.Self.ID(), Output: output})
				}
			}, time.Duration(delay(a.Input))*time.Millisecond)
			return func() { clock.ClearTimeout(timer) }
		})
	}
	fixed := func(ms int) func(any) int { return func(any) int { return ms } }
	ctx := context.Background()
	return api.Machine(svc).Provide(xs.Implementations{Actors: map[string]xs.ActorLogic{
		"verifyCredentials": service(fixed(20), func(input any) (any, error) { return svc.VerifyCredentials(ctx, input.(map[string]any)) }),
		"checkReportsTable": service(fixed(40), func(input any) (any, error) { return svc.CheckReportsTable(ctx, input.(api.BureauQuery)) }),
		"checkBureau": service(func(input any) int {
			return map[string]int{api.EquiGavin: 100, api.GavUnion: 200, api.Gavperian: 300}[input.(api.BureauQuery).BureauName]
		}, func(input any) (any, error) { return svc.CheckBureau(ctx, input.(api.BureauQuery)) }),
		"determineMiddleScore":  service(fixed(40), func(input any) (any, error) { return svc.DetermineMiddleScore(ctx, input.([]int)) }),
		"generateInterestRates": service(fixed(60), func(input any) (any, error) { return svc.GenerateInterestRate(ctx, input.(int)) }),
	}})
}
