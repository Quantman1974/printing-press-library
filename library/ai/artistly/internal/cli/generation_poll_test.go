// Copyright 2026 Vincent Colombo and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import "testing"

func TestCorrelateSubmissionIgnoresOtherPrompts(t *testing.T) {
	tracked := map[int]struct{}{}
	designs := []Design{
		{ID: 10, PositivePrompt: "ours", Status: "completed", Images: `["https://cdn.example/a.png"]`},
		{ID: 11, PositivePrompt: "other session", Status: "completed", Images: `["https://cdn.example/b.png"]`},
		{ID: 12, PositivePrompt: "ours", Status: "completed", Images: `["https://cdn.example/c.png"]`},
	}
	done := correlateSubmission(designs, 10, "ours", 1, tracked)
	if len(done) != 1 || done[0].ID != 12 {
		t.Fatalf("correlated %+v, want design 12 only", idsOf(done))
	}
	if _, ok := tracked[11]; ok {
		t.Fatal("locked the other session's design id")
	}
}

func TestCorrelateSubmissionIgnoresOlderSamePrompt(t *testing.T) {
	tracked := map[int]struct{}{}
	designs := []Design{
		{ID: 5, PositivePrompt: "fox", Status: "completed", Images: `["https://cdn.example/old.png"]`},
		{ID: 9, PositivePrompt: "fox", Status: "processing"},
	}
	done := correlateSubmission(designs, 8, "fox", 1, tracked)
	if len(done) != 0 {
		t.Fatalf("got done %+v, want none while ours is still processing", idsOf(done))
	}
	if _, ok := tracked[5]; ok {
		t.Fatal("locked a pre-cursor design")
	}
	if _, ok := tracked[9]; !ok {
		t.Fatal("did not lock the in-flight matching id")
	}
}

func TestCorrelateSubmissionLocksFirstCandidates(t *testing.T) {
	tracked := map[int]struct{}{}
	first := []Design{
		{ID: 21, PositivePrompt: "same", Status: "processing"},
		{ID: 22, PositivePrompt: "same", Status: "processing"},
	}
	if done := correlateSubmission(first, 20, "same", 1, tracked); len(done) != 0 {
		t.Fatalf("first poll done=%+v, want none", idsOf(done))
	}
	if _, ok := tracked[21]; !ok {
		t.Fatal("did not lock the lowest matching id")
	}
	if _, ok := tracked[22]; ok {
		t.Fatal("locked a second same-prompt id beyond want")
	}

	later := []Design{
		{ID: 21, PositivePrompt: "same", Status: "processing"},
		{ID: 22, PositivePrompt: "same", Status: "completed", Images: `["https://cdn.example/other.png"]`},
	}
	if done := correlateSubmission(later, 20, "same", 1, tracked); len(done) != 0 {
		t.Fatalf("attributed the other session's finished design: %+v", idsOf(done))
	}

	oursDone := []Design{
		{ID: 21, PositivePrompt: "same", Status: "completed", Images: `["https://cdn.example/ours.png"]`},
		{ID: 22, PositivePrompt: "same", Status: "completed", Images: `["https://cdn.example/other.png"]`},
	}
	done := correlateSubmission(oursDone, 20, "same", 1, tracked)
	if len(done) != 1 || done[0].ID != 21 {
		t.Fatalf("got %+v, want locked id 21", idsOf(done))
	}
}

func TestDecideQuotaSubmit(t *testing.T) {
	cases := []struct {
		name     string
		q        quotaInfo
		quantity int
		want     quotaSubmitDecision
	}{
		{"slots and budget open", quotaInfo{ConcurrentCount: 1, ConcurrentLimit: 4, TodaysCount: 10}, 1, quotaSubmitNow},
		{"unlimited concurrent when limit is zero", quotaInfo{ConcurrentCount: 9, ConcurrentLimit: 0, TodaysCount: 10}, 2, quotaSubmitNow},
		{"wait when concurrent is full", quotaInfo{ConcurrentCount: 4, ConcurrentLimit: 4, TodaysCount: 10}, 1, quotaWaitConcurrent},
		{"wait when quantity would overflow slots", quotaInfo{ConcurrentCount: 3, ConcurrentLimit: 4, TodaysCount: 10}, 2, quotaWaitConcurrent},
		{"refuse when daily cap would be exceeded", quotaInfo{ConcurrentCount: 0, ConcurrentLimit: 4, TodaysCount: 399}, 2, quotaDailyExhausted},
		{"refuse at the daily ceiling", quotaInfo{ConcurrentCount: 0, ConcurrentLimit: 4, TodaysCount: 400}, 1, quotaDailyExhausted},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := decideQuotaSubmit(tc.q, tc.quantity); got != tc.want {
				t.Fatalf("decideQuotaSubmit(%+v, %d)=%d, want %d", tc.q, tc.quantity, got, tc.want)
			}
		})
	}
}

func idsOf(designs []Design) []int {
	ids := make([]int, len(designs))
	for i, d := range designs {
		ids[i] = d.ID
	}
	return ids
}
