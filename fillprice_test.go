package bookstats_test

import (
	"testing"

	"github.com/allank/bookstats"
)

func fillPriceFixture() bookstats.EnrichedBook {
	bids := []bookstats.Order{
		{Price: 100, Volume: 1}, // CumNotional 100
		{Price: 99, Volume: 2},  // CumNotional 298
	}
	asks := []bookstats.Order{
		{Price: 101, Volume: 1}, // CumNotional 101
		{Price: 102, Volume: 3}, // CumNotional 407
	}
	return bookstats.Enrich(bids, asks)
}

func TestFillPrice_ExactBoundary(t *testing.T) {
	book := fillPriceFixture()

	price, ok := book.FillPrice(bookstats.AskSide, book.Asks[0].CumNotional)

	if !ok {
		t.Fatalf("ok = false, want true")
	}
	if price != book.Asks[0].Price {
		t.Errorf("price = %v, want %v", price, book.Asks[0].Price)
	}
}

func TestFillPrice_CrossesMidLevel(t *testing.T) {
	book := fillPriceFixture()

	// Between Bids[0].CumNotional (100) and Bids[1].CumNotional (298).
	target := (book.Bids[0].CumNotional + book.Bids[1].CumNotional) / 2

	price, ok := book.FillPrice(bookstats.BidSide, target)

	if !ok {
		t.Fatalf("ok = false, want true")
	}
	if price != book.Bids[1].Price {
		t.Errorf("price = %v, want %v (first row whose CumNotional reaches target)", price, book.Bids[1].Price)
	}
}

func TestFillPrice_InsufficientDepth(t *testing.T) {
	book := fillPriceFixture()

	totalAskNotional := book.Asks[len(book.Asks)-1].CumNotional

	price, ok := book.FillPrice(bookstats.AskSide, totalAskNotional+1)

	if ok {
		t.Fatalf("ok = true, want false (book too thin); got price %v", price)
	}
}

func TestFillPrice_EmptySide(t *testing.T) {
	book := bookstats.Enrich(nil, []bookstats.Order{{Price: 101, Volume: 1}})

	_, ok := book.FillPrice(bookstats.BidSide, 1)

	if ok {
		t.Fatalf("ok = true, want false for an empty side")
	}
}

func TestFillPrice_UnaffectedByDegenerateMid(t *testing.T) {
	// One-sided book: Mid is 0 (see Enrich), but FillPrice must still work
	// off CumNotional.
	book := bookstats.Enrich(nil, []bookstats.Order{
		{Price: 101, Volume: 1},
		{Price: 102, Volume: 1},
	})
	if book.Mid != 0 {
		t.Fatalf("fixture invalid: Mid = %v, want 0", book.Mid)
	}

	price, ok := book.FillPrice(bookstats.AskSide, book.Asks[0].CumNotional)

	if !ok || price != book.Asks[0].Price {
		t.Errorf("FillPrice = (%v, %v), want (%v, true)", price, ok, book.Asks[0].Price)
	}
}
