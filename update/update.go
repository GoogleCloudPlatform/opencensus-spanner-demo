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

// Simulate updates by adding test data to a Spanner database
package update

import (
	"context"
	"errors"
	"fmt"
	"math/rand"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	log "github.com/GoogleCloudPlatform/oc-spannerlab/applog"
)

const (
	NOT_FOUND     = -1
	SPANNER_ERROR = -2
)

type AppError struct {
	Message string
	Code    int
}

func (e AppError) Error() string {
	return e.Message
}

func albumNotFound(albumId int64, albumTitle string) *AppError {
	msg := fmt.Sprintf("Singer-Album %d %s not found", albumId, albumTitle)
	return &AppError{msg, NOT_FOUND}
}

// Adds a singer-album with a random  album id, not checking for existence
// Returns: The id of the newly created album
func addAlbum(ctx context.Context, client *spanner.Client, singerId int64,
	albumTitle string) (*int64, error) {
	albumId := rand.Int63()
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context,
		txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: `INSERT Albums (SingerId, AlbumId, AlbumTitle) VALUES
            (@SingerId, @AlbumId, @AlbumTitle)`,
			Params: map[string]interface{}{
				"SingerId":   singerId,
				"AlbumId":    albumId,
				"AlbumTitle": albumTitle,
			},
		}
		rowCount, err := txn.Update(ctx, stmt)
		if err != nil {
			return err
		}
		log.Printf(ctx, "%d record(s) inserted.\n", rowCount)
		return nil
	})
	return &albumId, err
}

// Adds a singer and album first checking for the existence of the singer but
// not in the same transaction.
// Returns: The id of the singer, either existing or newly created
func AddAllNoTxn(ctx context.Context, client *spanner.Client,
	firstName, lastName, albumTitle string) (*int64, error) {
	singerId, e := getSingerId(ctx, client, nil, firstName, lastName)
	if e != nil && e.Code != NOT_FOUND {
		log.Printf(ctx, "Error looking up singer")
		return nil, errors.New(e.Message)
	}
	if e != nil && e.Code == NOT_FOUND {
		var err error
		singerId, err = addSinger(ctx, client, firstName, lastName)
		if err != nil {
			log.Printf(ctx, "Could not add singer")
			return nil, err
		}
	}
	var albumId *int64
	albumId, e = getAlbumId(ctx, client, nil, singerId, albumTitle)
	if e != nil && e.Code != NOT_FOUND {
		log.Printf(ctx, "Error looking up album")
		return nil, errors.New(e.Message)
	}
	if e != nil && e.Code == NOT_FOUND {
		var err error
		albumId, err = addAlbum(ctx, client, singerId, albumTitle)
		if err != nil {
			log.Printf(ctx, "Could not add album")
			return nil, err
		}
	}
	return albumId, nil
}

// Adds a singer and album first checking for existence within the transaction
// Return the album id created or that already existed
func AddAllTxn(ctx context.Context, client *spanner.Client,
	firstName, lastName, albumTitle string) (*int64, error) {
	var albumId *int64
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context,
		txn *spanner.ReadWriteTransaction) error {
		// adds the singer with given singerId and name
		addAlbum := func(singerId int64, albumTitle string) (*int64, error) {
			albumId := rand.Int63()
			stmt := spanner.Statement{
				SQL: `INSERT Albums (SingerId, AlbumId, AlbumTitle) VALUES
            (@SingerId, @AlbumId, @AlbumTitle)`,
				Params: map[string]interface{}{
					"SingerId":   singerId,
					"AlbumId":    albumId,
					"AlbumTitle": albumTitle,
				},
			}
			_, err := txn.Update(ctx, stmt)
			return &albumId, err
		}

		addSinger := func(firstName, lastName string) (int64, error) {
			singerId := rand.Int63()
			stmt := spanner.Statement{
				SQL: `INSERT Singers (SingerId, FirstName, LastName) VALUES
              (@SingerId, @FirstName, @LastName)`,
				Params: map[string]interface{}{
					"SingerId":  singerId,
					"FirstName": firstName,
					"LastName":  lastName,
				},
			}
			_, err := txn.Update(ctx, stmt)
			return singerId, err
		}

		singerId, e := getSingerId(ctx, client, txn, firstName, lastName)
		if e != nil && e.Code != NOT_FOUND {
			return errors.New(e.Message)
		}
		if e == nil {
			return nil
		}
		// The singer will be added only if they do not exist already
		var err error
		singerId, err = addSinger(firstName, lastName)
		if err != nil {
			return err
		}
		log.Printf(ctx, "Added singer %s %s in transaction", firstName, lastName)

		// Add album
		albumId, e = getAlbumId(ctx, client, txn, singerId, albumTitle)
		if e != nil && e.Code != NOT_FOUND {
			log.Printf(ctx, "Error looking up album")
			return errors.New(e.Message)
		}
		if e != nil && e.Code == NOT_FOUND {
			var err error
			albumId, err = addAlbum(singerId, albumTitle)
			if err != nil {
				log.Printf(ctx, "Could not add album")
				return err
			}
		}
		return nil
	})
	return albumId, err
}

