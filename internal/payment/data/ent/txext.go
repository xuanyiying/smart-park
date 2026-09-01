package ent

import "entgo.io/ent/dialect/sql"

// UnderlyingSQLTx exposes the raw *sql.Tx that backs an Ent transaction.
//
// Ent keeps its transaction driver private, which is correct for its own
// builders but leaves no way for auxiliary writers to join the transaction Ent
// manages. The transactional outbox needs exactly that: an outbox row must
// commit (or roll back) together with the business change in the same
// transaction, otherwise the hand-off it is supposed to make atomic stays racy.
// Returns nil when the transaction is not backed by dialect/sql (sqlmock or
// another dialect), in which case callers fall back to their own connection.
func (t *Tx) UnderlyingSQLTx() *sql.Tx {
	drv, ok := t.config.driver.(*txDriver)
	if !ok {
		return nil
	}
	dialectTx, ok := drv.tx.(*sql.Tx)
	if !ok {
		return nil
	}
	return dialectTx
}
