package buildinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestString_Defaults(t *testing.T) {
	Version = "N/A"
	Date = "N/A"
	Commit = "N/A"

	assert.Equal(t, "version=N/A, date=N/A, commit=N/A", String())
}

func TestString_CustomValues(t *testing.T) {
	origVersion, origDate, origCommit := Version, Date, Commit
	defer func() {
		Version, Date, Commit = origVersion, origDate, origCommit
	}()

	Version = "1.2.3"
	Date = "2025-01-15"
	Commit = "abc123"

	assert.Equal(t, "version=1.2.3, date=2025-01-15, commit=abc123", String())
}

func TestPrint(t *testing.T) {
	origVersion, origDate, origCommit := Version, Date, Commit
	defer func() {
		Version, Date, Commit = origVersion, origDate, origCommit
	}()

	Version = "v1.0.0"
	Date = "2025-06-01"
	Commit = "deadbeef"

	assert.NotPanics(t, func() {
		Print()
	})
}
