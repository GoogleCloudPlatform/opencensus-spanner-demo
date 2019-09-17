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

// Package to generate test data

package testdata

import (
	"fmt"
	"math/rand"
	"time"
)

const (
	ACTION_QUERY_ALBUMS Action = iota + 1
	ACTION_QUERY_LIMIT
	ACTION_QUERY_SINGERS_FIRST
	ACTION_QUERY_SINGERS_LAST
	ACTION_JOIN_SINGER_ALBUM
	ACTION_ADD_ALL_TXN
	ACTION_ADD_SINGLE_TXNS
)

var ACTIONS = [...]Action{
	ACTION_QUERY_ALBUMS,
	ACTION_QUERY_LIMIT,
	ACTION_QUERY_SINGERS_FIRST,
	ACTION_QUERY_SINGERS_LAST,
	ACTION_JOIN_SINGER_ALBUM,
	ACTION_ADD_ALL_TXN,
	ACTION_ADD_SINGLE_TXNS}

type Action int

type SingerAlbum struct {
	FirstName, LastName, AlbumTitle string
}

func init() {
	rand.Seed(int64(time.Now().UnixNano()))
}

func (a Action) String() string {
	names := map[Action]string{
		ACTION_QUERY_ALBUMS:        "QueryAlbums",
		ACTION_QUERY_LIMIT:         "QueryLimit",
		ACTION_QUERY_SINGERS_FIRST: "QuerySingersFirstName",
		ACTION_QUERY_SINGERS_LAST:  "QuerySingersLastName",
		ACTION_JOIN_SINGER_ALBUM:   "JoinSingerAlbum",
		ACTION_ADD_ALL_TXN:         "AddAllInBigTransaction",
		ACTION_ADD_SINGLE_TXNS:     "AddEachInSingleTransactions",
	}
	if name, ok := names[a]; ok {
		return name
	} else {
		return "unknown"
	}
}

func NextUserAction() Action {
	r := rand.Intn(len(ACTIONS))
	return ACTIONS[r]
}

// Generate a random name for a singer
func RandomData() SingerAlbum {
	rank := []string{"Private", "Brigadier", "Sergeant", "Captain", "Commander",
		"Chief", "Lieutenant", "Officer", "First Officer", "Major",
		"General", "Five Star General", "Admiral", "Rear Admiral",
		"Vice General"}
	initial := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K",
		"L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X",
		"Y", "Z"}
	surname := []string{"Ryan", "General", "Major", "Zero", "Supreme",
		"Petty officer", "Governor", "In Charge", "Blunder", "Chaos"}
	generation := []string{"Junior", "II", "III", "IV", "V", "VI", "VII",
		"VIII", "IX", "X", "XI", "XII", "XIII", "XIV", "XV", "XVI",
		"XVII", "XVIII", "XIX", "XX"}
	part1 := []string{"Smoke", "Mist", "Rain", "Fog", "Thunder", "Lightening",
		"Frost", "Dew", "Snow", "Shadows", "Water", "Grass", "Trees",
		"Dust", "Wind", "Breeze", "Trash", "Graffiti", "Writing"}
	part2 := []string{"Water", "River", "Plains", "Road", "Mountain", "Hills",
		"Sea", "Bay", "Forest", "Highway", "Wall", "Blackboard"}
	m := rand.Int63n(int64(len(rank)))
	n := rand.Int63n(int64(len(initial)))
	p := rand.Int63n(int64(len(surname)))
	q := rand.Int63n(int64(len(generation)))
	r := rand.Int63n(int64(len(part1)))
	s := rand.Int63n(int64(len(part2)))
	firstName := fmt.Sprintf("%s %s", rank[m], initial[n])
	lastName := fmt.Sprintf("%s %s", surname[p], generation[q])
	albumTitle := fmt.Sprintf("%s on the %s", part1[r], part2[s])
	return SingerAlbum{firstName, lastName, albumTitle}
}
