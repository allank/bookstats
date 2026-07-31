# bookstats

`bookstats` enriches a raw Order Book — a list of bid and ask price/volume
pairs — with the columns needed to actually answer depth questions: how
much value is available at each price, how far each level sits from mid in
basis points, and what price fills a given order size. It then lets you
bucket that enriched view into fixed-width bands along whichever column
matches the question you're asking.

It exists because reading raw price/volume rows makes questions like these
awkward:

- "How much can I move within 10 bps of mid?"
- "What does depth look like, banded by dollar/rand size?"
- "Can this customer fill a 100,000 USDC order, and at what price?"

`bookstats` is a plain Go library — no I/O, no network calls, no
persistence, no dependency on any rendering or exchange-client package. It
takes `[]Order` in and returns plain structs out, so any consumer (a TUI,
a web service, a CLI) can convert its own Order Book representation into
`bookstats.Order` at the boundary.

## Installation

```sh
go get github.com/allank/bookstats
```

## Concepts

- **Order Book**: the aggregated set of buy (**Bids**) and sell (**Asks**)
  Orders for a trading pair.
- **Order**: a single price/volume pair — the fundamental unit of an Order
  Book, representing the total volume resting at one price.
- **Bid** / **Ask**: which collection an Order sits in. `Order` itself
  carries no field marking it as one or the other — bid-ness/ask-ness is
  purely positional (`EnrichedBook.Bids` vs `EnrichedBook.Asks`).
- **Side**: which side of the book an operation applies to (`BidSide` or
  `AskSide`).

## Quick example

```go
package main

import (
	"fmt"

	"github.com/allank/bookstats"
)

func main() {
	bids := []bookstats.Order{
		{Price: 100, Volume: 1},
		{Price: 99, Volume: 2},
	}
	asks := []bookstats.Order{
		{Price: 101, Volume: 1},
		{Price: 102, Volume: 3},
	}

	book := bookstats.Enrich(bids, asks)

	fmt.Printf("mid: %.2f\n", book.Mid) // mid: 100.50

	for _, row := range book.Asks {
		fmt.Printf("price=%.2f notional=%.2f cum_notional=%.2f slippage_bps=%.1f\n",
			row.Price, row.Notional, row.CumNotional, row.SlippageBps)
	}
	// price=101.00 notional=101.00 cum_notional=101.00 slippage_bps=49.8
	// price=102.00 notional=306.00 cum_notional=407.00 slippage_bps=149.3

	// "Can a customer fill a 250 notional ask order, and at what price?"
	if price, ok := book.FillPrice(bookstats.AskSide, 250); ok {
		fmt.Printf("fills at %.2f\n", price) // fills at 102.00
	}

	// Band the ask side by cumulative notional, in buckets of 100.
	for _, band := range bookstats.Bands(book.Asks, bookstats.ByCumNotional, 100) {
		fmt.Printf("[%.0f,%.0f) volume=%.2f notional=%.2f count=%d\n",
			band.Lower, band.Upper, band.Volume, band.Notional, band.Count)
	}
	// [100,200) volume=1.00 notional=101.00 count=1
	// [400,500) volume=3.00 notional=306.00 count=1
}
```

## API reference

### `type Order`

```go
type Order struct {
	Price  float64
	Volume float64
}
```

The raw input unit: the total `Volume` resting at `Price`. This is the only
type you construct by hand — everything else is computed by `Enrich`.

### `type Row`

```go
type Row struct {
	Order
	Notional    float64
	CumVolume   float64
	CumNotional float64
	SlippageBps float64
}
```

