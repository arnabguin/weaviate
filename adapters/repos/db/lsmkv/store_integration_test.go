//                           _       _
// __      _____  __ ___   ___  __ _| |_ ___
// \ \ /\ / / _ \/ _` \ \ / / |/ _` | __/ _ \
//  \ V  V /  __/ (_| |\ V /| | (_| | ||  __/
//   \_/\_/ \___|\__,_| \_/ |_|\__,_|\__\___|
//
//  Copyright © 2016 - 2023 Weaviate B.V. All rights reserved.
//
//  CONTACT: hello@weaviate.io
//

//go:build integrationTest
// +build integrationTest

package lsmkv

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreLifecycle(t *testing.T) {
	dirName := t.TempDir()

	t.Run("cycle 1", func(t *testing.T) {
		store, err := New(dirName, "", nullLogger(), nil)
		require.Nil(t, err)

		err = store.CreateOrLoadBucket(testCtx(), "bucket1", WithStrategy(StrategyReplace))
		require.Nil(t, err)

		b1 := store.Bucket("bucket1")
		require.NotNil(t, b1)

		err = b1.Put([]byte("name"), []byte("Jane Doe"))
		require.Nil(t, err)

		err = store.CreateOrLoadBucket(testCtx(), "bucket2", WithStrategy(StrategyReplace))
		require.Nil(t, err)

		b2 := store.Bucket("bucket2")
		require.NotNil(t, b2)

		err = b2.Put([]byte("foo"), []byte("bar"))
		require.Nil(t, err)

		err = store.Shutdown(context.Background())
		require.Nil(t, err)
	})

	t.Run("cycle 2", func(t *testing.T) {
		store, err := New(dirName, "", nullLogger(), nil)
		require.Nil(t, err)

		err = store.CreateOrLoadBucket(testCtx(), "bucket1", WithStrategy(StrategyReplace))
		require.Nil(t, err)

		b1 := store.Bucket("bucket1")
		require.NotNil(t, b1)

		err = store.CreateOrLoadBucket(testCtx(), "bucket2", WithStrategy(StrategyReplace))
		require.Nil(t, err)

		b2 := store.Bucket("bucket2")
		require.NotNil(t, b2)

		res, err := b1.Get([]byte("name"))
		require.Nil(t, err)
		assert.Equal(t, []byte("Jane Doe"), res)

		res, err = b2.Get([]byte("foo"))
		require.Nil(t, err)
		assert.Equal(t, []byte("bar"), res)

		err = store.Shutdown(context.Background())
		require.Nil(t, err)
	})
}
