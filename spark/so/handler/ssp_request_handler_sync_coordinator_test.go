//go:build lightspark

package handler

import (
	"testing"

	"github.com/google/uuid"
	pb "github.com/lightsparkdev/spark/proto/spark"
	"github.com/stretchr/testify/require"
)

func TestCheckParentsAndMarkBadLeavesRejectsNilParentNodeWithoutPanic(t *testing.T) {
	nodeID := uuid.NewString()
	parentA := uuid.NewString()
	parentB := uuid.NewString()
	operatorsTreeNodesList := NewOperatorsTreeNodesList(2)
	operatorsTreeNodesList.Add(OperatorTreeNodes{
		operatorID: "operator-a",
		nodes: map[string]*pb.TreeNode{
			nodeID:  {Id: nodeID, ParentNodeId: &parentA},
			parentA: nil,
		},
	})
	operatorsTreeNodesList.Add(OperatorTreeNodes{
		operatorID: "operator-b",
		nodes: map[string]*pb.TreeNode{
			nodeID:  {Id: nodeID, ParentNodeId: &parentB},
			parentB: {Id: parentB, ParentNodeId: new(uuid.NewString())},
		},
	})
	currentNodes := operatorsTreeNodesList.GetOperatorToTreeNodeMap(nodeID)

	var err error
	var skip bool
	require.NotPanics(t, func() {
		skip, err = checkParentsAndMarkBadLeaves(
			t.Context(),
			nodeID,
			currentNodes,
			&operatorsTreeNodesList,
			func(_, _, _ string) {},
		)
	})
	require.False(t, skip)
	require.ErrorContains(t, err, "parent node")
	require.ErrorContains(t, err, "not found")
}
