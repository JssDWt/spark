package handler

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lightsparkdev/spark/common/btcnetwork"
	pbcommon "github.com/lightsparkdev/spark/proto/common"
	pbgossip "github.com/lightsparkdev/spark/proto/gossip"
	pbspark "github.com/lightsparkdev/spark/proto/spark"
	pbsspsvc "github.com/lightsparkdev/spark/proto/spark_ssp"
	pbinternal "github.com/lightsparkdev/spark/proto/spark_internal"
	"github.com/lightsparkdev/spark/so"
	"github.com/lightsparkdev/spark/so/consensus"
	"github.com/lightsparkdev/spark/so/ent"
	st "github.com/lightsparkdev/spark/so/ent/schema/schematype"
	entutxoswap "github.com/lightsparkdev/spark/so/ent/utxoswap"
	sparkerrors "github.com/lightsparkdev/spark/so/errors"
	"github.com/lightsparkdev/spark/so/frost"
	"github.com/lightsparkdev/spark/so/helper"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// claimInstantStaticDepositCoordinatorFlow is the coordinator side of phase two of the instant
// static deposit claim: it drives the 2PC engine, delegates the optional secondary transfer to
// the send-transfer coordinator flow, aggregates the deposit-UTXO spend signature, completes the
// swap, and builds the public ClaimInstantStaticDepositUtxoSwap response.

type claimInstantStaticDepositCoordinatorFlow struct {
	*ClaimInstantStaticDepositFlowHandler

	req *pbinternal.ClaimInstantStaticDepositUtxoSwapRequest
	// transferCoord aggregates the optional secondary transfer's leaf
	// signatures, applies the transfer commit on the coordinator, and builds the
	// transfer response. nil when the reservation has no secondary credit.
	transferCoord *sendTransferCoordinatorFlow
	// Spend-tx FROST round-1 commitments the coordinator collected
	// (GetSigningCommitments) before Execute, keyed by operator id.
	spendCommitments       map[string]*pbcommon.SigningCommitment
	spendCommitmentsParsed map[string]frost.SigningCommitment

	// response is populated in BuildCommitPayload for the public handler to return.
	response *pbsspsvc.ClaimInstantStaticDepositUtxoSwapResponse
}

var _ consensus.CoordinatorFlow = (*claimInstantStaticDepositCoordinatorFlow)(nil)

func (f *claimInstantStaticDepositCoordinatorFlow) PrepareOp() proto.Message {
	return &pbinternal.ClaimInstantStaticDepositUtxoSwapPrepareRequest{
		OriginalRequest:           f.req,
		SpendTxSigningCommitments: f.spendCommitments,
	}
}

