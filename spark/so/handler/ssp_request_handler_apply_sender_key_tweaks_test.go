//go:build lightspark

package handler

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/lightsparkdev/spark/common/btcnetwork"
	"github.com/lightsparkdev/spark/common/keys"
	pb "github.com/lightsparkdev/spark/proto/spark"
	pbssp "github.com/lightsparkdev/spark/proto/spark_ssp_internal"
	"github.com/lightsparkdev/spark/so"
	"github.com/lightsparkdev/spark/so/db"
	"github.com/lightsparkdev/spark/so/ent"
	st "github.com/lightsparkdev/spark/so/ent/schema/schematype"
	enttransfer "github.com/lightsparkdev/spark/so/ent/transfer"
	"google.golang.org/protobuf/proto"
)

func TestApplySenderKeyTweaks_RecoversApplyingSenderKeyTweak(t *testing.T) {
	ctx, _ := db.ConnectToTestPostgres(t)
	dbTx, err := ent.GetDbFromContext(ctx)
	require.NoError(t, err)

	rng := rand.NewChaCha8([32]byte{9})
	senderPub := keys.MustGeneratePrivateKeyFromRand(rng).Public()
	receiverPub := keys.MustGeneratePrivateKeyFromRand(rng).Public()

	leaf := createDbLeaf(t, ctx, true)
	originalOwnerSigningPubkey := leaf.node.OwnerSigningPubkey

	transfer, err := dbTx.Transfer.Create().
		SetNetwork(btcnetwork.Regtest).
		SetType(st.TransferTypeTransfer).
		SetStatus(st.TransferStatusApplyingSenderKeyTweak).
		SetExpiryTime(time.Now().Add(24 * time.Hour)).
		SetTotalValue(1000).
		SetSenderIdentityPubkey(senderPub).
		SetReceiverIdentityPubkey(receiverPub).
		Save(ctx)
	require.NoError(t, err)

	_, err = dbTx.TransferLeaf.Create().
		SetTransfer(transfer).
		SetLeaf(leaf.node).
		SetKeyTweak(mustMarshalSimpleSendLeafKeyTweak(t, rng, leaf.node.ID.String())).
		SetPreviousRefundTx(leaf.node.RawRefundTx).
		SetIntermediateRefundTx(leaf.node.RawRefundTx).
		Save(ctx)
	require.NoError(t, err)

	sspHandler := NewSspRequestHandler(&so.Config{Identifier: "test-operator"})
	resp, err := sspHandler.ApplySenderKeyTweaks(ctx, &pbssp.ApplySenderKeyTweaksRequest{
		TransferIds: []string{transfer.ID.String()},
	})
	require.NoError(t, err)
	require.Equal(t, []string{transfer.ID.String()}, resp.UpdatedTransferIds)

	readDb, err := ent.GetDbFromContext(ctx)
	require.NoError(t, err)

	updatedTransfer, err := readDb.Transfer.Query().
		Where(enttransfer.IDEQ(transfer.ID)).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, st.TransferStatusSenderKeyTweaked, updatedTransfer.Status)

	updatedLeafs, err := updatedTransfer.QueryTransferLeaves().All(ctx)
	require.NoError(t, err)
	require.Len(t, updatedLeafs, 1)
	require.Nil(t, updatedLeafs[0].KeyTweak)
	require.NotNil(t, updatedLeafs[0].SecretCipher)
	require.NotNil(t, updatedLeafs[0].Signature)

	updatedNode, err := updatedLeafs[0].QueryLeaf().Only(ctx)
	require.NoError(t, err)
	require.NotEqual(t, originalOwnerSigningPubkey, updatedNode.OwnerSigningPubkey)
}

func mustMarshalSimpleSendLeafKeyTweak(t *testing.T, rng *rand.ChaCha8, leafID string) []byte {
	t.Helper()

	sharePriv := keys.MustGeneratePrivateKeyFromRand(rng)
	pubShareTweak := keys.MustGeneratePrivateKeyFromRand(rng).Public()

	keyTweakBinary, err := proto.Marshal(&pb.SendLeafKeyTweak{
		LeafId: leafID,
		SecretShareTweak: &pb.SecretShare{
			SecretShare: sharePriv.Serialize(),
			Proofs:      [][]byte{sharePriv.Public().Serialize()},
		},
		PubkeySharesTweak: map[string][]byte{
			"1": pubShareTweak.Serialize(),
		},
		SecretCipher: []byte("valid-secret-cipher"),
		Signature:    []byte("valid-signature"),
	})
	require.NoError(t, err)
	return keyTweakBinary
}
