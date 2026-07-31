package bookstats

import "sort"

// Enrich computes notional, cumulative volume, cumulative notional, and
// slippage in basis points for both sides of a raw Order Book, walking
// outward from the best price on each side regardless of input order. Mid
// is (best_bid+best_ask)/2, computed internally; if either side is empty,
// Mid and every row's SlippageBps are 0 rather than erroring.
func Enrich(bids, asks []Order) EnrichedBook {
	sortedBids := sortedCopy(bids, true)
	sortedAsks := sortedCopy(asks, false)

	var mid float64
	if len(sortedBids) > 0 && len(sortedAsks) > 0 {
		mid = (sortedBids[0].Price + sortedAsks[0].Price) / 2
	}

	return EnrichedBook{
		Bids: enrichSide(sortedBids, mid),
		Asks: enrichSide(sortedAsks, mid),
		Mid:  mid,
	}
}

// sortedCopy returns a sorted copy of orders, descending by price if
// descending is true (best bid first) or ascending otherwise (best ask
// first). The caller's slice is never mutated.
func sortedCopy(orders []Order, descending bool) []Order {
	out := make([]Order, len(orders))
	copy(out, orders)
	sort.SliceStable(out, func(i, j int) bool {
		if descending {
			return out[i].Price > out[j].Price
		}
		return out[i].Price < out[j].Price
	})
	return out
}

// enrichSide computes Row values for one side, walking outward from the
// best price (orders[0]).
func enrichSide(orders []Order, mid float64) []Row {
	rows := make([]Row, len(orders))
	var cumVolume, cumNotional float64
	for i, o := range orders {
		notional := o.Price * o.Volume
		cumVolume += o.Volume
		cumNotional += notional

		var slippageBps float64
		if mid != 0 {
			slippageBps = (o.Price - mid) / mid * 10000
		}

		rows[i] = Row{
			Order:       o,
			Notional:    notional,
			CumVolume:   cumVolume,
			CumNotional: cumNotional,
			SlippageBps: slippageBps,
		}
	}
	return rows
}
