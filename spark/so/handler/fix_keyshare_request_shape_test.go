//go:build lightspark

package handler

import (
	"testing"

	"github.com/lightsparkdev/spark/so"
	"github.com/stretchr/testify/require"
)

func TestFixKeyshareRejectsNilRequests(t *testing.T) {
	handler := NewFixKeyshareHandler(&so.Config{})

	err := handler.FixKeyshare(t.Context(), nil)
	require.ErrorContains(t, err, "fix keyshare request is required")

	round1Resp, err := handler.Round1(t.Context(), nil)
	require.Nil(t, round1Resp)
	require.ErrorContains(t, err, "fix keyshare round 1 request is required")

	round2Resp, err := handler.Round2(t.Context(), nil)
	require.Nil(t, round2Resp)
	require.ErrorContains(t, err, "fix keyshare round 2 request is required")
}

func TestSspFixKeyshareRejectsNilRequest(t *testing.T) {
	handler := NewSspRequestHandler(&so.Config{})

	err := handler.FixKeyshare(t.Context(), nil)

	require.ErrorContains(t, err, "request is required")
}