// BuildCommitPayload first delegates the optional secondary transfer to the
// send-transfer coordinator flow (leaf-signature aggregation, coordinator-local
// transfer commit, transfer response), then aggregates the spend-tx signature
// from the same prepare results, stores it on the coordinator's swap row, marks
// the swap COMPLETED, and builds the RPC response. The transfer delegation MUST
// run first: CompleteUtxoSwap requires every linked transfer to be sent.
func (f *claimInstantStaticDepositCoordinatorFlow) BuildCommitPayload(ctx context.Context, results map[string]*anypb.Any) (proto.Message, error) {
	var transferCommit *pbinternal.SendTransferCommitRequest
	if f.transferCoord != nil {
		transferCommitMsg, err := f.transferCoord.BuildCommitPayload(ctx, results)
		if err != nil {
			return nil, fmt.Errorf("failed to build claim secondary transfer commit: %w", err)
		}
		var ok bool
		transferCommit, ok = transferCommitMsg.(*pbinternal.SendTransferCommitRequest)
		if !ok {
			return nil, fmt.Errorf("unexpected transfer commit payload type %T", transferCommitMsg)
		}
	}

	allShares, _, err := collectSignatureShares(results)
	if err != nil {
		return nil, fmt.Errorf("failed to collect signature shares: %w", err)
	}

	db, err := ent.GetDbFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get db: %w", err)
	}
	// The coordinator's own flow.Prepare linked the utxo edge onto this row
	// earlier in this same request transaction, so both rows are exclusively
	// held by this tx — no extra ForUpdate is needed.
	transferID, err := uuid.Parse(f.req.GetTransferId())
	if err != nil {
		return nil, sparkerrors.InvalidArgumentMalformedField(fmt.Errorf("invalid transfer_id: %w", err))
	}
	swap, err := loadInstantSwapForClaim(ctx, db, transferID, false /* forUpdate */, st.UtxoSwapStatusCreated)
	if err != nil {
		return nil, err
	}
	targetUtxo, err := swap.QueryUtxo().Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load utxo linked by prepare for swap %s: %w", swap.ID, err)
	}
	depositAddress, err := targetUtxo.QueryDepositAddress().Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get utxo deposit address: %w", err)
	}
	signingKeyshare, err := depositAddress.QuerySigningKeyshare().Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get signing keyshare: %w", err)
	}
	verifyingKey := signingKeyshare.PublicKey.Add(depositAddress.OwnerSigningPubkey)
	spendTxSighash, _, err := GetTxSigningInfo(ctx, targetUtxo, f.req.GetSpendTxSigningJob().GetRawTx())
	if err != nil {
		return nil, fmt.Errorf("failed to get spend tx sighash: %w", err)
	}
	userNonce := frost.SigningCommitment{}
	if err := userNonce.UnmarshalProto(f.req.GetSpendTxSigningJob().GetSigningNonceCommitment()); err != nil {
		return nil, fmt.Errorf("failed to parse spend tx nonce commitment: %w", err)
	}

	jobID := claimInstantSpendTxJobID(f.req.GetOnChainUtxo().GetTxid(), f.req.GetOnChainUtxo().GetVout())
	job := &helper.SigningJob{
		JobID:             jobID,
		SigningKeyshareID: signingKeyshare.ID,
		Message:           spendTxSighash,
		VerifyingKey:      &verifyingKey,
		UserCommitment:    &userNonce,
	}

	operatorIDs := make([]string, 0, len(f.spendCommitmentsParsed))
	for id := range f.spendCommitmentsParsed {
		operatorIDs = append(operatorIDs, id)
	}
	selection, err := helper.NewPreSelectedOperatorSelection(f.config, operatorIDs)
	if err != nil {
		return nil, fmt.Errorf("unable to build signing operator selection: %w", err)
	}
	keyPackages, err := ent.GetKeyPackages(ctx, f.config, []uuid.UUID{signingKeyshare.ID})
	if err != nil {
		return nil, fmt.Errorf("unable to get key packages: %w", err)
	}
	round2, ok := allShares[jobID.String()]
	if !ok {
		return nil, fmt.Errorf("no round-2 shares collected for spend tx job %s", jobID)
	}
	signingResults, err := helper.BuildSigningResults(
		f.config, selection,
		[]*helper.SigningJob{job}, keyPackages,
		[]map[string]frost.SigningCommitment{f.spendCommitmentsParsed},
		map[uuid.UUID]map[string][]byte{jobID: round2},
	)
	if err != nil {
		return nil, fmt.Errorf("unable to build spend tx signing result: %w", err)
	}
	if len(signingResults) == 0 {
		return nil, fmt.Errorf("no signing result produced for spend tx job %s", jobID)
	}
	signingResultProto := signingResults[0].MarshalProto()
	signingResultBytes, err := proto.Marshal(signingResultProto)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal signing result bytes: %w", err)
	}
	if _, err := swap.Update().SetSpendTxSigningResult(signingResultBytes).Save(ctx); err != nil {
		return nil, fmt.Errorf("unable to store spend tx signing result: %w", err)
	}
	if err := CompleteUtxoSwap(ctx, swap); err != nil {
		return nil, fmt.Errorf("unable to complete coordinator utxo swap: %w", err)
	}

	var transferProto *pbspark.Transfer
	if f.transferCoord != nil {
		transferProto = f.transferCoord.response.GetTransfer()
	}
	f.response = &pbsspsvc.ClaimInstantStaticDepositUtxoSwapResponse{
		Transfer:             transferProto,
		SpendTxSigningResult: signingResultProto,
		DepositAddress: &pbspark.DepositAddressQueryResult{
			DepositAddress:       depositAddress.Address,
			UserSigningPublicKey: depositAddress.OwnerSigningPubkey.Serialize(),
			VerifyingPublicKey:   verifyingKey.Serialize(),
			LeafId:               new(depositAddress.NodeID.String()),
		},
	}
	return &pbinternal.ClaimInstantStaticDepositUtxoSwapCommitRequest{
		TransferId:     f.req.GetTransferId(),
		TransferCommit: transferCommit,
	}, nil
}