// Adds a singer with a random id, not checking for the existence of the singer.
// Returns: The id of the newly created singer
func addSinger(ctx context.Context, client *spanner.Client,
	firstName, lastName string) (int64, error) {
	singerId := rand.Int63()
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context,
		txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: `INSERT Singers (SingerId, FirstName, LastName) VALUES
            (@SingerId, @FirstName, @LastName)`,
			Params: map[string]interface{}{
				"SingerId":  singerId,
				"FirstName": firstName,
				"LastName":  lastName,
			},
		}
		rowCount, err := txn.Update(ctx, stmt)
		if err != nil {
			return err
		}
		log.Printf(ctx, "%d record(s) inserted.\n", rowCount)
		return nil
	})
	return singerId, err
}

// Count the singers with a select query
func CountRows(client *spanner.Client,
	fieldName, tableName string) (int64, error) {
	ctx := context.Background()
	selectCount := fmt.Sprintf("SELECT COUNT(%s) FROM %s", fieldName, tableName)
	stmt := spanner.Statement{
		SQL: selectCount,
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return -1, errors.New("No results")
		}
		if err != nil {
			return -1, err
		}
		var count int64
		err = row.Columns(&count)
		if err != nil {
			return -1, err
		}
		return count, nil
	}
	return -1, errors.New("No results")
}

// Return the id, if the album with given singer and title is in the database
func getAlbumId(ctx context.Context, client *spanner.Client,
	txn *spanner.ReadWriteTransaction, singerId int64,
	albumTitle string) (*int64, *AppError) {
	stmt := spanner.Statement{
		SQL: `SELECT
            SingerId, AlbumId
          FROM Albums 
          WHERE 
            SingerId = @SingerId AND AlbumTitle = @AlbumTitle`,
		Params: map[string]interface{}{
			"SingerId":   singerId,
			"AlbumTitle": albumTitle,
		},
	}
	// Reuse transaction if not nil
	var iter *spanner.RowIterator
	if txn != nil {
		iter = txn.Query(ctx, stmt)
	} else {
		iter = client.Single().Query(ctx, stmt)
	}
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return nil, albumNotFound(singerId, albumTitle)
		}
		if err != nil {
			return nil, &AppError{err.Error(), SPANNER_ERROR}
		}
		var singerID int64
		if row.Columns(&singerID) != nil {
			log.Printf(ctx, "Failed to parse row")
		}
		var albumId int64
		if row.Columns(&albumId) != nil {
			log.Printf(ctx, "Failed to parse row")
		}
		return &albumId, nil
	}
	return nil, albumNotFound(singerId, albumTitle)
}

// If the singer is in the database then return the id.
// If a transaction is supplied then use it. Otherwise, create a new, single
// query transaction.
func getSingerId(ctx context.Context, client *spanner.Client,
	txn *spanner.ReadWriteTransaction,
	firstName, lastName string) (int64, *AppError) {
	stmt := spanner.Statement{
		SQL: `SELECT SingerId FROM Singers 
          WHERE FirstName = @FirstName AND LastName = @LastName`,
		Params: map[string]interface{}{
			"FirstName": firstName,
			"LastName":  lastName,
		},
	}
	// Reuse transaction if not nil
	var iter *spanner.RowIterator
	if txn != nil {
		iter = txn.Query(ctx, stmt)
	} else {
		iter = client.Single().Query(ctx, stmt)
	}
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			msg := fmt.Sprintf("Singer not found", firstName, lastName)
			return -1, &AppError{msg, NOT_FOUND}
		}
		if err != nil {
			return -1, &AppError{err.Error(), SPANNER_ERROR}
		}
		var singerID int64
		if row.Columns(&singerID) != nil {
			log.Printf(ctx, "Failed to parse row")
		}
		return singerID, nil
	}
	msg := fmt.Sprintf("Singer %s %s not found", firstName, lastName)
	return -1, &AppError{msg, NOT_FOUND}
}
