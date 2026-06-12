//go:build lightspark

package grpctest

import (
	"crypto/rand"
	"crypto/sha256"
	"testing"

	"github.com/lightsparkdev/spark/common/keys"
	sparkpb "github.com/lightsparkdev/spark/proto/spark"
	"github.com/lightsparkdev/spark/testing/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestProvidePreimage_RejectsSenderInitiatedReceiveSwap is the end-to-end
// regression for the "preimage revealed without key-tweak commit" theft:
//
// A malicious SSP creates a HODL receive preimage swap but never delivers the
// transfer package, so the swap stays at SENDER_INITIATED (sender key tweaks
// were never staged on any SO). If the receiver then reveals the preimage, the
// public ProvidePreimage short-circuit must NOT store it — otherwise the SSP
// reads it back via QueryPreimage, settles the inbound Lightning HTLC, and the
// receiver is left with an unclaimable transfer.
//
// With the fix, ProvidePreimage fails closed for SENDER_INITIATED, the sender
// cannot read a preimage, and the receiver has no claimable transfer that it
// paid for with a leaked secret.
func TestProvidePreimage_RejectsSenderInitiatedReceiveSwap(t *testing.T) {
	userConfig := wallet.NewTestWalletConfig(t)
	sspConfig := wallet.NewTestWalletConfig(t)

	amountSats := uint64(100)
	preimage, paymentHash := randomPreimageHash(t)
	defer cleanUp(t, userConfig, paymentHash)

	sspLeafPrivKey := keys.GeneratePrivateKey()
	nodeToSend, err := wallet.CreateNewTree(sspConfig, faucet, sspLeafPrivKey, 12_345)
	require.NoError(t, err)

	newLeafPrivKey := keys.GeneratePrivateKey()
	leaves := []wallet.LeafKeyTweak{{
		Leaf:              nodeToSend,
		SigningPrivKey:    sspLeafPrivKey,
		NewSigningPrivKey: newLeafPrivKey,
	}}

	// HODL receive swap with no preimage share and no delivered transfer
	// package: the resulting transfer stays at SENDER_INITIATED.
	response, err := wallet.SwapNodesForPreimage(
		t.Context(),
		sspConfig,
		leaves,
		userConfig.IdentityPublicKey(),
		paymentHash[:],
		nil,
		0,
		true,
		amountSats,
	)
	require.NoError(t, err)
	require.Empty(t, response.GetPreimage())
	require.Equal(t, sparkpb.TransferStatus_TRANSFER_STATUS_SENDER_INITIATED, response.GetTransfer().GetStatus())

	// The receiver reveals the preimage. The fixed handler must reject this
	// rather than persisting the secret for a non-committable transfer.
	_, err = wallet.ProvidePreimage(t.Context(), userConfig, preimage[:])
	require.Error(t, err)
	require.Equal(t, codes.FailedPrecondition, status.Code(err))

	// The sender (SSP) must not be able to read the preimage.
	leaked := queryPreimageAsSender(t, sspConfig, paymentHash[:], userConfig.IdentityPublicKey())
	assert.Empty(t, leaked, "sender must not be able to read a preimage for a non-committable transfer")

	// The receiver must have no claimable transfer for this swap.
	userToken, err := wallet.AuthenticateWithServer(t.Context(), userConfig)
	require.NoError(t, err)
	userCtx := wallet.ContextWithToken(t.Context(), userToken)
	pending, err := wallet.QueryPendingTransfers(userCtx, userConfig)
	require.NoError(t, err)
	for _, transfer := range pending.GetTransfers() {
		require.NotEqual(t, response.GetTransfer().GetId(), transfer.GetId(),
			"receiver must not have a claimable transfer after the preimage was withheld")
	}
}

func randomPreimageHash(t *testing.T) ([32]byte, [32]byte) {
	t.Helper()

	var preimage [32]byte
	_, err := rand.Read(preimage[:])
	require.NoError(t, err)

	return preimage, sha256.Sum256(preimage[:])
}

func queryPreimageAsSender(
	t *testing.T,
	senderConfig *wallet.TestWalletConfig,
	paymentHash []byte,
	receiverIdentityPubKey keys.Public,
) []byte {
	t.Helper()

	conn, err := senderConfig.NewCoordinatorGRPCConnection()
	require.NoError(t, err)
	defer conn.Close()

	token, err := wallet.AuthenticateWithConnection(t.Context(), senderConfig, conn)
	require.NoError(t, err)
	ctx := wallet.ContextWithToken(t.Context(), token)

	client := sparkpb.NewSparkServiceClient(conn)
	resp, err := client.QueryPreimage(ctx, &sparkpb.QueryPreimageRequest{
		PaymentHash:            paymentHash,
		ReceiverIdentityPubkey: receiverIdentityPubKey.Serialize(),
	})
	// A NotFound is itself proof the sender cannot read the preimage, so treat
	// it as an empty (passing) result rather than a test failure. Any other
	// error is an unexpected infrastructure problem and should surface.
	if status.Code(err) == codes.NotFound {
		return nil
	}
	require.NoError(t, err)

	return resp.GetPreimage()
}