// RollbackPayload carries the primary transfer id for observability only —
// the participant Rollback is a deliberate no-op and shape-validates the
// payload without acting on it (see the handler Rollback doc).
func (f *claimInstantStaticDepositCoordinatorFlow) RollbackPayload() proto.Message {
	return &pbinternal.ClaimInstantStaticDepositUtxoSwapRollbackRequest{
		TransferId: f.req.GetTransferId(),
	}
}

// ---------------------------------------------------------------------------
// Coordinator entrypoint
// ---------------------------------------------------------------------------

// ClaimInstantStaticDepositUtxoSwap is the public entrypoint for phase two of the instant static
// deposit claim: the reserved UTXO has confirmed, so an external SSP asks the SE to co-sign its
// spend tx (and send any secondary transfer) and complete the swap. It loads the reservation the
// reserve phase created (by the utxo_swap_id it returned) and locks it FOR UPDATE, then drives
// the 2PC claim.
//
// The operator claim is one-shot: completing the swap persists the spend-tx FROST signature and
// consumes the round-1 nonces, so re-running consensus on a retry would fail. The reservation is
// therefore loaded in either CREATED or COMPLETED status: a COMPLETED row means an earlier claim
// already ran, so the stored signature is returned without touching the engine, letting a crashed
// SSP worker safely retry. The FOR UPDATE read also serializes concurrent retries so at most one
// runs the flow.
func (o *StaticDepositHandler) ClaimInstantStaticDepositUtxoSwap(ctx context.Context, config *so.Config, req *pbsspsvc.ClaimInstantStaticDepositUtxoSwapRequest) (*pbsspsvc.ClaimInstantStaticDepositUtxoSwapResponse, error) {
	swapID, err := uuid.Parse(req.GetUtxoSwapId())
	if err != nil {
		return nil, sparkerrors.InvalidArgumentMalformedField(fmt.Errorf("invalid utxo_swap_id: %w", err))
	}
	db, err := ent.GetDbFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get db: %w", err)
	}
	// utxo_swap_id names the coordinator's own reservation row (participant rows have per-SO
	// ids); the RequestType filter keeps a client-supplied id from resolving a row created by a
	// different swap flow.
	swap, err := db.UtxoSwap.Query().
		Where(
			entutxoswap.IDEQ(swapID),
			entutxoswap.RequestTypeEQ(st.UtxoSwapRequestTypeInstant),
			entutxoswap.StatusIn(st.UtxoSwapStatusCreated, st.UtxoSwapStatusCompleted),
		).
		ForUpdate().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, sparkerrors.NotFoundMissingEntity(fmt.Errorf("instant reservation %s not found", swapID))
		}
		return nil, fmt.Errorf("unable to load instant reservation %s: %w", swapID, err)
	}

	if swap.Status == st.UtxoSwapStatusCompleted {
		if len(swap.SpendTxSigningResult) == 0 {
			return nil, sparkerrors.FailedPreconditionInvalidState(fmt.Errorf("instant reservation %s is completed but has no stored spend tx signing result", swapID))
		}
		return o.buildCompletedClaimResponse(ctx, swap)
	}

	return o.claimInstantStaticDepositUtxoSwapConsensus(ctx, config, req, swap)
}

