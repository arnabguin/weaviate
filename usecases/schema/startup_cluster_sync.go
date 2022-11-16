//                           _       _
// __      _____  __ ___   ___  __ _| |_ ___
// \ \ /\ / / _ \/ _` \ \ / / |/ _` | __/ _ \
//  \ V  V /  __/ (_| |\ V /| | (_| | ||  __/
//   \_/\_/ \___|\__,_| \_/ |_|\__,_|\__\___|
//
//  Copyright © 2016 - 2022 SeMI Technologies B.V. All rights reserved.
//
//  CONTACT: hello@semi.technology
//

package schema

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
)

// startupClusterSync tries to determine what - if any - schema migration is
// required at startup. If a node is the first in a cluster the assumption is
// that its state is the truth.
//
// For the n-th node (where n>1) there is a potential for conflict if the
// schemas aren't in sync:
//
// - If Node 1 has a non-nil schema, but Node 2 has a nil-schema, then we can
// consider Node 2 to be a new node that is just joining the cluster. In this
// case, we can copy the state from the existing nodes (if they agree on a
// schema)
//
// - If Node 1 and Node 2 have an identical schema, then we can assume that the
// startup was just an ordinary (re)start of the node. No action is required.
//
// - If Node 1 and Node 2 both have a schema, but they aren't in sync, the
// cluster is broken. This state cannot be automatically recovered from and
// startup needs to fail. Manual intervention would be required in this case.
func (m *Manager) startupClusterSync(ctx context.Context,
	localSchema *State,
) error {
	nodes := m.clusterState.AllNames()
	if len(nodes) <= 1 {
		return m.startupHandleSingleNode(ctx, nodes)
	}

	if isEmpty(localSchema) {
		return m.startupJoinCluster(ctx, localSchema)
	}

	return m.validateSchemaCorruption(ctx, localSchema)
}

// startupHandleSingleNode deals with the case where there is only a single
// node in the cluster. In the vast majority of cases there is nothing to do.
// An edge case would be where the cluster has size=0, or size=1 but the node's
// name is not the local name's node. This would indicate a broken cluster and
// can't be recovered from
func (m *Manager) startupHandleSingleNode(ctx context.Context,
	nodes []string,
) error {
	localName := m.clusterState.LocalName()
	if len(nodes) == 0 {
		return fmt.Errorf("corrupt cluster state: cluster has size=0")
	}

	if nodes[0] != localName {
		return fmt.Errorf("corrupt cluster state: only node in the cluster does not "+
			"match local node name: %v vs %s", nodes, localName)
	}

	m.logger.WithFields(logrusStartupSyncFields()).
		Debug("Only node in the cluster at this point. " +
			"No schema sync necessary.")

	// startup is complete
	return nil
}

// startupJoinCluster migrates the schema for a new node. The assumption is
// that other nodes have schema state and we need to migrate this schema to the
// local node transactionally. In other words, this startup process can not
// occur concurrently with a user-initiated schema update. One of those must
// fail.
//
// There is one edge case: The cluster could consist of multiple nodes which
// are empty. In this case, no migration is required.
func (m *Manager) startupJoinCluster(ctx context.Context,
	localSchema *State,
) error {
	tx, err := m.cluster.BeginTransaction(ctx, ReadSchema, nil)
	if err != nil {
		return fmt.Errorf("read schema: open transaction: %w", err)
	}

	// this tx is read-only, so we don't have to worry about aborting it, the
	// close should be the same on both happy and unhappy path
	defer m.cluster.CloseReadTransaction(ctx, tx)

	pl, ok := tx.Payload.(ReadSchemaPayload)
	if !ok {
		return fmt.Errorf("unrecognized tx response payload: %T", tx.Payload)
	}

	// by the time we're here the consensus function has run, so we can be sure
	// that all other nodes agree on this schema.

	if isEmpty(pl.Schema) {
		// already in sync, nothing to do
		return nil
	}

	m.state = *pl.Schema

	m.saveSchema(ctx)

	return nil
}

// validateSchemaCorruption makes sure that - given that all nodes in the
// cluster have a schema - they are in sync. If not the cluster is considered
// broken and needs to be repaired manually
func (m *Manager) validateSchemaCorruption(ctx context.Context,
	localSchema *State,
) error {
	tx, err := m.cluster.BeginTransaction(ctx, ReadSchema, nil)
	if err != nil {
		return fmt.Errorf("read schema: open transaction: %w", err)
	}

	// this tx is read-only, so we don't have to worry about aborting it, the
	// close should be the same on both happy and unhappy path
	defer m.cluster.CloseReadTransaction(ctx, tx)

	pl, ok := tx.Payload.(ReadSchemaPayload)
	if !ok {
		return fmt.Errorf("unrecognized tx response payload: %T", tx.Payload)
	}

	if !Equal(localSchema, pl.Schema) {
		return fmt.Errorf("corrupt cluster: other nodes have consensus on schema, " +
			"but local node has a different (non-null) schema")
	}

	return nil
}

func logrusStartupSyncFields() logrus.Fields {
	return logrus.Fields{"action": "startup_cluster_schema_sync"}
}

func isEmpty(schema *State) bool {
	if schema.ObjectSchema == nil {
		return true
	}

	if len(schema.ObjectSchema.Classes) == 0 {
		return true
	}

	return false
}