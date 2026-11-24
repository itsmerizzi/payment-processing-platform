// Package system has the production implementations of the ID and clock ports.
package system

import (
	"time"

	"github.com/google/uuid"
)

type UUIDv7 struct{}

func (UUIDv7) NewID() (string, error) {
	id, err := uuid.NewV7()
	return id.String(), err
}

type Clock struct{}

func (Clock) Now() time.Time { return time.Now().UTC().Truncate(time.Microsecond) } // Postgres precision, so a replay returns identical timestamps
