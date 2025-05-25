package internal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSandboxMessage(t *testing.T) {
	var msg = new(SandboxMessage)

	msg.Command = `testFunc`
	msg.Args = []any{`hello`, true}

	require.Equal(t, `testFunc(string, bool) (any, error)`, msg.String())
}
