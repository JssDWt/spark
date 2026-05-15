//go:build lightspark

package handler

import (
	"testing"

	"github.com/google/uuid"
	"github.com/lightsparkdev/spark/common/btcnetwork"
	"github.com/lightsparkdev/spark/common/keys"
	pb "github.com/lightsparkdev/spark/proto/spark"
	pbssp "github.com/lightsparkdev/spark/proto/spark_ssp_internal"
	"github.com/lightsparkdev/spark/so"
	"github.com/lightsparkdev/spark/so/db"
	"github.com/lightsparkdev/spark/so/ent"
	st "github.com/lightsparkdev/spark/so/ent/schema/schematype"
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

func TestValidateExitingTreeRejectsAlreadyExitedTreeForSigning(t *testing.T) {
	ctx, _ := db.ConnectToTestPostgres(t)
	dbTx, err := ent.GetDbFromContext(ctx)
	require.NoError(t, err)

	ownerIdentity := keys.GeneratePrivateKey().Public()
	tree, err := dbTx.Tree.Create().
		SetID(uuid.New()).
		SetNetwork(btcnetwork.Regtest).
		SetStatus(st.TreeStatusExited).
		SetBaseTxid(st.NewRandomTxIDForTesting(t)).
		SetVout(0).
		SetOwnerIdentityPubkey(ownerIdentity).
		Save(ctx)
	require.NoError(t, err)

	secret := keys.GeneratePrivateKey()
	ks, err := dbTx.SigningKeyshare.Create().
		SetID(uuid.New()).
		SetStatus(st.KeyshareStatusAvailable).
		SetSecretShare(secret).
		SetPublicShares(map[string]keys.Public{"1": secret.Public()}).
		SetPublicKey(secret.Public()).
		SetMinSigners(1).
		SetCoordinatorIndex(0).
		Save(ctx)
	require.NoError(t, err)

	_, err = dbTx.TreeNode.Create().
		SetID(uuid.New()).
		SetTree(tree).
		SetSigningKeyshare(ks).
		SetValue(testSourceValue).
		SetVerifyingPubkey(secret.Public()).
		SetOwnerIdentityPubkey(ownerIdentity).
		SetOwnerSigningPubkey(secret.Public()).
		SetVout(0).
		SetNetwork(btcnetwork.Regtest).
		SetStatus(st.TreeNodeStatusExited).
		SetRawTx(makeMinimalRawTx(t)).
		SetRawRefundTx(makeMinimalRawTx(t)).
		Save(ctx)
	require.NoError(t, err)

	handler := NewSspRequestHandler(&so.Config{})
	_, err = handler.validateExitingTree(ctx, tree.ID.String(), ownerIdentity)

	require.ErrorContains(t, err, "not eligible for exit signing")
}
