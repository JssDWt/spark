//go:build lightspark

package handler

import (
	"testing"

	"github.com/lightsparkdev/spark/so"
	"github.com/stretchr/testify/require"
)

func TestDepositCleanupRejectsNilRequest(t *testing.T) {
	handler := NewSspRequestHandler(&so.Config{})

	err := handler.DepositCleanup(t.Context(), nil)

	require.ErrorContains(t, err, "request is required")
}