// buildCompletedClaimResponse reconstructs the claim response for an already-completed
// reservation entirely from persisted state — the stored spend-tx signature, the deposit address
// derived from the swap's linked (now-confirmed) UTXO, and the secondary transfer if the
// reservation carried one. It runs no consensus and mutates nothing, so a crashed SSP worker can
// retry the claim and get the same result. The COMPLETED status guarantees the utxo edge exists
// (the schema requires it), so the deposit-address lookups below cannot miss.
func (o *StaticDepositHandler) buildCompletedClaimResponse(ctx context.Context, swap *ent.UtxoSwap) (*pbsspsvc.ClaimInstantStaticDepositUtxoSwapResponse, error) {
	signingResult := &pbspark.SigningResult{}
	if err := proto.Unmarshal(swap.SpendTxSigningResult, signingResult); err != nil {
		return nil, fmt.Errorf("unable to unmarshal stored spend tx signing result: %w", err)
	}

	targetUtxo, err := swap.QueryUtxo().Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load utxo for completed swap %s: %w", swap.ID, err)
	}
	depositAddress, err := targetUtxo.QueryDepositAddress().Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get utxo deposit address: %w", err)
	}
	signingKeyshare, err := depositAddress.QuerySigningKeyshare().Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get signing keyshare: %w", err)
	}
	verifyingKey := signingKeyshare.PublicKey.Add(depositAddress.OwnerSigningPubkey)

	// The secondary transfer edge is present iff the reservation carried a secondary credit; its
	// absence leaves the response transfer nil, matching the live claim response.
	var transferProto *pbspark.Transfer
	secondaryTransfer, err := swap.QuerySecondaryTransfer().Only(ctx)
	switch {
	case err == nil:
		transferProto, err = secondaryTransfer.MarshalProto(ctx)
		if err != nil {
			return nil, fmt.Errorf("unable to marshal secondary transfer: %w", err)
		}
	case ent.IsNotFound(err):
	default:
		return nil, fmt.Errorf("unable to load secondary transfer: %w", err)
	}

	return &pbsspsvc.ClaimInstantStaticDepositUtxoSwapResponse{
		SpendTxSigningResult: signingResult,
		Transfer:             transferProto,
		DepositAddress: &pbspark.DepositAddressQueryResult{
			DepositAddress:       depositAddress.Address,
			UserSigningPublicKey: depositAddress.OwnerSigningPubkey.Serialize(),
			VerifyingPublicKey:   verifyingKey.Serialize(),
			LeafId:               new(depositAddress.NodeID.String()),
		},
	}, nil
}

// claimInstantStaticDepositUtxoSwapConsensus is the 2PC entrypoint for phase two of the
// instant static deposit claim. The public wrapper (ClaimInstantStaticDepositUtxoSwap) has
// already loaded and ForUpdate-locked the CREATED reservation and served the already-completed
// idempotency short-circuit, so this validates the spend-tx job, collects FROST round-1
// commitments (keeping the public RPC a single call), then drives the engine.
func (o *StaticDepositHandler) claimInstantStaticDepositUtxoSwapConsensus(ctx context.Context, config *so.Config, req *pbsspsvc.ClaimInstantStaticDepositUtxoSwapRequest, utxoSwap *ent.UtxoSwap) (*pbsspsvc.ClaimInstantStaticDepositUtxoSwapResponse, error) {
	// Reject a missing/malformed spend-tx job before any cross-operator work:
	// round-1 collection consumes a persisted FROST nonce on every selected
	// operator, and every participant's Prepare re-validates this same field.
	if req.GetSpendTxSigningJob() == nil {
		return nil, sparkerrors.InvalidArgumentMissingField(fmt.Errorf("spend_tx_signing_job is required"))
	}
	if req.GetSpendTxSigningJob().GetSigningNonceCommitment() == nil {
		return nil, sparkerrors.InvalidArgumentMissingField(fmt.Errorf("spend_tx_signing_job.signing_nonce_commitment is required"))
	}
	if err := (&frost.SigningCommitment{}).UnmarshalProto(req.GetSpendTxSigningJob().GetSigningNonceCommitment()); err != nil {
		return nil, sparkerrors.InvalidArgumentMalformedField(fmt.Errorf("invalid spend tx signing nonce commitment: %w", err))
	}

	// Fast-fail the UTXO confirmation and spend-tx checks against this SO's own
	// state before any cross-operator work, mirroring the fixed-swap consensus
	// entrypoint: round-1 collection consumes a persisted FROST nonce on every
	// selected operator and Execute fans Prepare out to all of them. Additive
	// only — every participant re-runs these same checks in Prepare
	// (linkUtxoToReservedSwap); a coordinator-side check is never a substitute.
	db, err := ent.GetDbFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get db: %w", err)
	}
	network, err := btcnetwork.FromProtoNetwork(req.GetOnChainUtxo().GetNetwork())
	if err != nil {
		return nil, fmt.Errorf("unable to parse network: %w", err)
	}
	targetUtxo, err := VerifiedTargetUtxoFromRequestWithThreshold(ctx, db, network, req.GetOnChainUtxo(), 1)
	if err != nil {
		return nil, fmt.Errorf("failed to verify on-chain utxo: %w", err)
	}
	if targetUtxo == nil {
		return nil, sparkerrors.FailedPreconditionInsufficientConfirmations(fmt.Errorf("on-chain utxo not found or not confirmed"))
	}
	if targetUtxo.inner.Amount != utxoSwap.UtxoValueSats {
		return nil, fmt.Errorf("utxo amount %d does not match swap utxo_value_sats %d", targetUtxo.inner.Amount, utxoSwap.UtxoValueSats)
	}
	if err := validateStaticDepositSpendTxSpendsTargetUtxo(targetUtxo, req.GetSpendTxSigningJob().GetRawTx()); err != nil {
		return nil, err
	}

	reqInternal := &pbinternal.ClaimInstantStaticDepositUtxoSwapRequest{
		OnChainUtxo:       req.GetOnChainUtxo(),
		Transfer:          req.GetTransfer(),
		SpendTxSigningJob: req.GetSpendTxSigningJob(),
		// Every SO locates its row by the primary transfer id from the
		// coordinator's ForUpdate-locked row — exactly what the legacy
		// SaveUtxoForInstantStaticDeposit fanout carried.
		TransferId: utxoSwap.RequestedTransferID.String(),
	}

	// Collect spend-tx FROST round-1 commitments server-side so round-2 can run
	// inside the engine's Prepare while the public RPC stays a single call.
	round1, err := helper.GetSigningCommitments(ctx, config, 1, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to collect round-1 signing commitments: %w", err)
	}

	flow, err := buildClaimInstantStaticDepositCoordinatorFlow(ctx, config, reqInternal, round1)
	if err != nil {
		return nil, fmt.Errorf("unable to build coordinator flow: %w", err)
	}
	engine, err := consensus.GetEngine(ctx)
	if err != nil {
		return nil, err
	}
	selection := helper.OperatorSelection{Option: helper.OperatorSelectionOptionAll}
	if _, err := engine.Execute(ctx, pbgossip.ConsensusOperationType_CONSENSUS_OPERATION_TYPE_CLAIM_INSTANT_STATIC_DEPOSIT_UTXO_SWAP, &selection, flow); err != nil {
		return nil, fmt.Errorf("consensus claim instant static deposit utxo swap failed: %w", err)
	}
	if flow.response == nil {
		return nil, fmt.Errorf("claim instant static deposit consensus completed without building a response")
	}
	return flow.response, nil
}

