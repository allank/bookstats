package bookstats

import "math"

// ByColumn selects which enriched column Bands buckets on.
type ByColumn int

const (
	ByCumNotional ByColumn = iota
	ByCumVolume
	BySlippageBps
)

// Band is the per-band aggregate within [Lower, Upper) of the chosen
// column's magnitude. Volume/Notional/Count are totals within this band,
// not running totals — callers wanting a running total sum bands up to
// that point.
type Band struct {
	Lower    float64
	Upper    float64
	Volume   float64
	Notional float64
	Count    int
}

// Bands buckets rows into fixed-width bands of the given column's absolute
// value — SlippageBps is negative on the bid side, so banding on magnitude
// keeps bids and asks comparable on the same boundaries. rows must be
// ordered with non-decreasing magnitude of the chosen column, as Enrich's
// Bids/Asks already are (walking outward from best price). Only bands
// containing at least one row are returned. width must be positive.
func Bands(rows []Row, by ByColumn, width float64) []Band {
	var bands []Band
	lastIdx := 0

	for i, row := range rows {
		idx := int(math.Floor(math.Abs(columnValue(row, by)) / width))

		if i == 0 || idx != lastIdx {
			bands = append(bands, Band{Lower: float64(idx) * width, Upper: float64(idx+1) * width})
			lastIdx = idx
		}

		band := &bands[len(bands)-1]
		band.Volume += row.Volume
		band.Notional += row.Notional
		band.Count++
	}

	return bands
}

func columnValue(row Row, by ByColumn) float64 {
	switch by {
	case ByCumVolume:
		return row.CumVolume
	case BySlippageBps:
		return row.SlippageBps
	default:
		return row.CumNotional
	}
}
