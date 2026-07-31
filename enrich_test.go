package bookstats_test

import (
	"testing"

	"github.com/allank/bookstats"
)

func TestEnrich_MultiLevelTwoSidedBook(t *testing.T) {
	bids := []bookstats.Order{
		{Price: 100, Volume: 1},
		{Price: 99, Volume: 2},
	}
	asks := []bookstats.Order{
		{Price: 101, Volume: 1},
		{Price: 102, Volume: 3},
	}

	got := bookstats.Enrich(bids, asks)

	wantMid := 100.5
	if got.Mid != wantMid {
		t.Fatalf("Mid = %v, want %v", got.Mid, wantMid)
	}

	wantBids := []bookstats.Row{
		{Order: bookstats.Order{Price: 100, Volume: 1}, Notional: 100, CumVolume: 1, CumNotional: 100, SlippageBps: (100 - wantMid) / wantMid * 10000},
		{Order: bookstats.Order{Price: 99, Volume: 2}, Notional: 198, CumVolume: 3, CumNotional: 298, SlippageBps: (99 - wantMid) / wantMid * 10000},
	}
	if len(got.Bids) != len(wantBids) {
		t.Fatalf("len(Bids) = %d, want %d", len(got.Bids), len(wantBids))
	}
	for i, want := range wantBids {
		if got.Bids[i] != want {
			t.Errorf("Bids[%d] = %+v, want %+v", i, got.Bids[i], want)
		}
	}

	wantAsks := []bookstats.Row{
		{Order: bookstats.Order{Price: 101, Volume: 1}, Notional: 101, CumVolume: 1, CumNotional: 101, SlippageBps: (101 - wantMid) / wantMid * 10000},
		{Order: bookstats.Order{Price: 102, Volume: 3}, Notional: 306, CumVolume: 4, CumNotional: 407, SlippageBps: (102 - wantMid) / wantMid * 10000},
	}
	if len(got.Asks) != len(wantAsks) {
		t.Fatalf("len(Asks) = %d, want %d", len(got.Asks), len(wantAsks))
	}
	for i, want := range wantAsks {
		if got.Asks[i] != want {
			t.Errorf("Asks[%d] = %+v, want %+v", i, got.Asks[i], want)
		}
	}
}

func TestEnrich_UnsortedInputIsOrderedFromBestPriceOutward(t *testing.T) {
	// Best bid is the highest price, best ask is the lowest — regardless of
	// the order the caller supplies them in.
	bids := []bookstats.Order{
		{Price: 99, Volume: 2},
		{Price: 100, Volume: 1},
	}
	asks := []bookstats.Order{
		{Price: 102, Volume: 3},
		{Price: 101, Volume: 1},
	}

	got := bookstats.Enrich(bids, asks)

	if got.Bids[0].Price != 100 || got.Bids[1].Price != 99 {
		t.Errorf("Bids not walked from best (highest) price outward: %+v", got.Bids)
	}
	if got.Asks[0].Price != 101 || got.Asks[1].Price != 102 {
		t.Errorf("Asks not walked from best (lowest) price outward: %+v", got.Asks)
	}
}

func TestEnrich_EmptyBook(t *testing.T) {
	got := bookstats.Enrich(nil, nil)

	if got.Mid != 0 {
		t.Errorf("Mid = %v, want 0", got.Mid)
	}
	if len(got.Bids) != 0 {
		t.Errorf("len(Bids) = %d, want 0", len(got.Bids))
	}
	if len(got.Asks) != 0 {
		t.Errorf("len(Asks) = %d, want 0", len(got.Asks))
	}
}

func TestEnrich_OneSidedBook_BidsOnly(t *testing.T) {
	bids := []bookstats.Order{
		{Price: 100, Volume: 1},
		{Price: 99, Volume: 1},
	}

	got := bookstats.Enrich(bids, nil)

	if got.Mid != 0 {
		t.Errorf("Mid = %v, want 0", got.Mid)
	}
	if len(got.Asks) != 0 {
		t.Errorf("len(Asks) = %d, want 0", len(got.Asks))
	}
	if len(got.Bids) != 2 {
		t.Fatalf("len(Bids) = %d, want 2", len(got.Bids))
	}
	for _, row := range got.Bids {
		if row.SlippageBps != 0 {
			t.Errorf("SlippageBps = %v, want 0 for row %+v", row.SlippageBps, row)
		}
	}
	if got.Bids[0].CumVolume != 1 {
		t.Errorf("Bids[0].CumVolume = %v, want 1", got.Bids[0].CumVolume)
	}
	if got.Bids[1].CumVolume != 2 {
		t.Errorf("Bids[1].CumVolume = %v, want 2", got.Bids[1].CumVolume)
	}
}

func TestEnrich_OneSidedBook_AsksOnly(t *testing.T) {
	asks := []bookstats.Order{{Price: 101, Volume: 2}}

	got := bookstats.Enrich(nil, asks)

	if got.Mid != 0 {
		t.Errorf("Mid = %v, want 0", got.Mid)
	}
	if len(got.Bids) != 0 {
		t.Errorf("len(Bids) = %d, want 0", len(got.Bids))
	}
	if len(got.Asks) != 1 {
		t.Fatalf("len(Asks) = %d, want 1", len(got.Asks))
	}
	if got.Asks[0].SlippageBps != 0 {
		t.Errorf("SlippageBps = %v, want 0", got.Asks[0].SlippageBps)
	}
	if got.Asks[0].Notional != 202 {
		t.Errorf("Notional = %v, want 202", got.Asks[0].Notional)
	}
}

func TestEnrich_TiesAtSamePrice(t *testing.T) {
	bids := []bookstats.Order{
		{Price: 100, Volume: 1},
		{Price: 100, Volume: 2},
	}

	got := bookstats.Enrich(bids, nil)

	if len(got.Bids) != 2 {
		t.Fatalf("len(Bids) = %d, want 2", len(got.Bids))
	}
	if got.Bids[0].CumVolume != 1 {
		t.Errorf("Bids[0].CumVolume = %v, want 1", got.Bids[0].CumVolume)
	}
	if got.Bids[1].CumVolume != 3 {
		t.Errorf("Bids[1].CumVolume = %v, want 3", got.Bids[1].CumVolume)
	}
	if got.Bids[1].CumNotional != 300 {
		t.Errorf("Bids[1].CumNotional = %v, want 300", got.Bids[1].CumNotional)
	}
}