An enriched Order Book level — an `Order` (embedded, so `row.Price` and
`row.Volume` are available directly) plus four computed columns. See
[Enrichment formulas](#enrichment-formulas) below for exactly how each is
computed.

### `type EnrichedBook`

```go
type EnrichedBook struct {
	Bids []Row
	Asks []Row
	Mid  float64
}
```

The output of `Enrich`. `Bids` and `Asks` are each ordered walking outward
from the best price on that side (highest-first for Bids, lowest-first for
Asks) — regardless of what order you passed into `Enrich`. `Mid` is `0` if
either side was empty (see [Edge cases](#edge-cases)).

### `type Side`

```go
type Side int

const (
	BidSide Side = iota
	AskSide
)
```

Selects which side of an `EnrichedBook` an operation applies to — currently
used by `FillPrice`.

### `func Enrich`

```go
func Enrich(bids, asks []Order) EnrichedBook
```

Computes the full enriched view for both sides of a raw Order Book. Mid is
always computed internally from the best bid and best ask — there is no way
to supply a different reference price (e.g. last trade, index price).

`bids` and `asks` don't need to be pre-sorted; `Enrich` sorts internally
(a defensive copy — your input slices are never mutated) so the "walking
outward from best price" guarantee holds regardless of the order you pass
in.

### `type ByColumn` and `type Band`

```go
type ByColumn int

const (
	ByCumNotional ByColumn = iota
	ByCumVolume
	BySlippageBps
)

type Band struct {
	Lower    float64
	Upper    float64
	Volume   float64
	Notional float64
	Count    int
}
```

`ByColumn` selects which enriched column `Bands` buckets on. `Band` is one
resulting bucket: `Volume`, `Notional`, and `Count` are **totals within**
`[Lower, Upper)` of the chosen column — not running/cumulative totals. If
you want a running total, sum the bands up to the one you care about.

### `func Bands`

```go
func Bands(rows []Row, by ByColumn, width float64) []Band
```

Buckets `rows` into fixed-width bands of `width`, keyed on the **absolute
value** of whichever column `by` selects. Only bands containing at least
one row are returned — there's no zero-filling of empty buckets in
between. `width` must be positive.

Pass one side at a time — `book.Bids` or `book.Asks` — since `rows` is
assumed to arrive ordered with **non-decreasing magnitude** of the chosen
column, which is exactly what `EnrichedBook.Bids`/`.Asks` already guarantee
(walking outward from best price, `CumVolume`, `CumNotional`, and
`|SlippageBps|` are all monotonically non-decreasing). Banding on magnitude
rather than the raw signed value matters because `SlippageBps` is negative
on the bid side — without `abs()`, bid bands and ask bands would live on
different, incomparable boundaries.

Pick the column to match the question:

| Question | `by` |
|---|---|
| "How much depth exists at each dollar/rand risk size?" | `ByCumNotional` |
| "How much depth exists at each base-asset size?" | `ByCumVolume` |
| "How much can move within X bps of mid?" | `BySlippageBps` |

### `func (EnrichedBook) FillPrice`

```go
func (b EnrichedBook) FillPrice(side Side, notional float64) (price float64, ok bool)
```

Returns the price at which cumulative notional on `side` first reaches or
exceeds `notional` — "what price fills an order of this size". `ok` is
`false` when that side can't fill the requested size (insufficient depth,
or an empty side); **always check `ok`** before trusting `price`, since a
defaulted `0` would otherwise be indistinguishable from a real fill at
price zero. `FillPrice` operates purely on `CumNotional`, so it works
correctly even when `Mid` is `0` (see [Edge cases](#edge-cases)).

## Enrichment formulas

For each `Row` at index `i`, walking outward from the best price on its
side (`i = 0` is the best bid or best ask):

| Column | Formula |
|---|---|
| `Notional` | `Price × Volume` |
| `CumVolume` | `Σ Volume[0..i]` (running sum, best price outward) |
| `CumNotional` | `Σ Notional[0..i]` (running sum, best price outward) |
| `Mid` | `(BestBidPrice + BestAskPrice) / 2` |
| `SlippageBps` | `(Price − Mid) / Mid × 10000` |

Worked example, from the [Quick example](#quick-example) above — asks
`[{101, 1}, {102, 3}]` against a mid of `100.5`:

```
Mid = (100 + 101) / 2 = 100.5

Row 0: Price=101, Volume=1
  Notional     = 101 × 1        = 101
  CumVolume    = 1
  CumNotional  = 101
  SlippageBps  = (101 − 100.5) / 100.5 × 10000 ≈ 49.8

Row 1: Price=102, Volume=3
  Notional     = 102 × 3        = 306
  CumVolume    = 1 + 3          = 4
  CumNotional  = 101 + 306      = 407
  SlippageBps  = (102 − 100.5) / 100.5 × 10000 ≈ 149.3
```

## Banding formula

For a row's chosen column value `v` (`CumNotional`, `CumVolume`, or
`SlippageBps`) and a band `width`:

```
index = floor(|v| / width)
Lower = index × width
Upper = Lower + width
```

Every row whose `|v|` falls in `[Lower, Upper)` contributes its `Volume`
and `Notional` to that band's totals, and increments `Count` by one.

Worked example — banding the asks above (`CumNotional` values `101` and
`407`) with `width = 100`:

```
Row 0: CumNotional=101 → index = floor(101/100) = 1 → band [100, 200)
Row 1: CumNotional=407 → index = floor(407/100) = 4 → band [400, 500)
```

Each row lands in its own band here since they're far apart; rows whose
column values fall in the same bucket accumulate into a single `Band`
(their `Volume`/`Notional` sum, `Count` increments per row).

## Fill-price lookup

`FillPrice(side, notional)` walks `side`'s rows in order and returns the
first row where:

```
CumNotional[i] ≥ notional
```

returning that row's `Price` and `ok = true`. If no row satisfies this
(the side's total `CumNotional` never reaches `notional`), it returns
`(0, false)`.

## Edge cases

`Enrich` never panics or returns an error, even for degenerate input:

| Input | `Mid` | `SlippageBps` | `Notional` / `CumVolume` / `CumNotional` |
|---|---|---|---|
| Both sides empty | `0` | n/a (no rows) | n/a (no rows) |
| One side empty | `0` | `0` on every row of the non-empty side | Computed normally |
| Both sides present | `(bestBid + bestAsk) / 2` | Computed normally | Computed normally |

`SlippageBps` is forced to `0` rather than dividing by a zero `Mid` — the
value is defined as meaningless in that case, not left to produce `NaN` or
`Inf`. `FillPrice` is unaffected either way, since it only reads
`CumNotional`.

## Design notes

- **`float64` throughout.** Order Book data is already `float64` at the
  source (matching a typical exchange client's Order type), so no
  decimal/fixed-point type is used.
- **No dependency on any rendering or exchange-client package.** Consumers
  convert their own Order representation into `bookstats.Order` at the
  boundary.
- **`Enrich` sorts internally**, so callers don't need to pre-sort bids
  descending / asks ascending themselves — the "walking outward from best
  price" guarantee holds regardless of input order.
- **`Bands` assumes pre-sorted input** (as `Enrich`'s output already is)
  rather than re-sorting itself, since re-deriving an ordering guarantee
  `Enrich` already provides would be redundant work on every call.

## Testing

All tests are black-box (`package bookstats_test`), exercising only the
exported API — there's no internal seam to fake or mock since this is a
pure, dependency-free library. Fixtures are hand-written synthetic Order
Books, not captured exchange data, so edge cases (empty book, one-sided
book, ties at the same price, uneven band widths) are exact and
reproducible.

```sh
go test ./...
```
