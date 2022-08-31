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

package diskAnn_test

import (
	"context"
	"encoding/gob"
	"fmt"
	"math/rand"
	"os"
	"time"

	"testing"

	"github.com/pkg/errors"
	ssdhelpers "github.com/semi-technologies/weaviate/adapters/repos/db/vector/ssdHelpers"
	testinghelpers "github.com/semi-technologies/weaviate/adapters/repos/db/vector/testingHelpers"
)

/*func TestClusteredVamana(t *testing.T) {
	R := 4
	L := 10
	dimensions := 2
	vectors_size := 1000
	vectors := generate_vecs(vectors_size, dimensions)
	w := 1024
	index, _ := New(Config{
		R:     R,
		L:     L,
		Alpha: float32(1.1),
		VectorForIDThunk: func(ctx context.Context, id uint64) ([]float32, error) {
			return vectors[int(id)], nil
		},
		VectorsSize:        uint64(vectors_size),
		Distance:           ssdhelpers.L2,
		ClustersSize:       40,
		ClusterOverlapping: 2,
	})
	index.BuildIndex()

	index2, _ := New(Config{
		R:     R,
		L:     L,
		Alpha: float32(1.2),
		VectorForIDThunk: func(ctx context.Context, id uint64) ([]float32, error) {
			return vectors[int(id)], nil
		},
		VectorsSize:        uint64(vectors_size),
		Distance:           ssdhelpers.L2,
		ClustersSize:       40,
		ClusterOverlapping: 2,
	})
	index2.BuildIndexSharded()

	testinghelpers.Normalize(vectors, w)
	testinghelpers.PlotGraphHighLighted("vamana_test.png", index.edges, vectors, w, w, index.s_index, 5)
	testinghelpers.PlotGraphHighLighted("vamana_test_sharded.png", index2.edges, vectors, w, w, index2.s_index, 5)
}*/

func loadQueries(queries_size int) [][]float32 {
	f, err := os.Open("./sift/sift_queries.gob")
	if err != nil {
		panic(errors.Wrap(err, "Could not open truths file"))
	}
	defer f.Close()

	queries := make([][]float32, queries_size)
	cDec := gob.NewDecoder(f)
	err = cDec.Decode(&queries)
	if err != nil {
		panic(errors.Wrap(err, "Could not decode truths"))
	}
	return queries
}

func TestBigDataVamana(t *testing.T) {
	rand.Seed(0)
	dimensions := 128
	vectors_size := 100000
	queries_size := 1000
	before := time.Now()
	vectors, queries := testinghelpers.ReadVecs(vectors_size, dimensions, queries_size)
	if vectors == nil {
		panic("Error generating vectors")
	}
	fmt.Printf("generating data took %s\n", time.Since(before))

	paramsRs := []int{32, 70}
	paramsLs := []int{50, 125}
	results := make(map[string][][]float32, 0)
	for paramIndex := range paramsRs {
		paramR := paramsRs[paramIndex]
		paramL := paramsLs[paramIndex]
		paramAlpha := float32(1.2)
		before = time.Now()
		index := testinghelpers.BuildVamana(
			paramR,
			paramL,
			paramAlpha,
			func(ctx context.Context, id uint64) ([]float32, error) {
				return vectors[int(id)], nil
			},
			uint64(vectors_size),
			ssdhelpers.L2,
			"./data",
		)
		fmt.Printf("Index built in: %s\n", time.Since(before))
		//Ks := [3]int{1, 10, 100}
		Ks := []int{10}
		L := []int{100, 200, 300, 400, 500, 1000}
		for _, k := range Ks {
			fmt.Println("K\tL\trecall\t\tquerying")
			truths := testinghelpers.BuildTruths(queries_size, queries, vectors, k, ssdhelpers.L2)
			data := make([][]float32, len(L))
			for i, l := range L {
				index.SetL(l)
				var relevant uint64
				var retrieved int

				var querying time.Duration = 0
				for i := 0; i < len(queries); i++ {
					before = time.Now()
					results := index.SearchByVector(queries[i], k)
					querying += time.Since(before)
					retrieved += k
					relevant += testinghelpers.MatchesInLists(truths[i], results)
				}

				recall := float32(relevant) / float32(retrieved)
				queryingTime := float32(querying.Microseconds()) / 1000
				data[i] = []float32{queryingTime, recall}
				//fmt.Printf("%d\t%d\t%s\t%f\n", k, l, querying/1000, recall)
				fmt.Printf("{%f,%f},\n", float32(querying.Microseconds())/float32(1000), recall)
			}
			results[fmt.Sprintf("Vamana - K: %d (R: %d, L: %d, alpha:%.1f)", k, paramR, paramL, paramAlpha)] = data
		}
	}
	testinghelpers.ChartData("Recall Vs Latency", "", results, "index.html")
}

