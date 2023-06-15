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

package hybrid

import (
	"fmt"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/weaviate/weaviate/entities/search"
)

func TestFusionRelativeScore(t *testing.T) {
	cases := []struct {
		weights        []float64
		inputScores    [][]float32
		expectedScores []float32
		expectedOrder  []uint64
	}{
		{weights: []float64{0.5, 0.5}, inputScores: [][]float32{{1, 2, 3}, {0, 1, 2}}, expectedScores: []float32{1, 0.5, 0}, expectedOrder: []uint64{2, 1, 0}},
		{weights: []float64{0.5, 0.5}, inputScores: [][]float32{{0, 2, 0.1}, {0, 0.2, 2}}, expectedScores: []float32{0.55, 0.525, 0}, expectedOrder: []uint64{1, 2, 0}},
		{weights: []float64{0.75, 0.25}, inputScores: [][]float32{{0.5, 0.5, 0}, {0, 0.01, 0.001}}, expectedScores: []float32{1, 0.75, 0.025}, expectedOrder: []uint64{1, 0, 2}},
		{weights: []float64{0.75, 0.25}, inputScores: [][]float32{{}, {}}, expectedScores: []float32{}, expectedOrder: []uint64{}},
		{weights: []float64{0.75, 0.25}, inputScores: [][]float32{{1, 1}, {1, 2}}, expectedScores: []float32{0.25, 0}, expectedOrder: []uint64{1, 0}},
	}
	for _, tt := range cases {
		t.Run("hybrid fusion", func(t *testing.T) {
			var results [][]*Result
			for i := range tt.inputScores {
				var result []*Result
				for j, score := range tt.inputScores[i] {
					result = append(result, &Result{uint64(j), &search.Result{SecondarySortValue: score, ID: strfmt.UUID(fmt.Sprint(j))}})
				}
				results = append(results, result)
			}
			fused := FusionRelativeScore(tt.weights, results)
			fusedScores := []float32{} // don't use nil slice declaration, should be explicitly empty
			fusedOrder := []uint64{}

			for _, score := range fused {
				fusedScores = append(fusedScores, score.Score)
				fusedOrder = append(fusedOrder, score.DocID)
			}

			assert.InDeltaSlice(t, tt.expectedScores, fusedScores, 0.0001)
			assert.Equal(t, tt.expectedOrder, fusedOrder)
		})
	}
}