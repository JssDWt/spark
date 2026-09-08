package handler

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lightsparkdev/spark/common/btcnetwork"
	"github.com/lightsparkdev/spark/common/keys"
	pbgossip "github.com/lightsparkdev/spark/proto/gossip"
	pbinternal "github.com/lightsparkdev/spark/proto/spark_internal"
	pbsspsvc "github.com/lightsparkdev/spark/proto/spark_ssp"
	"github.com/lightsparkdev/spark/so"
	"github.com/lightsparkdev/spark/so/consensus"
	"github.com/lightsparkdev/spark/so/ent"
	"github.com/lightsparkdev/spark/so/ent/depositaddress"
	st "github.com/lightsparkdev/spark/so/ent/schema/schematype"
	entutxoswap "github.com/lightsparkdev/spark/so/ent/utxoswap"
	sparkerrors "github.com/lightsparkdev/spark/so/errors"
	"github.com/lightsparkdev/spark/so/helper"
	"github.com/lightsparkdev/spark/so/knobs"
	"github.com/lightsparkdev/spark/so/staticdeposit"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// ---------------------------------------------------------------------------
// reserveInstantStaticDepositCoordinatorFlow — coordinator side
// ---------------------------------------------------------------------------
//
// Coordinator side of phase one of the instant static deposit claim: it drives
// the 2PC engine, delegates the nested SSP→user credit transfer to the
// send-transfer coordinator flow, and builds the public
// ReserveInstantStaticDepositUtxoSwap response.

type reserveInstantStaticDepositCoordinatorFlow struct {
	*ReserveInstantStaticDepositFlowHandler

	req *pbinternal.ReserveInstantStaticDepositUtxoSwapRequest
	// transferCoord aggregates the nested transfer's leaf signatures, applies
	// the transfer commit on the coordinator, records partner attribution, and
	// builds the transfer response. Its embedded handler is this flow's
	// utxo-swap-typed SendTransferFlowHandler.
	transferCoord *sendTransferCoordinatorFlow

	// response is populated in BuildCommitPayload for the public handler to return.
	response *pbsspsvc.ReserveInstantStaticDepositUtxoSwapResponse
}

var _ consensus.CoordinatorFlow = (*reserveInstantStaticDepositCoordinatorFlow)(nil)

func (f *reserveInstantStaticDepositCoordinatorFlow) PrepareOp() proto.Message {
	return &pbinternal.ReserveInstantStaticDepositUtxoSwapPrepareRequest{
		OriginalRequest: f.req,
		// Every other SO validates the nested transfer package against the
		// proofs the coordinator decrypted from its own slice, so they travel
		// with the prepare.
		SenderKeyTweakProofs: f.transferCoord.senderKeyTweakProofs,
	}
}

// BuildCommitPayload delegates the nested transfer to the send-transfer
// coordinator flow (leaf-signature aggregation, coordinator-local transfer
// commit → SENDER_KEY_TWEAKED, partner attribution, transfer response) and
// builds the RPC response. There is no swap-side commit work: the reservation
// deliberately stays CREATED for the claim phase, and there is no spend-tx
// signature to aggregate at reserve time.
func (f *reserveInstantStaticDepositCoordinatorFlow) BuildCommitPayload(ctx context.Context, results map[string]*anypb.Any) (proto.Message, error) {
	transferCommitMsg, err := f.transferCoord.BuildCommitPayload(ctx, results)
	if err != nil {
		return nil, fmt.Errorf("failed to build instant reserve transfer commit: %w", err)
	}
	transferCommit, ok := transferCommitMsg.(*pbinternal.SendTransferCommitRequest)
	if !ok {
		return nil, fmt.Errorf("unexpected transfer commit payload type %T", transferCommitMsg)
	}

	// The reservation handle returned to the SSP is the coordinator's own swap
	// row id (participant rows have per-SO ids); the claim phase loads it by
	// this id on the coordinator and by requested_transfer_id on participants,
	// exactly as legacy.
	db, err := ent.GetDbFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get db: %w", err)
	}
	transferID, err := uuid.Parse(f.req.GetTransfer().GetTransferId())
	if err != nil {
		return nil, sparkerrors.InvalidArgumentMalformedField(fmt.Errorf("invalid transfer id: %w", err))
	}
	swap, err := db.UtxoSwap.Query().
		Where(
			entutxoswap.RequestedTransferIDEQ(transferID),
			entutxoswap.RequestTypeEQ(st.UtxoSwapRequestTypeInstant),
			entutxoswap.StatusEQ(st.UtxoSwapStatusCreated),
		).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("coordinator instant swap not found for transfer %s after prepare: %w", transferID, err)
	}

	f.response = &pbsspsvc.ReserveInstantStaticDepositUtxoSwapResponse{
		Transfer:   f.transferCoord.response.GetTransfer(),
		UtxoSwapId: swap.ID.String(),
	}
	return &pbinternal.ReserveInstantStaticDepositUtxoSwapCommitRequest{TransferCommit: transferCommit}, nil
}

