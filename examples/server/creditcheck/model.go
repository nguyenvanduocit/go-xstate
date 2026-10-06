// Package creditcheck ports references/xstate/examples/mongodb-credit-check-api: the credit check
// machine (machine.ts), its services (services/machineLogicService.ts), the durable actor
// helper (services/actorService.ts) and the HTTP API (index.ts).
package creditcheck

// CreditProfile is the machine context and the document saved by saveCreditProfile
// (models/creditProfile.ts). The JSON keys are the JS context keys.
type CreditProfile struct {
	SSN                 string    `json:"SSN" bson:"SSN"`
	FirstName           string    `json:"FirstName" bson:"FirstName"`
	LastName            string    `json:"LastName" bson:"LastName"`
	GavUnionScore       int       `json:"GavUnionScore" bson:"GavUnionScore"`
	EquiGavinScore      int       `json:"EquiGavinScore" bson:"EquiGavinScore"`
	GavperianScore      int       `json:"GavperianScore" bson:"GavperianScore"`
	ErrorMessage        string    `json:"ErrorMessage" bson:"ErrorMessage"`
	MiddleScore         int       `json:"MiddleScore" bson:"MiddleScore"`
	InterestRateOptions []float64 `json:"InterestRateOptions" bson:"InterestRateOptions"`
}

// CreditReport is a bureau score stored by saveCreditReport (models/creditReport.ts).
type CreditReport struct {
	SSN         string `json:"ssn" bson:"ssn"`
	BureauName  string `json:"bureauName" bson:"bureauName"`
	CreditScore int    `json:"creditScore" bson:"creditScore"`
}

// UserCredential is the validated Submit payload (userCredential in machineLogicService.ts).
type UserCredential struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	SSN       string `json:"SSN"`
}

// BureauQuery is the input of the checkBureau and checkReportsTable actors.
type BureauQuery struct {
	SSN        string `json:"ssn"`
	BureauName string `json:"bureauName"`
}

// Bureau names used by the machine.
const (
	EquiGavin = "EquiGavin"
	GavUnion  = "GavUnion"
	Gavperian = "Gavperian"
)
