// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Test application to demonstrate identification of latency problems.
package main

// [START spannerlab_imports]
import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"

	"cloud.google.com/go/spanner"

	"contrib.go.opencensus.io/exporter/stackdriver"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"

	log "github.com/GoogleCloudPlatform/oc-spannerlab/applog"
	"github.com/GoogleCloudPlatform/oc-spannerlab/query"
	"github.com/GoogleCloudPlatform/oc-spannerlab/testdata"
	"github.com/GoogleCloudPlatform/oc-spannerlab/update"
)

// [END spannerlab_imports]

// Initialize OpenCensus
// [START spannerlab_initoc]
func initOC(project string) *stackdriver.Exporter {
	se, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID:    project,
		MetricPrefix: "spanner-oc-test",
	})
	if err != nil {
		ctx := context.Background()
		log.Fatalf(ctx, "Failed to create exporter: %v", err)
	}
	trace.RegisterExporter(se)
	view.RegisterExporter(se)
	if err := view.Register(ocgrpc.DefaultClientViews...); err != nil {
		ctx := context.Background()
		log.Fatalf(ctx, "Failed to register gRPC default client views: %v", err)
	}
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
	return se
}

// [END spannerlab_initoc]

// Run the query tests
func runQueryTest(client *spanner.Client) {
	ctx := context.Background()
	buf := bytes.NewBufferString("")
	query.QueryAlbums(ctx, client, buf)
}

// Run a simulation with a mix of queries and adds
func runSimulation(client *spanner.Client, iterations int) {
	fmt.Printf("Running simulation with %d iterations\n", iterations)
	ctx := context.Background()
	for i := 0; i < iterations; i++ {
		if i%10 == 0 {
			fmt.Printf("Iteration %d\n", i)
		}
		action := testdata.NextUserAction()
		log.Printf(ctx, "Next user action is %d.\n", action)
		buf := bytes.NewBufferString("")
		switch action {
		case testdata.ACTION_QUERY_ALBUMS:
			query.QueryAlbums(ctx, client, buf)
		case testdata.ACTION_QUERY_LIMIT:
			query.QueryAlbumsLimit(ctx, client, buf)
		case testdata.ACTION_QUERY_SINGERS_FIRST:
			query.QuerySingersFirstName(ctx, client, buf)
		case testdata.ACTION_QUERY_SINGERS_LAST:
			query.QuerySingersLastName(ctx, client, buf)
		case testdata.ACTION_JOIN_SINGER_ALBUM:
			query.JoinSingerAlbum(ctx, client, buf)
		case testdata.ACTION_ADD_ALL_TXN:
			data := testdata.RandomData()
			ctx, span := trace.StartSpan(ctx, "add-album-single-txns")
			_, err := update.AddAllNoTxn(ctx, client, data.FirstName, data.LastName,
				data.AlbumTitle)
			if err != nil {
				log.Printf(ctx, "Error adding singer %v", err)
			}
			span.End()
		case testdata.ACTION_ADD_SINGLE_TXNS:
			data := testdata.RandomData()
			ctx, span := trace.StartSpan(ctx, "add-album-all-one-txn")
			_, err := update.AddAllTxn(ctx, client, data.FirstName,
				data.LastName, data.AlbumTitle)
			if err != nil {
				log.Printf(ctx, "Error adding singer in transaction %v", err)
			}
			span.End()
		}
	}
}

// Run the update tests
func runUpdateSmallTxns(client *spanner.Client) {
	ctx := context.Background()
	data := testdata.RandomData()
	ctx, span := trace.StartSpan(ctx, "add-album-single-txns")
	albumId, err := update.AddAllNoTxn(ctx, client, data.FirstName,
		data.LastName, data.AlbumTitle)
	if err != nil {
		log.Errorf(ctx, "Error adding singer %v", err)
	} else {
		log.Printf(ctx, "runTest not in transaction %d", albumId)
	}
	sNum, err := update.CountRows(client, "SingerId", "Singers")
	if err != nil {
		log.Errorf(ctx, "Error querying singers %v", err)
	} else {
		log.Printf(ctx, "Total number of singers %d", sNum)
	}
	aNum, err := update.CountRows(client, "AlbumId", "Albums")
	if err != nil {
		log.Errorf(ctx, "Error querying singers %v", err)
	} else {
		log.Printf(ctx, "Total number of albums %d", aNum)
	}
	span.End()
}

// Run the update tests
func runUpdateBigTxn(client *spanner.Client) {
	ctx := context.Background()
	data := testdata.RandomData()
	ctx, span := trace.StartSpan(ctx, "add-album-all-one-txn")
	albumId, err := update.AddAllTxn(ctx, client, data.FirstName,
		data.LastName, data.AlbumTitle)
	if err != nil {
		log.Printf(ctx, "Error adding singer in transaction %v", err)
	} else {
		log.Printf(ctx, "runTest in transaction %d", albumId)
	}
	sNum, err := update.CountRows(client, "SingerId", "Singers")
	if err != nil {
		log.Printf(ctx, "Error querying singers %v", err)
	} else {
		log.Printf(ctx, "Total number of singers %d", sNum)
	}
	aNum, err := update.CountRows(client, "AlbumId", "Albums")
	if err != nil {
		log.Printf(ctx, "Error querying singers %v", err)
	} else {
		log.Printf(ctx, "Total number of albums %d", aNum)
	}
	span.End()
}

// Entry point for the application
func main() {
	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	var projPtr = flag.String("project", project, "The project id")
	var instance = flag.String("instance", "test-instance",
		"The Spanner instance")
	var db = flag.String("database", "test", "The Spanner database name")
	var command = flag.String("command", "simulation",
		"One of [update_big_txn | update_small_txns | query_test | simulation]")
	var iterations = flag.Int("iterations", 100,
		"Number of iterations to run for the 'simulation' command")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr,
			`Usage: oc-spannerlab --project=$GOOGLE_CLOUD_PROJECT \
  --instance=$SPANNER_INSTANCE \
  --database=$DATABASE \
  --command=COMMAND \
  [--iterations=iterations]
`)
	}
	flag.Parse()
	if *projPtr == "" {
		fmt.Println("project flag must have a value")
		flag.Usage()
		os.Exit(2)
	}

	log.Initialize(project)
	defer log.Close()

	databaseName := fmt.Sprintf("projects/%s/instances/%s/databases/%s", *projPtr,
		*instance, *db)

	// Initialize OpenCensus
	se := initOC(project)
	defer se.Flush()

	// Initialize Spanner client
	ctx := context.Background()
	client, err := spanner.NewClient(ctx, databaseName)
	if err != nil {
		fmt.Printf("Failed to create Spanner client %v", err)
		os.Exit(1)
	}
	defer client.Close()

	if *command == "update_big_txn" {
		runUpdateBigTxn(client)
	} else if *command == "update_small_txns" {
		runUpdateSmallTxns(client)
	} else if *command == "query_test" {
		runQueryTest(client)
	} else if *command == "simulation" {
		runSimulation(client, *iterations)
	} else {
		fmt.Printf("Command %s not understood", command)
		flag.Usage()
		os.Exit(2)
	}
}