// buildClaimInstantStaticDepositCoordinatorFlow parses the coordinator-collected
// round-1 commitments and pre-builds the delegated send-transfer coordinator
// flow for the optional secondary transfer.
func buildClaimInstantStaticDepositCoordinatorFlow(ctx context.Context, config *so.Config, req *pbinternal.ClaimInstantStaticDepositUtxoSwapRequest, round1 map[string][]frost.SigningCommitment) (*claimInstantStaticDepositCoordinatorFlow, error) {
	spendCommitments := make(map[string]*pbcommon.SigningCommitment, len(round1))
	spendCommitmentsParsed := make(map[string]frost.SigningCommitment, len(round1))
	for opID, commitments := range round1 {
		// GetSigningCommitments is called with count=1, so exactly one
		// commitment per operator is expected; guard against a future count
		// change silently desyncing round-1 from round-2 aggregation.
		if len(commitments) != 1 {
			return nil, fmt.Errorf("expected exactly 1 round-1 commitment for operator %s, got %d", opID, len(commitments))
		}
		spendCommitments[opID] = commitments[0].MarshalProto()
		spendCommitmentsParsed[opID] = commitments[0]
	}

	handler := NewClaimInstantStaticDepositFlowHandler(config)
	var transferCoord *sendTransferCoordinatorFlow
	if req.GetTransfer() != nil {
		// Built on the claim's typed transfer handler so the secondary carries
		// the swap's transfer semantics from construction.
		transferReq := convertV2ToV3SendTransferRequest(req.GetTransfer())
		parsedTransfer, err := parseSendTransferRequest(transferReq)
		if err != nil {
			return nil, fmt.Errorf("unable to parse secondary transfer request: %w", err)
		}
		transferCoord, err = buildSendTransferCoordinatorFlow(ctx, config, transferReq, parsedTransfer, "", handler.transfer)
		if err != nil {
			return nil, fmt.Errorf("unable to build secondary transfer coordinator flow: %w", err)
		}
	}

	return &claimInstantStaticDepositCoordinatorFlow{
		ClaimInstantStaticDepositFlowHandler: handler,
		req:                                  req,
		transferCoord:                        transferCoord,
		spendCommitments:                     spendCommitments,
		spendCommitmentsParsed:               spendCommitmentsParsed,
	}, nil
}
