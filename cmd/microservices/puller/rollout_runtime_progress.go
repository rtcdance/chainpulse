package main

import (
	"sync/atomic"
	"time"
)

type pullerLoopRuntimeProgress struct {
	pollCount      atomic.Int64
	lastPollUnix   atomic.Int64
	observedBlock  atomic.Int64
	processedBlock atomic.Int64
}

type pullerLoopRuntimeProgressSnapshot struct {
	PollCount      int64
	LastPollUnix   int64
	ObservedBlock  int64
	ProcessedBlock int64
}

func (p *pullerLoopRuntimeProgress) recordPoll(now time.Time) {
	if p == nil {
		return
	}
	p.pollCount.Add(1)
	p.lastPollUnix.Store(now.Unix())
}

func (p *pullerLoopRuntimeProgress) recordObservedBlock(blockNumber int64) {
	if p == nil || blockNumber <= 0 {
		return
	}
	p.observedBlock.Store(blockNumber)
}

func (p *pullerLoopRuntimeProgress) recordProcessedBlock(blockNumber int64) {
	if p == nil || blockNumber <= 0 {
		return
	}
	p.processedBlock.Store(blockNumber)
}

func (p *pullerLoopRuntimeProgress) snapshot() pullerLoopRuntimeProgressSnapshot {
	if p == nil {
		return pullerLoopRuntimeProgressSnapshot{}
	}
	return pullerLoopRuntimeProgressSnapshot{
		PollCount:      p.pollCount.Load(),
		LastPollUnix:   p.lastPollUnix.Load(),
		ObservedBlock:  p.observedBlock.Load(),
		ProcessedBlock: p.processedBlock.Load(),
	}
}
