package bookstats_test

import (
	"testing"

	"github.com/allank/bookstats"
)

// askFixture returns four enriched ask rows via a real Enrich call, so
// fixture values are exactly what the implementation produces rather than
// hand-computed floats that could drift by a ULP from independently
// recomputed expectations.
func askFixture() []bookstats.Row {
	bids := []bookstats.Order{{Price: 100, Volume: 1}}
	asks := []bookstats.Order{
		{Price: 101, Volume: 1},
		{Price: 102, Volume: 2},
		{Price: 110, Volume: 5},
		{Price: 110.5, Volume: 0.3},
	}
	return bookstats.Enrich(bids, asks).Asks
}

func TestBands_ByCumNotional(t *testing.T) {
	asks := askFixture()

	got := bookstats.Bands(asks, bookstats.ByCumNotional, 100)

	want := []bookstats.Band{
		{Lower: 100, Upper: 200, Volume: asks[0].Volume, Notional: asks[0].Notional, Count: 1},
		{Lower: 300, Upper: 400, Volume: asks[1].Volume, Notional: asks[1].Notional, Count: 1},
		{Lower: 800, Upper: 900, Volume: asks[2].Volume + asks[3].Volume, Notional: asks[2].Notional + asks[3].Notional, Count: 2},
	}
	assertBands(t, got, want)

	// width=100 doesn't evenly divide the data's range: the last band's
	// Upper (900) extends beyond the actual max CumNotional in the fixture.
	if asks[3].CumNotional >= want[len(want)-1].Upper {
		t.Fatalf("fixture invalid: last row's CumNotional %v should be < %v", asks[3].CumNotional, want[len(want)-1].Upper)
	}
}

func TestBands_ByCumVolume(t *testing.T) {
	asks := askFixture()

	got := bookstats.Bands(asks, bookstats.ByCumVolume, 3)

	want := []bookstats.Band{
		{Lower: 0, Upper: 3, Volume: asks[0].Volume, Notional: asks[0].Notional, Count: 1},
		{Lower: 3, Upper: 6, Volume: asks[1].Volume, Notional: asks[1].Notional, Count: 1},
		{Lower: 6, Upper: 9, Volume: asks[2].Volume + asks[3].Volume, Notional: asks[2].Notional + asks[3].Notional, Count: 2},
	}
	assertBands(t, got, want)
}

func TestBands_BySlippageBps(t *testing.T) {
	asks := askFixture()

	got := bookstats.Bands(asks, bookstats.BySlippageBps, 100)

	want := []bookstats.Band{
		{Lower: 0, Upper: 100, Volume: asks[0].Volume, Notional: asks[0].Notional, Count: 1},
		{Lower: 100, Upper: 200, Volume: asks[1].Volume, Notional: asks[1].Notional, Count: 1},
		{Lower: 900, Upper: 1000, Volume: asks[2].Volume + asks[3].Volume, Notional: asks[2].Notional + asks[3].Notional, Count: 2},
	}
	assertBands(t, got, want)
}

func TestBands_BySlippageBps_BidsBandOnMagnitude(t *testing.T) {
	// Bids sit below mid, so SlippageBps is negative. Banding must use its
	// magnitude, not the raw signed value, so bids and asks band the same
	// way — otherwise every bid band boundary would be negative.
	bids := []bookstats.Order{
		{Price: 99, Volume: 1},
		{Price: 98, Volume: 1},
	}
	asks := []bookstats.Order{{Price: 101, Volume: 1}}

	enriched := bookstats.Enrich(bids, asks)
	got := bookstats.Bands(enriched.Bids, bookstats.BySlippageBps, 100)

	if len(got) == 0 {
		t.Fatalf("expected at least one band")
	}
	for _, band := range got {
		if band.Lower < 0 || band.Upper < 0 {
			t.Errorf("negative band boundary for negative SlippageBps input: %+v", band)
		}
	}
}

func TestBands_BoundaryIsHalfOpen(t *testing.T) {
	rows := []bookstats.Row{
		{Order: bookstats.Order{Price: 1, Volume: 1}, Notional: 1, CumNotional: 200},
	}

	got := bookstats.Bands(rows, bookstats.ByCumNotional, 100)

	want := []bookstats.Band{{Lower: 200, Upper: 300, Volume: 1, Notional: 1, Count: 1}}
	assertBands(t, got, want)
}

func TestBands_EmptyRows(t *testing.T) {
	got := bookstats.Bands(nil, bookstats.ByCumNotional, 100)
	if len(got) != 0 {
		t.Errorf("len(Bands) = %d, want 0", len(got))
	}
}

func assertBands(t *testing.T, got, want []bookstats.Band) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len(Bands) = %d, want %d\ngot:  %+v\nwant: %+v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Bands[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}
