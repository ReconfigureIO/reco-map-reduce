package main

import (
	"github.com/ReconfigureIO/fixed"
	"github.com/ReconfigureIO/math/rand"
)

// Ret is our return structure, which is the output of our mapper.
// In order to simplify the implementation, we'll just calculate the
// sum FPGA side, and divide by length to get the mean on the CPU.  A
// more robust implementation would use a streaming mean algorithm.
type Ret struct {
	avg         fixed.Int26_6 // Our running avg. Note that we'll divide this by length CPU side
	zero_trials uint32        // The number of zero trials we got
}

// RetInit provides the intiial value for Ret.
// In abstract algebra, this would be the identity element of the monoid
func RetInit() Ret {
	return Ret{avg: 0, zero_trials: 0}
}

// Payoff joins two Rets into a single Ret by addition.
// In abstract algebra, this would be the binary operation of the monoid
func Payoff(a Ret, b Ret) Ret {
	return Ret{
		avg:         a.avg + b.avg,
		zero_trials: a.zero_trials + b.zero_trials,
	}
}

// Serialize takes our stream of Ret and produces a stream of uint32s
func Serialize(inputChan <-chan Ret, outputChan chan<- uint32) {
	for {
		ret := <-inputChan
		outputChan <- uint32(ret.avg)
		outputChan <- uint32(ret.zero_trials)
	}
}

// Param is the input structure to our mapper.
// It represents all the parameters to our simulation.
type Param struct {
	s0         fixed.Int26_6 // The initial price
	drift      fixed.Int26_6 // Daily drift term
	volatility fixed.Int26_6 // Daily volatility
	k          fixed.Int26_6 // Strike price
	days       uint32        // number of days to simulate
}

// Deserialize takes our stream of uint32s and produces a stream of Params
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

// Rands takes our initial seed, and produces a stream of normally distributed random numbers.
func Rands(seed uint32, out chan<- fixed.Int26_6) {
	r := rand.New(seed)
	r.Normals(out)
}

// Sim takes a stream of rands, and a Param, and runs the specified simulation.
func Sim(rands <-chan fixed.Int26_6, p Param) Ret {
	// An iterative model, calculates some integral over 'period'

	// We'll split our pipeline a bit for more parallelism.
	intermediates := make(chan fixed.Int26_6, 1)

	go func() {
		// explanation of + 1 following
		d := p.drift + fixed.I26(1)
		v := p.volatility

		for i := uint16(p.days); i != 0; i-- {
			W := <-rands
			// The original formulation of this is as follows
			// s0 = s0 + d.Mul(s0) + W.Mul(v.Mul(s0))
			// we can factor out s0 to give the following
			// s0 = s0.Mul(1 + d + W.Mul(v))
			// 1 + d is constant, so refactor out
			// hence the following

			// We'll separate out the m aggregator
			// on s0 in the next for loop.
			intermediates <- d + W.Mul(v)
		}

	}()

	s0 := p.s0
	for i := uint16(p.days); i != 0; i-- {
		interm := <-intermediates
		// Aggregate all the `s0 *= intermediate` operations here
		s0 = s0.Mul(interm)
	}

	k := p.k

	var ret Ret
	if s0 > k {
		// The multiplication with exp(-r/t_*days) to be calculated by the host
		ret = Ret{avg: s0 - k}
	} else {
		ret = Ret{zero_trials: 1}
	}

	return ret
}

// BenchmarkSim is to show an example usage.
func BenchmarkSim(n uint32) {
	rs := make(chan fixed.Int26_6, 1)
	go Rands(42, rs)
	Sim(rs, Param{days: n})
}
