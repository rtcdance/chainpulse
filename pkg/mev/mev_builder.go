package mev

import (
	"math"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/histogram"
)

var knownBuilderAddresses = map[common.Address]string{
	common.HexToAddress("0x95222290DD7278Aa3Ddd389Cc1E1d165CC4BAfe5"): "Flashbots",
	common.HexToAddress("0x388C818CA8B9251b393131C08a736A67ccB19297"): "Builder0x69",
	common.HexToAddress("0x690B9A9E9aa1C9dB991C7721a92d351Db4FaC990"): "Rsync Builder",
	common.HexToAddress("0xA31F52Ee7eFdd5547b3Dd3b1F27ad1Fc0B5De790"): "Beaver Build",
	common.HexToAddress("0x4838B106FCe9647Bdf1E7877BF73cE8B0BAD5f97"): "BloxRoute",
	common.HexToAddress("0x9964f8e2e988C8dB6Ba7C5DDa4a1C1e0E6953c43"): "Eden Network",
	common.HexToAddress("0x3e7D8F0eA0e9E55D0E9bB55DFf6D6C7bF9B3A8d1"): "Manifold",
	common.HexToAddress("0x7c1F9Ea6B1e3E3261eEFa4244e256eF3D6c04792"): "BoostRB",
	common.HexToAddress("0x4779699B4D1E23c2235B507c2D82FC61ee5eCD4e"): "Flashbots: Genesis",
	common.HexToAddress("0xDAfea21D4367A26E6290DAa0ae0C0fC01c1911d1"): "Flashbots: Megabuilder",
}

func DetectBlockBuilder(block *blockchain.Block) *blockchain.BlockBuilder {
	if block == nil {
		return nil
	}

	addr := block.Miner
	if addr == (common.Address{}) {
		return nil
	}

	name, known := knownBuilderAddresses[addr]
	if !known {
		return nil
	}

	return &blockchain.BlockBuilder{
		BuilderName:    name,
		BuilderAddress: addr,
		IsMevBoost:     true,
	}
}

func IsMevBoostBlock(block *blockchain.Block) bool {
	return DetectBlockBuilder(block) != nil
}

func GetKnownBuilderNames() map[string]string {
	result := make(map[string]string)
	for _, name := range knownBuilderAddresses {
		result[name] = name
	}
	return result
}

type BuilderReputation struct {
	mu sync.RWMutex

	builders map[common.Address]*builderStats

	relayLatencies map[string]*histogram.Histogram

	maxBuilders  int
	latencyCapMs float64
}

type builderStats struct {
	name             string
	blocksProposed   uint64
	blocksMissed     uint64
	totalBlockValue  float64
	avgGasUsedPct    float64
	emptyBlocks      uint64
	lastSeenBlockNum uint64
	lastSeenTime     time.Time
}

func NewBuilderReputation(maxBuilders int, latencyCapMs float64) *BuilderReputation {
	if maxBuilders <= 0 {
		maxBuilders = 100
	}
	if latencyCapMs <= 0 {
		latencyCapMs = 10000
	}
	return &BuilderReputation{
		builders:       make(map[common.Address]*builderStats),
		relayLatencies: make(map[string]*histogram.Histogram),
		maxBuilders:    maxBuilders,
		latencyCapMs:   latencyCapMs,
	}
}

func (br *BuilderReputation) RecordBlock(builderAddr common.Address, builderName string, blockNum uint64, blockValueEth float64, gasUsedPct float64) {
	br.mu.Lock()
	defer br.mu.Unlock()

	stats, ok := br.builders[builderAddr]
	if !ok {
		if len(br.builders) >= br.maxBuilders {
			var oldestAddr common.Address
			var oldestTime time.Time
			for addr, s := range br.builders {
				if oldestTime.IsZero() || s.lastSeenTime.Before(oldestTime) {
					oldestAddr = addr
					oldestTime = s.lastSeenTime
				}
			}
			delete(br.builders, oldestAddr)
		}
		stats = &builderStats{name: builderName}
		br.builders[builderAddr] = stats
	}

	stats.blocksProposed++
	stats.totalBlockValue += blockValueEth

	if stats.blocksProposed == 1 {
		stats.avgGasUsedPct = gasUsedPct
	} else {
		stats.avgGasUsedPct = stats.avgGasUsedPct + (gasUsedPct-stats.avgGasUsedPct)/float64(stats.blocksProposed)
	}

	if gasUsedPct < 5.0 {
		stats.emptyBlocks++
	}

	stats.lastSeenBlockNum = blockNum
	stats.lastSeenTime = time.Now()
}

func (br *BuilderReputation) RecordMissedSlot(builderAddr common.Address) {
	br.mu.Lock()
	defer br.mu.Unlock()

	if stats, ok := br.builders[builderAddr]; ok {
		stats.blocksMissed++
	}
}