// RollbackPayload carries only the requested transfer id, which selects the
// participant's own swap row; the transfer to roll back is derived from that
// row (see ReserveInstantStaticDepositUtxoSwapRollbackRequest's doc).
func (f *reserveInstantStaticDepositCoordinatorFlow) RollbackPayload() proto.Message {
	return &pbinternal.ReserveInstantStaticDepositUtxoSwapRollbackRequest{
		RequestedTransferId: f.req.GetTransfer().GetTransferId(),
	}
}

// ---------------------------------------------------------------------------
// Coordinator entrypoint
// ---------------------------------------------------------------------------

// ReserveInstantStaticDepositUtxoSwap is the public 2PC entrypoint for phase one of the instant
// static deposit claim: an external SSP credits the user a Spark transfer against a deposit UTXO
// that may still be unconfirmed, and the reservation stays open for the claim phase. It runs the
// coordinator-only validation (instant-enabled knob, soft UTXO check + duplicate short-circuit),
// fast-fails the instant user signature before any cross-operator work, then drives the engine.
// No FROST round-1 collection — reserve signs nothing against the UTXO.
func (o *StaticDepositHandler) ReserveInstantStaticDepositUtxoSwap(ctx context.Context, config *so.Config, req *pbsspsvc.ReserveInstantStaticDepositUtxoSwapRequest) (*pbsspsvc.ReserveInstantStaticDepositUtxoSwapResponse, error) {
	if req.GetTransfer().GetTransferPackage() == nil {
		return nil, sparkerrors.InvalidArgumentMissingField(fmt.Errorf("transfer_package is required"))
	}

	db, err := ent.GetDbFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get db: %w", err)
	}
	schemaNetwork, err := btcnetwork.FromProtoNetwork(req.GetOnChainUtxo().GetNetwork())
	if err != nil {
		return nil, err
	}
	knobService := knobs.GetKnobsService(ctx)
	if knobService == nil || knobService.GetValueTarget(knobs.KnobEnableInstantStaticDeposit, new(schemaNetwork.String()), 0) == 0 {
		return nil, sparkerrors.FailedPreconditionInvalidState(fmt.Errorf("instant static deposit is not enabled"))
	}

	// Soft UTXO check, matching legacy: the UTXO may be unconfirmed at reserve
	// time (that is the point of the instant flow), so only check for a
	// duplicate registration when it is already confirmed and reachable.
	targetUtxo, err := VerifiedTargetUtxoFromRequestWithThreshold(ctx, db, schemaNetwork, req.GetOnChainUtxo(), 1)
	if err != nil {
		return nil, err
	}
	if targetUtxo != nil {
		existingSwap, err := staticdeposit.GetRegisteredUtxoSwapForUtxo(ctx, db, targetUtxo.inner)
		if err != nil {
			return nil, fmt.Errorf("unable to check if utxo swap is already registered: %w", err)
		}
		if existingSwap != nil {
			return nil, sparkerrors.AlreadyExistsDuplicateOperation(fmt.Errorf("utxo swap is already registered"))
		}
	}

	// Fast-fail the instant user signature (and the cheap checks it depends on)
	// before any cross-operator work. Additive only — every participant
	// re-verifies in Prepare (createInstantReserveSwap); a coordinator-side
	// check is never a substitute.
	if req.GetValueSats() <= 0 || req.GetCreditAmountSats() < 0 || req.GetSecondaryCreditAmountSats() < 0 {
		return nil, sparkerrors.InvalidArgumentMalformedField(fmt.Errorf("amounts must be non-negative and value_sats must be positive"))
	}
	// Overflow-safe: compare each term against the remaining headroom (see the participant check).
	if req.GetCreditAmountSats() > req.GetValueSats() || req.GetSecondaryCreditAmountSats() > req.GetValueSats()-req.GetCreditAmountSats() {
		return nil, sparkerrors.InvalidArgumentMalformedField(fmt.Errorf("total credit amount exceeds value_sats"))
	}
	// Cross-field secondary invariants, mirrored from the participant so the
	// fast-fail rejects the same shapes before any cross-operator work.
	if req.GetSecondaryCreditAmountSats() == 0 && req.GetRequestedSecondaryTransferId() != "" {
		return nil, sparkerrors.InvalidArgumentMalformedField(fmt.Errorf("requested_secondary_transfer_id provided without secondary_credit_amount_sats"))
	}
	if req.GetSecondaryCreditAmountSats() > 0 && req.GetRequestedSecondaryTransferId() == "" {
		return nil, sparkerrors.InvalidArgumentMalformedField(fmt.Errorf("secondary_credit_amount_sats provided without requested_secondary_transfer_id"))
	}
	receiverIdentityPubKey, err := keys.ParsePublicKey(req.GetTransfer().GetReceiverIdentityPublicKey())
	if err != nil {
		return nil, sparkerrors.InvalidArgumentMalformedKey(fmt.Errorf("invalid receiver identity public key: %w", err))
	}
	if _, err := db.DepositAddress.Query().
		Where(
			depositaddress.Address(req.GetDestinationAddress()),
			depositaddress.OwnerIdentityPubkey(receiverIdentityPubKey),
			depositaddress.IsStatic(true),
		).
		Only(ctx); err != nil {
		if ent.IsNotFound(err) {
			return nil, sparkerrors.NotFoundMissingEntity(fmt.Errorf("deposit address %s not found", req.GetDestinationAddress()))
		}
		return nil, fmt.Errorf("unable to get deposit address: %w", err)
	}
	leafRefundMap, _, _ := loadLeafRefundMaps(req.GetTransfer())
	// Non-locking read: only immutable leaf values feed the signature
	// fast-fail; the engine-driven Prepare takes the authoritative locks.
	leaves, _, err := loadLeaves(ctx, db, leafRefundMap, false)
	if err != nil {
		return nil, fmt.Errorf("unable to load leaves: %w", err)
	}
	if len(leaves) == 0 {
		return nil, sparkerrors.InvalidArgumentMissingField(fmt.Errorf("no leaves found"))
	}
	totalAmount := getTotalTransferValue(leaves)
	if totalAmount != uint64(req.GetCreditAmountSats()) {
		return nil, fmt.Errorf("instant static deposit total leaf amount %d does not match credit_amount_sats %d", totalAmount, req.GetCreditAmountSats())
	}
	if err := validateInstantUserSignature(
		receiverIdentityPubKey,
		req.GetUserSignature(),
		req.GetSspSignature(),
		schemaNetwork,
		totalAmount,
		uint64(req.GetSecondaryCreditAmountSats()),
		req.GetDestinationAddress(),
		uint64(req.GetValueSats()),
	); err != nil {
		return nil, fmt.Errorf("user signature validation failed: %w", err)
	}

	reqInternal := &pbinternal.ReserveInstantStaticDepositUtxoSwapRequest{
		OnChainUtxo:                  req.GetOnChainUtxo(),
		SspSignature:                 req.GetSspSignature(),
		UserSignature:                req.GetUserSignature(),
		Transfer:                     req.GetTransfer(),
		DestinationAddress:           req.GetDestinationAddress(),
		ValueSats:                    req.GetValueSats(),
		CreditAmountSats:             req.GetCreditAmountSats(),
		SecondaryCreditAmountSats:    req.GetSecondaryCreditAmountSats(),
		RequestedSecondaryTransferId: req.GetRequestedSecondaryTransferId(),
	}

	handler := NewReserveInstantStaticDepositFlowHandler(config)
	// The delegated coordinator flow is built directly on this flow's typed
	// transfer handler so it carries the swap's transfer semantics from
	// construction.
	transferReq := convertV2ToV3SendTransferRequest(req.GetTransfer())
	parsedTransfer, err := parseSendTransferRequest(transferReq)
	if err != nil {
		return nil, fmt.Errorf("unable to parse reserve transfer request: %w", err)
	}
	transferCoord, err := buildSendTransferCoordinatorFlow(ctx, config, transferReq, parsedTransfer, "", handler.transfer)
	if err != nil {
		return nil, fmt.Errorf("unable to build transfer coordinator flow: %w", err)
	}
	flow := &reserveInstantStaticDepositCoordinatorFlow{
		ReserveInstantStaticDepositFlowHandler: handler,
		req:                                    reqInternal,
		transferCoord:                          transferCoord,
	}

	engine, err := consensus.GetEngine(ctx)
	if err != nil {
		return nil, err
	}
	selection := helper.OperatorSelection{Option: helper.OperatorSelectionOptionAll}
	if _, err := engine.Execute(ctx, pbgossip.ConsensusOperationType_CONSENSUS_OPERATION_TYPE_RESERVE_INSTANT_STATIC_DEPOSIT_UTXO_SWAP, &selection, flow); err != nil {
		return nil, fmt.Errorf("consensus reserve instant static deposit utxo swap failed: %w", err)
	}
	if flow.response == nil {
		return nil, fmt.Errorf("reserve instant static deposit consensus completed without building a response")
	}
	return flow.response, nil
}
