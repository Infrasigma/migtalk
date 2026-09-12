package identity

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Genesis represents the hidden ground truth of the environment.
// The capability NEVER sees this directly. It is strictly for the Evaluator.
type Genesis struct {
	WorldSeed      string          `json:"world_seed"`
	DynamicsSchema json.RawMessage `json:"dynamics_schema"`
	GoalDefinition json.RawMessage `json:"goal_definition"`
}

// ChallengeIdentity is the cryptographically bound ID of the specific task instance.
type ChallengeIdentity struct {
	GenesisHash string `json:"genesis_hash"`
}

// Bind generates the immutable hash of the genesis state.
// This proves the world wasn't altered mid-evaluation.
func Bind(g *Genesis) (*ChallengeIdentity, error) {
	b, err := json.Marshal(g)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(b)
	return &ChallengeIdentity{
		GenesisHash: hex.EncodeToString(hash[:]),
	}, nil
}

// TrajectoryLink binds an intervention to the previous state to generate the next state hash.
// This proves the causal chain: State_N = Hash(State_{N-1} + Intervention_N + Observation_{N+1})
func TrajectoryLink(previousStateHash string, intervention json.RawMessage, nextObservation json.RawMessage) string {
	payload := fmt.Sprintf("%s|%s|%s", previousStateHash, string(intervention), string(nextObservation))
	hash := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(hash[:])
}
