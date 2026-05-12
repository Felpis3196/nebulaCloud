// Package audit owns the platform's audit trail. Other modules call
// Recorder.Record to append a row to `audit_logs`. The package is small on
// purpose: it has no domain logic of its own — it is a port + adapter.
package audit

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nebulacloud/nebula/internal/platform/httpx"
	"github.com/nebulacloud/nebula/internal/platform/logger"
)

// Recorder appends audit entries to Postgres.
type Recorder struct {
	pool *pgxpool.Pool
}

// NewRecorder constructs a Recorder.
func NewRecorder(pool *pgxpool.Pool) *Recorder { return &Recorder{pool: pool} }

// Record writes an audit entry.
//
// Best-effort: failures are logged at warn level but never bubble up to the
// caller — losing an audit row must not also fail the user-facing operation.
func (r *Recorder) Record(ctx context.Context, action string, actorID *uuid.UUID, metadata map[string]any) {
	if r == nil || r.pool == nil {
		return
	}
	correlationID := httpx.CorrelationIDFromContext(ctx)

	var meta []byte
	if len(metadata) > 0 {
		if b, err := json.Marshal(metadata); err == nil {
			meta = b
		}
	}
	if meta == nil {
		meta = []byte(`{}`)
	}

	const q = `
		INSERT INTO audit_logs (actor_id, correlation_id, action, metadata)
		VALUES ($1, NULLIF($2,''), $3, $4::jsonb)
	`
	if _, err := r.pool.Exec(ctx, q, actorID, correlationID, action, meta); err != nil {
		logger.FromContext(ctx).Warn("audit.record.failed", "action", action, "error", err)
	}
}