func (br *BuilderReputation) RecordRelayLatency(relayURL string, latencyMs float64) {
	br.mu.Lock()
	defer br.mu.Unlock()

	if latencyMs > br.latencyCapMs {
		latencyMs = br.latencyCapMs
	}

	h, ok := br.relayLatencies[relayURL]
	if !ok {
		h = histogram.New(256)
		br.relayLatencies[relayURL] = h
	}
	h.Record(latencyMs)
}

type BuilderMetrics struct {
	Address        common.Address `json:"address"`
	Name           string         `json:"name"`
	BlocksProposed uint64         `json:"blocks_proposed"`
	BlocksMissed   uint64         `json:"blocks_missed"`
	MissRate       float64        `json:"miss_rate"`
	TotalValueEth  float64        `json:"total_value_eth"`
	AvgValueEth    float64        `json:"avg_value_eth"`
	AvgGasUsedPct  float64        `json:"avg_gas_used_pct"`
	EmptyBlockRate float64        `json:"empty_block_rate"`
	LastSeenBlock  uint64         `json:"last_seen_block"`
	LastSeenTime   time.Time      `json:"last_seen_time"`
}

func (br *BuilderReputation) GetBuilderMetrics(addr common.Address) *BuilderMetrics {
	br.mu.RLock()
	defer br.mu.RUnlock()

	stats, ok := br.builders[addr]
	if !ok {
		return nil
	}

	return br.toMetrics(addr, stats)
}

func (br *BuilderReputation) AllBuilderMetrics() []BuilderMetrics {
	br.mu.RLock()
	defer br.mu.RUnlock()

	result := make([]BuilderMetrics, 0, len(br.builders))
	for addr, stats := range br.builders {
		result = append(result, *br.toMetrics(addr, stats))
	}
	return result
}

func (br *BuilderReputation) toMetrics(addr common.Address, s *builderStats) *BuilderMetrics {
	var missRate, avgValue, emptyRate float64
	if s.blocksProposed > 0 {
		missRate = float64(s.blocksMissed) / float64(s.blocksProposed+s.blocksMissed)
		avgValue = s.totalBlockValue / float64(s.blocksProposed)
		emptyRate = float64(s.emptyBlocks) / float64(s.blocksProposed)
	}

	return &BuilderMetrics{
		Address:        addr,
		Name:           s.name,
		BlocksProposed: s.blocksProposed,
		BlocksMissed:   s.blocksMissed,
		MissRate:       math.Round(missRate*10000) / 10000,
		TotalValueEth:  math.Round(s.totalBlockValue*1000) / 1000,
		AvgValueEth:    math.Round(avgValue*1000) / 1000,
		AvgGasUsedPct:  math.Round(s.avgGasUsedPct*100) / 100,
		EmptyBlockRate: math.Round(emptyRate*10000) / 10000,
		LastSeenBlock:  s.lastSeenBlockNum,
		LastSeenTime:   s.lastSeenTime,
	}
}

type RelayHealth struct {
	URL         string  `json:"url"`
	SampleCount int     `json:"sample_count"`
	LatencyP50  float64 `json:"latency_p50_ms"`
	LatencyP90  float64 `json:"latency_p90_ms"`
	LatencyP99  float64 `json:"latency_p99_ms"`
	IsHealthy   bool    `json:"is_healthy"`
}

func (br *BuilderReputation) GetRelayHealth(relayURL string) *RelayHealth {
	br.mu.RLock()
	defer br.mu.RUnlock()

	h, ok := br.relayLatencies[relayURL]
	if !ok {
		return &RelayHealth{URL: relayURL, IsHealthy: false}
	}

	p50 := h.Percentile(50)
	p90 := h.Percentile(90)
	p99 := h.Percentile(99)

	return &RelayHealth{
		URL:         relayURL,
		SampleCount: h.Count(),
		LatencyP50:  math.Round(p50*100) / 100,
		LatencyP90:  math.Round(p90*100) / 100,
		LatencyP99:  math.Round(p99*100) / 100,
		IsHealthy:   p50 < 500 && p99 < 2000,
	}
}

func (br *BuilderReputation) TopBuilders(n int) []BuilderMetrics {
	all := br.AllBuilderMetrics()
	if n > len(all) {
		n = len(all)
	}

	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			if all[j].BlocksProposed > all[i].BlocksProposed {
				all[i], all[j] = all[j], all[i]
			}
		}
	}

	return all[:n]
}

func (br *BuilderReputation) BuilderConcentrationRatio(topN int) float64 {
	br.mu.RLock()
	defer br.mu.RUnlock()

	var totalBlocks uint64
	for _, s := range br.builders {
		totalBlocks += s.blocksProposed
	}
	if totalBlocks == 0 {
		return 0
	}

	topBuilders := br.TopBuilders(topN)
	var topBlocks uint64
	for _, b := range topBuilders {
		topBlocks += b.BlocksProposed
	}

	return math.Round(float64(topBlocks)/float64(totalBlocks)*10000) / 10000
}
