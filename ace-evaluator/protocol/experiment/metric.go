package experiment

import (
	"errors"
	"fmt"
)

// CompoundingTrial tracks the exact resource cost of capability acquisition
type CompoundingTrial struct {
	TargetTaskFamily     string
	BaselineInteractions uint64 // C(T_{n+1} | K_0) - Cost with fresh/deleted state
	TransferInteractions uint64 // C(T_{n+1} | K_n) - Cost with retained structure
}

// CalculateRn computes the prospective transfer ratio.
// R_n < 1.0 indicates genuine structural transfer (lowered future acquisition cost).
// R_n >= 1.0 indicates rigid memorization, overfitting, or negative transfer.
func (t *CompoundingTrial) CalculateRn() (float64, error) {
	if t.BaselineInteractions == 0 {
		return 0, errors.New("scientific invalidation: baseline solved in 0 interactions (target leakage suspected)")
	}
	return float64(t.TransferInteractions) / float64(t.BaselineInteractions), nil
}

// ValidateProspectiveBoundary ensures the retained state isn't just a cached solver
// by enforcing a strict interaction limit based on the K_0 baseline.
func (t *CompoundingTrial) ValidateProspectiveBoundary(maxAllowedRn float64) error {
	rn, err := t.CalculateRn()
	if err != nil {
		return err
	}
	if rn > maxAllowedRn {
		return fmt.Errorf("prospective cognition failed: R_n (%.2f) exceeds boundary (%.2f)", rn, maxAllowedRn)
	}
	return nil
}
