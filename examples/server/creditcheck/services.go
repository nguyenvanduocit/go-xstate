package creditcheck

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

// Services is what the machine's actors and actions call: the functions of
// services/machineLogicService.ts. NewServices builds the real ones; tests inject stubs.
type Services struct {
	// VerifyCredentials validates the Submit event (verifyCredentials).
	VerifyCredentials func(ctx context.Context, event map[string]any) (UserCredential, error)
	// DetermineMiddleScore returns the middle of the three scores (determineMiddleScore).
	DetermineMiddleScore func(ctx context.Context, scores []int) (int, error)
	// CheckReportsTable looks up a stored report; nil when there is none (checkReportsTable).
	CheckReportsTable func(ctx context.Context, q BureauQuery) (*CreditReport, error)
	// CheckBureau asks a credit bureau for a score (checkBureauService).
	CheckBureau func(ctx context.Context, q BureauQuery) (int, error)
	// GenerateInterestRate returns the interest rate for a score (generateInterestRate).
	GenerateInterestRate func(ctx context.Context, creditScore int) (float64, error)
	// SaveCreditReport upserts a report by ssn and bureau (saveCreditReport).
	SaveCreditReport func(report CreditReport) error
	// SaveCreditProfile upserts a profile by SSN (saveCreditProfile).
	SaveCreditProfile func(profile CreditProfile) error
	// Log receives what the JS code prints with console.log, one formatted line per call. May be nil.
	Log func(line string)
}

func (s Services) log(args ...any) {
	if s.Log != nil {
		s.Log(consoleFormat(args...))
	}
}

// Env holds the sources of time and randomness behind the bureau and rate simulations.
type Env struct {
	Sleep  func(ctx context.Context, d time.Duration) error // default: wait d or until ctx is done
	Random func() float64                                   // default: math/rand/v2 Float64
}

func (e Env) withDefaults() Env {
	if e.Sleep == nil {
		e.Sleep = sleep
	}
	if e.Random == nil {
		e.Random = rand.Float64
	}
	return e
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}

// rangeInt mirrors range({min, max}).
func (e Env) rangeInt(min, max int) int {
	return int(math.Floor(e.Random()*float64(max-min) + float64(min)))
}

// NewServices returns the real services backed by store, mirroring machineLogicService.ts.
func NewServices(store Store, env Env, log func(line string)) Services {
	env = env.withDefaults()
	s := Services{Log: log}
	s.VerifyCredentials = func(_ context.Context, event map[string]any) (UserCredential, error) {
		s.log("Verifying Credentials...")
		c, err := verifyCredentials(event)
		if err != nil {
			err = errors.New("Invalid Credentials. Details: " + err.Error())
			s.log(err.Error())
		}
		return c, err
	}
	s.DetermineMiddleScore = func(_ context.Context, scores []int) (int, error) {
		return determineMiddleScore(scores), nil
	}
	s.CheckReportsTable = func(ctx context.Context, q BureauQuery) (*CreditReport, error) {
		s.log("Checking for an existing report....")
		report, err := store.FindCreditReport(ctx, q.SSN, q.BureauName)
		if err != nil {
			s.log("Error checking reports table", err)
		}
		return report, err
	}
	s.CheckBureau = func(ctx context.Context, q BureauQuery) (int, error) {
		switch q.BureauName {
		case EquiGavin, GavUnion, Gavperian:
			if err := env.Sleep(ctx, time.Duration(env.rangeInt(1000, 10000))*time.Millisecond); err != nil {
				return 0, err
			}
			return env.rangeInt(300, 850), nil
		}
		return 0, nil // JS returns undefined; the machine reads it as 0
	}
	s.GenerateInterestRate = func(ctx context.Context, creditScore int) (float64, error) {
		if err := env.Sleep(ctx, time.Duration(env.rangeInt(1000, 10000))*time.Millisecond); err != nil {
			return 0, err
		}
		return interestRate(creditScore), nil
	}
	s.SaveCreditReport = func(report CreditReport) error {
		err := store.SaveCreditReport(context.Background(), report)
		if err != nil {
			s.log("Error saving credit report", err)
		}
		return err
	}
	s.SaveCreditProfile = func(profile CreditProfile) error {
		err := store.SaveCreditProfile(context.Background(), profile)
		if err != nil {
			s.log("Error saving credit profile", err)
		}
		return err
	}
	return s
}

func interestRate(creditScore int) float64 {
	switch {
	case creditScore > 700:
		return 3.5
	case creditScore > 600:
		return 5
	default:
		return 200
	}
}

// determineMiddleScore mirrors `scores.sort(); return scores[1]`: Array.prototype.sort
// without a comparator orders numbers by their decimal string.
func determineMiddleScore(scores []int) int {
	sorted := append([]int(nil), scores...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return utf16Less(strconv.Itoa(sorted[i]), strconv.Itoa(sorted[j]))
	})
	if len(sorted) < 2 {
		return 0
	}
	return sorted[1]
}

func utf16Less(a, b string) bool {
	return compareUTF16(utf16.Encode([]rune(a)), utf16.Encode([]rune(b))) < 0
}

func compareUTF16(a, b []uint16) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return int(a[i]) - int(b[i])
		}
	}
	return len(a) - len(b)
}

// verifyCredentials mirrors userCredentialSchema: firstName and lastName are strings of
// 1..255 characters, SSN is a string of exactly 9. Lengths count UTF-16 code units like JS.
// The error names the first invalid field; zod's own message format is not reproduced.
func verifyCredentials(event map[string]any) (UserCredential, error) {
	field := func(key string, min, max int) (string, error) {
		s, ok := event[key].(string)
		if !ok {
			return "", fmt.Errorf("invalid %s", key)
		}
		if n := len(utf16.Encode([]rune(s))); n < min || n > max {
			return "", fmt.Errorf("invalid %s", key)
		}
		return s, nil
	}
	var c UserCredential
	var err error
	if c.FirstName, err = field("firstName", 1, 255); err != nil {
		return UserCredential{}, err
	}
	if c.LastName, err = field("lastName", 1, 255); err != nil {
		return UserCredential{}, err
	}
	if c.SSN, err = field("SSN", 9, 9); err != nil {
		return UserCredential{}, err
	}
	return c, nil
}

// consoleFormat renders console.log arguments the way Node prints the values this
// example logs: strings as is, numbers, error messages, and number arrays as `[ 1, 2 ]`.
func consoleFormat(args ...any) string {
	parts := make([]string, len(args))
	for i, a := range args {
		switch v := a.(type) {
		case string:
			parts[i] = v
		case int:
			parts[i] = strconv.Itoa(v)
		case float64:
			parts[i] = strconv.FormatFloat(v, 'f', -1, 64)
		case []float64:
			nums := make([]string, len(v))
			for j, n := range v {
				nums[j] = strconv.FormatFloat(n, 'f', -1, 64)
			}
			if len(nums) == 0 {
				parts[i] = "[]"
			} else {
				parts[i] = "[ " + strings.Join(nums, ", ") + " ]"
			}
		case error:
			parts[i] = v.Error()
		default:
			parts[i] = fmt.Sprint(v)
		}
	}
	return strings.Join(parts, " ")
}
