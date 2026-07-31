// Package bookstats enriches raw Order Book data (price/volume pairs) with
// notional value, cumulative volume, cumulative notional, and slippage in
// basis points from mid, and supports banding and fill-price lookups on top
// of that enriched view.
package bookstats

// Order is a single price/volume pair — the fundamental unit of an Order
// Book, representing the total volume resting at one price.
type Order struct {
	Price  float64
	Volume float64
}

// Row is an enriched Order Book level.
type Row struct {
	Order
	Notional    float64
	CumVolume   float64
	CumNotional float64
	SlippageBps float64
}

// EnrichedBook holds enriched Bids and Asks, plus the mid price used to
// compute SlippageBps. Mid is 0 if either side is empty.
type EnrichedBook struct {
	Bids []Row
	Asks []Row
	Mid  float64
}

// Side identifies which side of an Order Book an operation applies to.
type Side int

const (
	BidSide Side = iota
	AskSide
)