/*
func TestBigDataVamanaSharded(t *testing.T) {
	rand.Seed(0)
	dimensions := 128
	vectors_size := 100000
	queries_size := 1000
	before := time.Now()
	vectors, queries := testinghelpers.ReadVecs(vectors_size, dimensions, queries_size)
	if vectors == nil {
		panic("Error generating vectors")
	}
	fmt.Printf("generating data took %s\n", time.Since(before))

	paramsRs := []int{32, 70}
	paramsLs := []int{50, 125}
	results := make(map[string][][]float32, 0)
	for paramIndex := range paramsRs {
		paramR := paramsRs[paramIndex]
		paramL := paramsLs[paramIndex]
		paramAlpha := float32(1.2)
		before = time.Now()
		index := testinghelpers.BuildVamanaSharded(
			paramR,
			paramL,
			paramAlpha,
			func(ctx context.Context, id uint64) ([]float32, error) {
				return vectors[int(id)], nil
			},
			uint64(vectors_size),
			ssdhelpers.L2,
			"./data",
		)
		fmt.Printf("Index built in: %s\n", time.Since(before))
		//Ks := [3]int{1, 10, 100}
		Ks := []int{10}
		fmt.Println("K\tL\trecall\t\tquerying")
		L := [10]int{10, 20, 30, 40, 50, 100}

		for _, k := range Ks {
			truths := testinghelpers.BuildTruths(queries_size, queries, vectors, k, ssdhelpers.L2)
			data := make([][]float32, len(L))
			for i, l := range L {
				index.SetL(l)
				var relevant uint64
				var retrieved int

				var querying time.Duration = 0
				for i := 0; i < len(queries); i++ {
					before = time.Now()
					results := index.SearchByVector(queries[i], k)
					querying += time.Since(before)
					retrieved += k
					relevant += testinghelpers.MatchesInLists(truths[i], results)
				}

				recall := float32(relevant) / float32(retrieved)
				queryingTime := float32(querying.Microseconds()) / 1000
				data[i] = []float32{queryingTime, recall}
				fmt.Printf("%d\t%s\t%f\n", k, querying/1000, recall)
			}
			results[fmt.Sprintf("Vamana - K: %d (R: %d, L: %d, alpha:%.1f)", k, paramR, paramL, paramAlpha)] = data
		}
	}
	testinghelpers.ChartData("Recall Vs Latency", "", results, "index.html")
}*/

/*
func TestBigDataHNSW(t *testing.T) {
	rand.Seed(0)
	dimensions := 128
	vectors_size := 100000
	queries_size := 1000
	k := 100
	before := time.Now()
	vectors, queries := testinghelpers.ReadVecs(vectors_size, dimensions, queries_size)
	if vectors == nil {
		panic("Error generating vectors")
	}
	//testinghelpers.BuildTruths(queries_size, queries, vectors, k, euclidean_distance)
	truths := testinghelpers.LoadTruths(queries_size, k)
	fmt.Printf("generating data took %s\n", time.Since(before))

	efc := 128
	ef := 64
	maxN := 32

	index, _ := hnsw.New(hnsw.Config{
		RootPath:              "doesnt-matter-as-committlogger-is-mocked-out",
		ID:                    "recallbenchmark",
		MakeCommitLoggerThunk: hnsw.MakeNoopCommitLogger,
		DistanceProvider:      distancer.NewL2SquaredProvider(),
		VectorForIDThunk: func(ctx context.Context, id uint64) ([]float32, error) {
			return vectors[int(id)], nil
		},
	}, hnsw.UserConfig{
		MaxConnections: maxN,
		EFConstruction: efc,
		EF:             ef,
	})

	vectorIndex := index

	workerCount := runtime.GOMAXPROCS(0)
	jobsForWorker := make([][][]float32, workerCount)

	before = time.Now()
	for i, vec := range vectors {
		workerID := i % workerCount
		jobsForWorker[workerID] = append(jobsForWorker[workerID], vec)
	}

	wg := &sync.WaitGroup{}
	for workerID, jobs := range jobsForWorker {
		wg.Add(1)
		go func(workerID int, myJobs [][]float32) {
			defer wg.Done()
			for i, vec := range myJobs {
				originalIndex := (i * workerCount) + workerID
				err := vectorIndex.Add(uint64(originalIndex), vec)
				require.Nil(t, err)
			}
		}(workerID, jobs)
	}

	wg.Wait()
	indexing := time.Since(before)
	fmt.Printf("Indexing done in: %s\n", indexing)
	efs := []int{8, 16, 32, 64, 128, 256}
	fmt.Println("ef	recall	querying")
	for _, efSearch := range efs {
		index.UpdateUserConfig(hnsw.UserConfig{
			MaxConnections: maxN,
			EFConstruction: efc,
			EF:             efSearch,
		})
		var relevant uint64
		var retrieved int

		var querying time.Duration = 0
		before = time.Now()
		for i := 0; i < len(queries); i++ {
			before = time.Now()
			results, _, err := vectorIndex.SearchByVector(queries[i], k, nil)
			querying += time.Since(before)
			require.Nil(t, err)

			retrieved += k
			relevant += matchesInLists(truths[i], results)
		}

		recall := float32(relevant) / float32(retrieved)
		fmt.Printf("%d	%f	%s\n", efSearch, recall, querying/1000)
	}
}*/

