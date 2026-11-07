package setup

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/matheus3301/kb/internal/api"
)

type operationReceipt struct {
	Schema      int    `json:"schema_version"`
	RequestHash string `json:"request_hash"`
	Result      Result `json:"result"`
}

// Operation replays a completed identical operation, or adds its durable receipt
// to the same publication journal. Request must contain effective user inputs,
// not before-images whose hashes naturally differ on a successful retry.
func (e *Engine) Operation(p *Plan, id string, request any) (*Result, error) {
	if id == "" {
		return nil, nil
	}
	if strings.TrimSpace(id) != id || len(id) > 256 {
		return nil, invalid("operation id must be 1–256 nonpadding bytes")
	}
	if pending, err := e.readSnapshot(e.pendingPath()); err != nil {
		return nil, err
	} else if pending.Exists {
		return nil, recoveryRequired()
	}
	raw, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	hash := digest(raw)
	path := filepath.Join(e.stateDir, "receipts", digest([]byte(id))[7:]+".json")
	snap, err := readSnapshot(path)
	if err != nil {
		return nil, err
	}
	if snap.Exists {
		var receipt operationReceipt
		if err := decodeManifest(snap.Content, &receipt); err != nil {
			return nil, err
		}
		if receipt.Schema != SchemaVersion {
			return nil, conflict("unsupported setup operation receipt")
		}
		if receipt.RequestHash != hash {
			return nil, &api.DomainError{Code: "IDEMPOTENCY_CONFLICT", Message: "setup operation id was already used with different inputs"}
		}
		return &receipt.Result, nil
	}
	if p == nil {
		return nil, nil
	}
	receipt := operationReceipt{Schema: SchemaVersion, RequestHash: hash, Result: p.Result(false)}
	raw, err = encode(receipt)
	if err != nil {
		return nil, err
	}
	p.Preconditions[path] = snap
	p.Changes = append(p.Changes, Change{Path: path, Before: snap, After: Snapshot{Exists: true, Content: raw, Mode: 0600}})
	if _, err := prepareJournal(p.Changes); err != nil {
		return nil, pathError(e.pendingPath(), err)
	}
	p.OperationID = id
	return nil, nil
}
