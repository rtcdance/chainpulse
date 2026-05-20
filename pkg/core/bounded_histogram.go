package core

import "github.com/rtcdance/chainpulse/pkg/histogram"

type boundedHistogram = histogram.Histogram

func newBoundedHistogram(capacity int) *boundedHistogram {
	return histogram.New(capacity)
}