func TestCharts(t *testing.T) {
	results := make(map[string][][]float32, 0)
	results["Vamana-100K (R: 32, L: 50, alpha:1.2)"] = [][]float32{
		{74.778999, 0.595000},
		{99.362000, 0.866500},
		{120.258003, 0.955600},
		{144.362000, 0.980400},
		{156.733994, 0.989000},
		{230.479996, 0.994000},
	}
	results["Vamana-100K (R: 70, L: 125, alpha:1.2)"] = [][]float32{
		{123.399002, 0.505000},
		{136.130005, 0.732600},
		{171.880005, 0.862000},
		{189.201996, 0.930800},
		{205.563004, 0.966900},
		{292.096985, 0.997800},
	}
	results["HNSW-100K (efC: 512, ef: 256, maxN: 128)"] = [][]float32{
		{123.772, 0.960200},
		{126.898, 0.974600},
		{203.975, 0.989800},
		{345.065, 0.997000},
		{634.041, 0.999200},
		{1079.155, 0.999700},
	}
	results["HNSW-100K (efC: 128, ef: 64, maxN: 32)"] = [][]float32{
		{74.86, 0.9500},
		{114.28, 0.9761},
		{151.50, 0.9852},
		{188.41, 0.9900},
		{223.06, 0.9927},
		{374.78, 0.9978},
	}
	results["Vamana Microsoft (R: 32, L: 50, alpha:1.2)"] = [][]float32{
		{345.83, 0.9500},
		{517.27, 0.9761},
		{675.10, 0.9852},
		{823.33, 0.9900},
		{964.15, 0.9927},
		{1588.55, 0.9978},
	}
	testinghelpers.ChartData("Recall vs Latency", "", results, "line-10.html")

	results = make(map[string][][]float32, 0)
	results["Vamana-100K (R: 32, L: 50, alpha:1.2)"] = [][]float32{
		{238.320007, 0.304020},
		{358.404999, 0.497330},
		{464.095001, 0.658860},
		{553.460999, 0.792190},
		{662.239014, 0.885430},
		{1045.656982, 0.995270},
	}
	results["Vamana-100K (R: 70, L: 125, alpha:1.2)"] = [][]float32{
		{309.812012, 0.203390},
		{417.647003, 0.318280},
		{530.294006, 0.412280},
		{626.781006, 0.494910},
		{711.590027, 0.571570},
		{1112.654053, 0.874120},
	}
	results["HNSW-100K (efC: 128, ef: 64, maxN: 32)"] = [][]float32{
		{633.732, 0.983720},
		{603.067, 0.983720},
		{556.734, 0.983720},
		{489.457, 0.983720},
		{745.091, 0.990590},
		{1087.982, 0.998640},
	}
	results["HNSW-100K (efC: 512, ef: 256, maxN: 128)"] = [][]float32{
		{755.518, 0.996870},
		{653.413, 0.996870},
		{648.089, 0.996870},
		{643.147, 0.996870},
		{780.003, 0.998520},
		{1338.566, 0.999820},
	}
	results["Vamana Microsoft (R: 32, L: 50, alpha:1.2)"] = [][]float32{
		{451.03, 0.9795},
		{735.23, 0.9955},
		{991.78, 0.9984},
		{1229.70, 0.9993},
		{1456.79, 0.9996},
		{2483.10, 0.9999},
	}
	testinghelpers.ChartData("100-Recall@100 vs Latency", "", results, "line-100.html")
}