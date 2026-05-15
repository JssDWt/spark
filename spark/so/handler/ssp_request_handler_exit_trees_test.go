//go:build lightspark

package handler

import (
	"testing"

	"github.com/lightsparkdev/spark/common/keys"
	pb "github.com/lightsparkdev/spark/proto/spark"
	pbssp "github.com/lightsparkdev/spark/proto/spark_ssp_internal"
	"github.com/lightsparkdev/spark/so"
	"github.com/stretchr/testify/require"
)

func TestExitTreesRejectsNilRequest(t *testing.T) {
	handler := NewSspRequestHandler(&so.Config{})

	resp, err := handler.ExitTrees(t.Context(), nil)

	require.Nil(t, resp)
	require.ErrorContains(t, err, "request is required")
}

func TestExitTreesRejectsNilExitingTreeBeforeDBLookup(t *testing.T) {
	cfg := setUpTestConfigWithRegtestNoAuthz(t)
	handler := NewSspRequestHandler(cfg)
	ownerIdentity := keys.GeneratePrivateKey().Public()

	resp, err := handler.ExitTrees(t.Context(), &pbssp.ExitTreesRequest{
		OwnerIdentityPublicKey: ownerIdentity.Serialize(),
		ExitingTrees:           []*pb.ExitingTree{nil},
	})

	require.Nil(t, resp)
	require.ErrorContains(t, err, "exiting_trees[0] is required")
}
