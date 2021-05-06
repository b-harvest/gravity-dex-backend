package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type Stats struct {
	ActiveAddresses map[string]struct{}
	ByHour          map[string]*HourlyStats
	mux             sync.Mutex
}

func NewStats() *Stats {
	return &Stats{
		ActiveAddresses: make(map[string]struct{}),
		ByHour:          make(map[string]*HourlyStats),
	}
}

func (s *Stats) NumActiveAddresses() int {
	return len(s.ActiveAddresses)
}

func (s *Stats) NumDeposits() int {
	cnt := 0
	for _, hs := range s.ByHour {
		for _, n := range hs.NumDepositsByPoolID {
			cnt += n
		}
	}
	return cnt
}

func (s *Stats) NumDepositsSince(t time.Time) int {
	t = t.UTC().Truncate(time.Hour)
	now := time.Now()
	cnt := 0
	for !t.After(now) {
		hs, ok := s.ByHour[HourKey(t)]
		if ok {
			for _, n := range hs.NumDepositsByPoolID {
				cnt += n
			}
		}
		t = t.Add(time.Hour)
	}
	return cnt
}

func (s *Stats) NumSwaps() int {
	cnt := 0
	for _, hs := range s.ByHour {
		for _, n := range hs.NumSwapsByPoolID {
			cnt += n
		}
	}
	return cnt
}

func (s *Stats) NumSwapsSince(t time.Time) int {
	t = t.UTC().Truncate(time.Hour)
	now := time.Now()
	cnt := 0
	for !t.After(now) {
		hs, ok := s.ByHour[HourKey(t)]
		if ok {
			for _, n := range hs.NumSwapsByPoolID {
				cnt += n
			}
		}
		t = t.Add(time.Hour)
	}
	return cnt
}

func (s *Stats) OfferCoins() Coins {
	cs := make(Coins)
	for _, hs := range s.ByHour {
		for _, v := range hs.SwapVolumeByPoolID {
			cs.Add(v.OfferCoins)
		}
	}
	return cs
}

func (s *Stats) DemandCoins() Coins {
	cs := make(Coins)
	for _, hs := range s.ByHour {
		for _, v := range hs.SwapVolumeByPoolID {
			cs.Add(v.DemandCoins)
		}
	}
	return cs
}

func (s *Stats) OfferCoinsSince(t time.Time) Coins {
	t = t.UTC().Truncate(time.Hour)
	now := time.Now()
	cs := make(Coins)
	for !t.After(now) {
		hs, ok := s.ByHour[HourKey(t)]
		if ok {
			for _, v := range hs.SwapVolumeByPoolID {
				cs.Add(v.OfferCoins)
			}
		}
		t = t.Add(time.Hour)
	}
	return cs
}

func (s *Stats) DemandCoinsSince(t time.Time) Coins {
	t = t.UTC().Truncate(time.Hour)
	now := time.Now()
	cs := make(Coins)
	for !t.After(now) {
		hs, ok := s.ByHour[HourKey(t)]
		if ok {
			for _, v := range hs.SwapVolumeByPoolID {
				cs.Add(v.DemandCoins)
			}
		}
		t = t.Add(time.Hour)
	}
	return cs
}

func (s *Stats) AddActiveAddress(addr string) {
	s.ActiveAddresses[addr] = struct{}{}
}

func (s *Stats) AddNumSwaps(hourKey string, poolID uint64, amount int) {
	s.HourlyStats(hourKey).AddNumSwaps(poolID, amount)
}

func (s *Stats) AddNumDeposits(hourKey string, poolID uint64, amount int) {
	s.HourlyStats(hourKey).AddNumDeposits(poolID, amount)
}

func (s *Stats) AddOfferCoins(hourKey string, poolID uint64, coins Coins) {
	s.HourlyStats(hourKey).SwapVolume(poolID).AddOfferCoins(coins)
}

func (s *Stats) AddDemandCoins(hourKey string, poolID uint64, coins Coins) {
	s.HourlyStats(hourKey).SwapVolume(poolID).AddDemandCoins(coins)
}

type HourlyStats struct {
	NumSwapsByPoolID    map[uint64]int
	NumDepositsByPoolID map[uint64]int
	SwapVolumeByPoolID  map[uint64]*SwapVolume
}

func (s *Stats) HourlyStats(hourKey string) *HourlyStats {
	hs, ok := s.ByHour[hourKey]
	if !ok {
		hs = &HourlyStats{
			NumSwapsByPoolID:    make(map[uint64]int),
			NumDepositsByPoolID: make(map[uint64]int),
			SwapVolumeByPoolID:  make(map[uint64]*SwapVolume),
		}
		s.ByHour[hourKey] = hs
	}
	return hs
}

func (hs *HourlyStats) AddNumSwaps(poolID uint64, amount int) {
	hs.NumSwapsByPoolID[poolID] += amount
}

func (hs *HourlyStats) AddNumDeposits(poolID uint64, amount int) {
	hs.NumDepositsByPoolID[poolID] += amount
}

type SwapVolume struct {
	OfferCoins  Coins
	DemandCoins Coins
}

func (hs *HourlyStats) SwapVolume(poolID uint64) *SwapVolume {
	v, ok := hs.SwapVolumeByPoolID[poolID]
	if !ok {
		v = &SwapVolume{
			OfferCoins:  make(Coins),
			DemandCoins: make(Coins),
		}
		hs.SwapVolumeByPoolID[poolID] = v
	}
	return v
}

func (v *SwapVolume) AddOfferCoins(coins Coins) {
	v.OfferCoins.Add(coins)
}

func (v *SwapVolume) AddDemandCoins(coins Coins) {
	v.DemandCoins.Add(coins)
}

type Coins map[string]int64

func (cs Coins) String() string {
	var denoms []string
	for denom := range cs {
		denoms = append(denoms, denom)
	}
	sort.Strings(denoms)
	var ss []string
	for _, denom := range denoms {
		ss = append(ss, fmt.Sprintf("%d%s", cs[denom], denom))
	}
	return strings.Join(ss, ",")
}

func (cs Coins) Add(coins Coins) {
	for denom, amount := range coins {
		cs[denom] += amount
	}
}

func HourKey(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:00:00")
}
