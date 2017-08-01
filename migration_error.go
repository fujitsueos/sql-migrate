package migrate

import "fmt"

// TxError is returned when any error is encountered during a database
// transaction. It contains the relevant *Migration and notes it's Id in the
// Error function output.
type TxError struct {
	Migration *Migration
	Err       error
}

func newTxError(migration *PlannedMigration, err error) error {
	return &TxError{
		Migration: migration.Migration,
		Err:       err,
	}
}

func (e *TxError) Error() string {
	return fmt.Sprintf("%s handling %d", e.Err.Error(), e.Migration.Id)
}
