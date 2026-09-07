package grpc

import (
	"context"

	pb "github.com/lightsparkdev/spark/proto/spark"
	pbsspsvc "github.com/lightsparkdev/spark/proto/spark_ssp"
	"github.com/lightsparkdev/spark/so"
	"github.com/lightsparkdev/spark/so/handler"
)

// SparkSspServer implements SparkSspService, the calls a Spark service provider
// makes that a wallet never does.
type SparkSspServer struct {
	pbsspsvc.UnimplementedSparkSspServiceServer
	config *so.Config
}

// NewSparkSspServer creates a new SparkSspServer.
func NewSparkSspServer(config *so.Config) *SparkSspServer {
	return &SparkSspServer{config: config}
}

// PrepareTreeAddress prepares the addresses of a deposit tree's nodes.
func (s *SparkSspServer) PrepareTreeAddress(ctx context.Context, req *pb.PrepareTreeAddressRequest) (*pb.PrepareTreeAddressResponse, error) {
	treeCreationHandler := handler.NewTreeCreationHandler(s.config)
	return treeCreationHandler.PrepareTreeAddress(ctx, req)
}

// CreateTree builds and signs a transaction tree from a deposit.
func (s *SparkSspServer) CreateTree(ctx context.Context, req *pb.CreateTreeRequest) (*pb.CreateTreeResponse, error) {
	treeCreationHandler := handler.NewTreeCreationHandler(s.config)
	return treeCreationHandler.CreateTree(ctx, req)
}

// CounterLeafSwapV3 answers a user's swap with the counter transfer.
func (s *SparkSspServer) CounterLeafSwapV3(ctx context.Context, req *pb.CounterLeafSwapRequest) (*pb.CounterLeafSwapResponse, error) {
	transferHandler := handler.NewTransferHandler(s.config)
	return transferHandler.CounterLeafSwapV3(ctx, req)
}

// InitiateUtxoSwap claims a static deposit as a fixed-amount utxo swap.
func (s *SparkSspServer) InitiateUtxoSwap(ctx context.Context, req *pb.InitiateUtxoSwapRequest) (*pb.InitiateUtxoSwapResponse, error) {
	depositHandler := handler.NewStaticDepositHandler(s.config)
	return depositHandler.InitiateUtxoSwap(ctx, s.config, req)
}

// ReserveInstantStaticDepositUtxoSwap credits the user against a deposit that
// may not have confirmed yet.
func (s *SparkSspServer) ReserveInstantStaticDepositUtxoSwap(ctx context.Context, req *pbsspsvc.ReserveInstantStaticDepositUtxoSwapRequest) (*pbsspsvc.ReserveInstantStaticDepositUtxoSwapResponse, error) {
	depositHandler := handler.NewStaticDepositHandler(s.config)
	return depositHandler.ReserveInstantStaticDepositUtxoSwap(ctx, s.config, req)
}

// ClaimInstantStaticDepositUtxoSwap completes a reserved swap once its deposit
// has confirmed.
func (s *SparkSspServer) ClaimInstantStaticDepositUtxoSwap(ctx context.Context, req *pbsspsvc.ClaimInstantStaticDepositUtxoSwapRequest) (*pbsspsvc.ClaimInstantStaticDepositUtxoSwapResponse, error) {
	depositHandler := handler.NewStaticDepositHandler(s.config)
	return depositHandler.ClaimInstantStaticDepositUtxoSwap(ctx, s.config, req)
}
