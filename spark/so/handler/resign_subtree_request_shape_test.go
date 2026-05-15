//go:build lightspark

package handler

import (
	"testing"

	"github.com/btcsuite/btcd/wire"
	pb "github.com/lightsparkdev/spark/proto/spark"
	pbssp "github.com/lightsparkdev/spark/proto/spark_ssp_internal"
	"github.com/lightsparkdev/spark/so/ent"
	"github.com/stretchr/testify/require"
)

func TestReSignSubtreeRejectsNilRequest(t *testing.T) {
	handler := NewReSignSubtreeHandler(nil)

	resp, err := handler.ReSignSubtree(t.Context(), nil)

	require.Nil(t, resp)
	require.ErrorContains(t, err, "resign subtree request is required")
}

func TestValidateReSignSubtreeSigningJobsRejectsMalformedInputs(t *testing.T) {
	tests := []struct {
		name          string
		nodeJobs      *pbssp.NodeSigningJobs
		expectedCount int
		wantErr       string
	}{
		{
			name:          "nil node jobs",
			nodeJobs:      nil,
			expectedCount: 1,
			wantErr:       "signing jobs for node node-1 are required",
		},
		{
			name:          "wrong job count",
			nodeJobs:      &pbssp.NodeSigningJobs{},
			expectedCount: 1,
			wantErr:       "node node-1 expects exactly 1 signing job, got 0",
		},
		{
			name: "nil signing job",
			nodeJobs: &pbssp.NodeSigningJobs{
				SigningJobs: []*pb.UserSignedTxSigningJob{nil},
			},
			expectedCount: 1,
			wantErr:       "signing job 0 for node node-1 is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateReSignSubtreeSigningJobs("node-1", test.nodeJobs, test.expectedCount, "node")
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestBuildReSignSubtreeSigningJobsRejectsNilJobBeforeDereference(t *testing.T) {
	handler := NewReSignSubtreeHandler(nil)

	_, err := handler.buildSplitTxSigningJobs(
		&ent.TreeNode{},
		&ent.SigningKeyshare{},
		wire.NewTxOut(1, []byte{0x51}),
		nil,
		&pbssp.NodeSigningJobs{SigningJobs: []*pb.UserSignedTxSigningJob{nil}},
	)

	require.ErrorContains(t, err, "signing job 0 for split node")
}
