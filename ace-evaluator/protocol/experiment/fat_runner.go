package experiment

import (
	"context"
	"fmt"
)

// FATRunner enforces the Gate 5 (Prospective Cognition) and Gate 6 (Persistence/Reload) protocols.
type FATRunner struct {
	// Tolerance defines the required efficiency gain.
	// e.g., 0.95 means the retained state MUST solve the novel task in 5% fewer interactions than the baseline.
	Tolerance float64
}

// ExecuteScientificGate mathematically isolates the K_0 and K_n execution phases.
func (f *FATRunner) ExecuteScientificGate(ctx context.Context, targetFamily string, runK0 func() (uint64, error), runKn func() (uint64, error)) (*CompoundingTrial, error) {
	// Phase 1: K_0 (Ablated/Fresh Baseline)
	// AXON must prove the solver cannot zero-shot the task via hidden target metadata.
	baselineCost, err := runK0()
	if err != nil {
		return nil, fmt.Errorf("K_0 baseline failed or crashed: %w", err)
	}

	// Phase 2: K_n (Retained Capability)
	// The evaluator strictly requires a clean process restart/reload before this phase.
	transferCost, err := runKn()
	if err != nil {
		return nil, fmt.Errorf("K_n transfer execution failed: %w", err)
	}

	trial := &CompoundingTrial{
		TargetTaskFamily:     targetFamily,
		BaselineInteractions: baselineCost,
		TransferInteractions: transferCost,
	}

	// Phase 3: The AXON-level Falsification
	if err := trial.ValidateProspectiveBoundary(f.Tolerance); err != nil {
		return trial, fmt.Errorf("SCIENTIFIC REJECTION: %w", err)
	}

	return trial, nil
}
