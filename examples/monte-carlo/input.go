package main

import (
	"github.com/ReconfigureIO/fixed"
	"github.com/ReconfigureIO/math/rand"
)

type Ret struct {
	avg         fixed.Int26_6
	zero_trials uint32
}

func RetInit() Ret {
	return Ret{avg: 0, zero_trials: 0}
}

type Param struct {
	s0         fixed.Int26_6
	drift      fixed.Int26_6
	volatility fixed.Int26_6
	k          fixed.Int26_6
	days       uint32
}

func Deserialize(inputChan <-chan uint32, outputChan chan<- Param) {
	for {
		ps := [5]uint32{}
		for i := 0; i < 5; i++ {
			ps[i] = <-inputChan
		}

		outputChan <- Param{
			s0:         fixed.Int26_6(ps[0]),
			drift:      fixed.Int26_6(ps[1]),
			volatility: fixed.Int26_6(ps[2]),
			k:          fixed.Int26_6(ps[3]),
			days:       ps[4],
		}
	}
}

func Serialize(inputChan <-chan Ret, outputChan chan<- uint32) {
	for {
		ret := <-inputChan
		outputChan <- uint32(ret.avg)
		outputChan <- uint32(ret.zero_trials)
	}
}

func Rands(seed uint32, out chan<- fixed.Int26_6) {
	r := rand.New(seed)
	r.Normals(out)
}

func StocWalk(rands <-chan fixed.Int26_6, p Param) Ret {
	// An iterative model, calculates some integral over 'period'
	// explanation of + 1 following
	intermediates := make(chan fixed.Int26_6, 1)

	go func() {
		d := p.drift + fixed.I26(1)
		v := p.volatility
		for i := uint16(p.days); i != 0; i-- {
			W := <-rands
			intermediates <- d + W.Mul(v)
		}

	}()

	s0 := p.s0
	for i := uint16(p.days); i != 0; i-- {
		interm := <-intermediates
		// The original formulation of this is as follows
		// s0 = s0 + d.Mul(s0) + W.Mul(v.Mul(s0))
		// we can factor out s0 to give the following
		// s0 = s0.Mul(1 + d + W.Mul(v))
		// 1 + d is constant, so refactor out
		// hence the following
		s0 = s0.Mul(interm)
	}

	k := p.k

	var ret Ret
	if s0 > k {
		// The multiplication with exp(-r/t_*days) to be calculated by the host
		// payoff = payoff * fixed.Int26_6(2<<uint32(-p.r/p.t_*p.days))
		ret = Ret{avg: s0 - k}
	} else {
		ret = Ret{zero_trials: 1}
	}

	return ret
}

func Payoff(a Ret, b Ret) Ret {

	return Ret{avg: a.avg + b.avg,
		zero_trials: a.zero_trials + b.zero_trials}
}

func BenchmarkStocWalk(n uint32) {
	rs := make(chan fixed.Int26_6, 1)
	go Rands(42, rs)
	StocWalk(rs, Param{days: n})
}
