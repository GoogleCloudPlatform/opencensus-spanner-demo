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
package query

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/spanner"
	"go.opencensus.io/trace"
	"google.golang.org/api/iterator"

	log "github.com/GoogleCloudPlatform/opencensus-spanner-demo/applog"
)

// Queries albums and singers with a join
func JoinSingerAlbum(ctx context.Context, client *spanner.Client,
	w io.Writer) {
	ctx, span := trace.StartSpan(ctx, "join-singer-album")
	defer span.End()
	q := `SELECT s.SingerId, s.FirstName, a.AlbumTitle
				FROM Singers AS s
				JOIN Albums AS a ON s.SingerId = a.SingerId;`
	err := querySingers(span, ctx, client, w, q)
	if err != nil {
		log.Errorf(ctx, "JoinSingerAlbum Error %v", err)
	}
}

// Queries albums in the Spanner database
func QueryAlbums(ctx context.Context, client *spanner.Client, w io.Writer) {
	// [START spannerlab_query_albums_span]
	ctx, span := trace.StartSpan(ctx, "query-albums")
	defer span.End()
	// [END spannerlab_query_albums_span]
	q := `SELECT SingerId, AlbumId, AlbumTitle FROM Albums`
	err := queryAlbums(ctx, client, w, q)
	if err != nil {
		log.Errorf(ctx, "Error querying albums %v for query %s", err, q)
	}
}

// Queries albums in the Spanner database with a limit
func QueryAlbumsLimit(ctx context.Context, client *spanner.Client,
	w io.Writer) {
	ctx, span := trace.StartSpan(ctx, "query-limit")
	defer span.End()
	q := `SELECT SingerId, AlbumId, AlbumTitle FROM Albums LIMIT 10`
	err := queryAlbums(ctx, client, w, q)
	if err != nil {
		log.Printf(ctx, "QueryLimit Error %v", err)
	}
}

// Execute a query with no parameters
func queryAlbums(ctx context.Context, client *spanner.Client, w io.Writer,
	q string) error {
	// [START querylbums_ReadOnlyTransaction]
	ro := client.ReadOnlyTransaction()
	defer ro.Close()
	// [END querylbums_ReadOnlyTransaction]
	stmt := spanner.Statement{SQL: q}
	iter := ro.Query(ctx, stmt)
	defer iter.Stop()
	counter := 0
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		var singerID int64
		var albumID int64
		var albumTitle string
		if err := row.Columns(&singerID, &albumID, &albumTitle); err != nil {
			return err
		}
		counter++
		fmt.Fprintf(w, "%d %d %s", singerID, albumID, albumTitle)
	}
	log.Printf(ctx, "queryAlbums: %d results for query: %s", counter, q)
	return nil
}

// Queries singers by first name (has an index)
func QuerySingersFirstName(ctx context.Context, client *spanner.Client,
	w io.Writer) {
	ctx, span := trace.StartSpan(ctx, "query-singers-first")
	defer span.End()
	q := `SELECT SingerId, FirstName, LastName FROM Singers
				WHERE FirstName = 'Captain'`
	err := querySingers(span, ctx, client, w, q)
	if err != nil {
		log.Printf(ctx, "QuerySingersFirstName Error %v", err)
	}
}

// Queries singers by last name (has an index)
func QuerySingersLastName(ctx context.Context, client *spanner.Client,
	w io.Writer) {
	ctx, span := trace.StartSpan(ctx, "query-singers-last")
	defer span.End()
	q := `SELECT SingerId, FirstName, LastName
				FROM Singers@{FORCE_INDEX=SingersByLastName}
				WHERE LastName = 'Zero'`
	err := querySingers(span, ctx, client, w, q)
	if err != nil {
		log.Printf(ctx, "QuerySingersLastName Error %v", err)
	}
}

// Execute a query with no parameters
func querySingers(span *trace.Span, ctx context.Context,
	client *spanner.Client, w io.Writer, q string) error {
	ro := client.ReadOnlyTransaction()
	defer ro.Close()
	stmt := spanner.Statement{SQL: q}
	iter := ro.Query(ctx, stmt)
	defer iter.Stop()
	counter := 0
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		var singerID int64
		var firstName string
		var lastName string
		if err := row.Columns(&singerID, &firstName, &lastName); err != nil {
			return err
		}
		counter++
		fmt.Fprintf(w, "%d %s %s", singerID, firstName, lastName)
	}
	log.Printf(ctx, "querySingers # results: %d for query: %s", counter, q)
	return nil
}
