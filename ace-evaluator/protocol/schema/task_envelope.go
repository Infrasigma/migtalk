package schema

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// TaskEnvelope replaces opaque []byte with a strict schema
type TaskEnvelope struct {
	WorldID        string          `json:"world_id"`
	TaskID         string          `json:"task_id"`
	Observation    json.RawMessage `json:"observation"`
	AllowedActions []string        `json:"allowed_actions"`
}

// CanonicalHash generates the cryptographic binding for the task
func (t *TaskEnvelope) CanonicalHash() (string, error) {
	// standard json.Marshal serializes struct fields in definition order.
	b, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:]), nil
}
