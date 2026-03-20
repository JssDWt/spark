//go:build lightspark

package handler

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lightsparkdev/spark/common/btcnetwork"
	"github.com/lightsparkdev/spark/common/keys"
	sparkProto "github.com/lightsparkdev/spark/proto/spark"
	pbinternal "github.com/lightsparkdev/spark/proto/spark_internal"
	"github.com/lightsparkdev/spark/so/db"
	"github.com/lightsparkdev/spark/so/ent"
	st "github.com/lightsparkdev/spark/so/ent/schema/schematype"
	sparktesting "github.com/lightsparkdev/spark/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeliverSenderKeyTweak_MissingKeyTweakForLeaf(t *testing.T) {
	ctx, dbCtx := db.ConnectToTestPostgres(t)
	rng := rand.NewChaCha8([32]byte{99})

	cfg := sparktesting.TestConfig(t)

	senderIdentityPrivKey := keys.MustGeneratePrivateKeyFromRand(rng)
	receiverIdentityPrivKey := keys.MustGeneratePrivateKeyFromRand(rng)
	ownerIdentityPrivKey := senderIdentityPrivKey

	// Create two signing keyshares, trees, and leaves.
	var leaves [2]*ent.TreeNode
	for i := range leaves {
		keysharePrivKey := keys.MustGeneratePrivateKeyFromRand(rng)
		publicSharePrivKey := keys.MustGeneratePrivateKeyFromRand(rng)
		signingKeyshare, err := dbCtx.Client.SigningKeyshare.Create().
			SetStatus(st.KeyshareStatusAvailable).
			SetSecretShare(keysharePrivKey).
			SetPublicShares(map[string]keys.Public{"test": publicSharePrivKey.Public()}).
			SetPublicKey(keysharePrivKey.Public()).
			SetMinSigners(2).
			SetCoordinatorIndex(0).
			Save(ctx)
		require.NoError(t, err)

		baseTxid := st.NewRandomTxIDForTesting(t)
		tree, err := dbCtx.Client.Tree.Create().
			SetStatus(st.TreeStatusAvailable).
			SetNetwork(btcnetwork.Regtest).
			SetOwnerIdentityPubkey(ownerIdentityPrivKey.Public()).
			SetBaseTxid(baseTxid).
			SetVout(0).
			Save(ctx)
		require.NoError(t, err)

		verifyingPrivKey := keys.MustGeneratePrivateKeyFromRand(rng)
		ownerSigningPrivKey := keys.MustGeneratePrivateKeyFromRand(rng)
		leaf, err := dbCtx.Client.TreeNode.Create().
			SetStatus(st.TreeNodeStatusAvailable).
			SetTree(tree).
			SetNetwork(tree.Network).
			SetSigningKeyshare(signingKeyshare).
			SetValue(1000).
			SetVerifyingPubkey(verifyingPrivKey.Public()).
			SetOwnerIdentityPubkey(ownerIdentityPrivKey.Public()).
			SetOwnerSigningPubkey(ownerSigningPrivKey.Public()).
			SetRawTx(createTestTxBytes(t, int64(3000+i))).
			SetRawRefundTx(createTestTxBytes(t, int64(3100+i))).
			SetVout(0).
			Save(ctx)
		require.NoError(t, err)
		leaves[i] = leaf
	}

	// Create a transfer in SenderInitiated with both leaves.
	transferID := uuid.New()
	transfer, err := dbCtx.Client.Transfer.Create().
		SetID(transferID).
		SetNetwork(btcnetwork.Regtest).
		SetStatus(st.TransferStatusSenderInitiated).
		SetType(st.TransferTypeTransfer).
		SetSenderIdentityPubkey(senderIdentityPrivKey.Public()).
		SetReceiverIdentityPubkey(receiverIdentityPrivKey.Public()).
		SetTotalValue(2000).
		SetExpiryTime(time.Now().Add(24 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	for _, leaf := range leaves {
		_, err = dbCtx.Client.TransferLeaf.Create().
			SetTransfer(transfer).
			SetLeaf(leaf).
			SetPreviousRefundTx(createTestTxBytes(t, 4000)).
			SetIntermediateRefundTx(createTestTxBytes(t, 4001)).
			Save(ctx)
		require.NoError(t, err)
	}

	// Build a transfer package with key tweaks for ONLY the first leaf (not the second).
	keyTweakPackage, userSignature := createMockKeyTweakPackage(t, cfg, rng, leaves[0].ID, ownerIdentityPrivKey, transferID)

	req := &pbinternal.DeliverSenderKeyTweakRequest{
		TransferId:              transferID.String(),
		SenderIdentityPublicKey: senderIdentityPrivKey.Public().Serialize(),
		TransferPackage: &sparkProto.TransferPackage{
			LeavesToSend: []*sparkProto.UserSignedTxSigningJob{
				{LeafId: leaves[0].ID.String(), RawTx: leaves[0].RawRefundTx},
				{LeafId: leaves[1].ID.String(), RawTx: leaves[1].RawRefundTx},
			},
			KeyTweakPackage: keyTweakPackage,
			UserSignature:   userSignature,
		},
	}

	handler := NewInternalTransferHandler(cfg)
	err = handler.DeliverSenderKeyTweak(ctx, req)

	// Should fail because leaf[1] has no key tweak in the encrypted package.
	require.Error(t, err)
	assert.Contains(t, err.Error(), "key tweak not found for leaf")
	assert.Contains(t, err.Error(), leaves[1].ID.String())

	// Verify transfer status was NOT updated to SenderKeyTweakPending.
	updatedTransfer, err := dbCtx.Client.Transfer.Get(ctx, transferID)
	require.NoError(t, err)
	assert.Equal(t, st.TransferStatusSenderInitiated, updatedTransfer.Status,
		"transfer must remain SenderInitiated when DeliverSenderKeyTweak fails")
}
