package schema

import (
	"encoding/json"
	"fmt"
)

// ActionContract ensures the capability commits to a specific, typed intervention
type ActionContract struct {
	EnvelopeHash string          `json:"envelope_hash"` // Binds to TaskEnvelope.CanonicalHash
	ActionName   string          `json:"action_name"`
	Payload      json.RawMessage `json:"payload"` // Strict task-agnostic payload
}

// VerifyReceipt prevents the solver from leaking target info back or executing out-of-context
func (a *ActionContract) VerifyReceipt(expectedHash string) error {
	if a.EnvelopeHash != expectedHash {
		return fmt.Errorf("cryptographic receipt mismatch: capability claimed %s, expected %s", a.EnvelopeHash, expectedHash)
	}
	if a.ActionName == "" {
		return fmt.Errorf("action contract violation: missing action name")
	}
	return nil
}
