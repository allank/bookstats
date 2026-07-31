package bookstats

// FillPrice returns the price at which cumulative notional on the given
// side first reaches or exceeds notional. ok is false if the side can't
// fill that size (insufficient depth, or an empty side) — callers must
// check ok before trusting price. Operates purely on CumNotional, so it's
// unaffected by a degenerate (0) Mid.
func (b EnrichedBook) FillPrice(side Side, notional float64) (price float64, ok bool) {
	rows := b.Bids
	if side == AskSide {
		rows = b.Asks
	}
	for _, row := range rows {
		if row.CumNotional >= notional {
			return row.Price, true
		}
	}
	return 0, false
}
