package main

import (
	"math"
	"math/rand"
	"testing"

	"github.com/ReconfigureIO/fixed"
	"github.com/ReconfigureIO/fixed/host"
)

type ParamF struct {
	s0         float32
	drift      float32
	volatility float32
	k          float32
	days       uint32
}

type RetF struct {
	avg         float32
	zero_trials uint32
}

func StocWalkF(rands *rand.Rand, p ParamF) RetF {
	d := p.drift
	v := p.volatility
	s0 := p.s0

	for i := p.days; i != 0; i-- {
		W := float32(rands.NormFloat64())
		s0 += (d * s0) + W*(v*(s0))
	}

	k := p.k
	var ret RetF
	if s0 > k {
		ret = RetF{avg: s0 - k}
	} else {
		ret = RetF{zero_trials: 1}
	}

	return ret
}

func PayoffF(a RetF, b RetF) RetF {
	return RetF{
		avg:         a.avg + b.avg,
		zero_trials: a.zero_trials + b.zero_trials,
	}

}

func TestMapReduce(t *testing.T) {
	// Initialize the reducer.empty
	ret := RetInit()

	t_ := float64(365)   // Total periods in a year
	r := float64(0.033)  // Risk free rate (yearly)
	N := float64(100000) // Number of Monte Carlo trials

	// Parameters
	p := Param{
		s0:         host.I26Float64(114.64),    // Actual price
		drift:      host.I26Float64(0.0016273), // Drft term (daily)
		volatility: host.I26Float64(0.088864),  // Volatility (daily)
		k:          host.I26Float64(100),       // Strike price
		days:       2,                          // Days until option expiration
	}

	rands := make(chan fixed.Int26_6)
	go Rands(123, rands)

	// Simulation loop
	for i := 0; i < int(N); i++ {
		ret = Payoff(ret, StocWalk(rands, p))
	}
	// Finalize the price calculation
	avg := float64(ret.avg) / float64(1<<6)
	days := float64(p.days)
	ret.avg = host.I26Float64((avg * math.Exp(-r/t_*days)) / N)

	// Calculated using the original python code
	expected := 16.064522771
	convert := func(a fixed.Int26_6) float64 {
		return float64(a) / float64(1<<6)
	}

	err_avg := math.Abs(expected-convert(ret.avg)) / expected

	// Error if >10%
	if err_avg > 0.1 {
		t.Errorf("Expected price %f got %f (%f)", expected, convert(ret.avg), err_avg)
	}
}

// Test the float32 version
func TestMapReduceF(t *testing.T) {
	t_ := 365.0 // Total periods in a year
	r := 0.033  // Risk free rate (yearly)
	N := 100000 // Number of Monte Carlo trials

	// Parameters
	p := ParamF{
		s0:         114.64,    // Actual price
		drift:      0.0016273, // Drft term (daily)
		volatility: 0.088864,  // Volatility (daily)
		k:          100,       // Strike price
		days:       2,         // Days until option expiration
	}

	rands := rand.New(rand.NewSource(42))
	ret := RetF{}

	// Simulation loop
	for i := 0; i < N; i++ {
		ret = PayoffF(ret, StocWalkF(rands, p))
	}

	// Finalize the price calculation
	avg := ret.avg
	days := p.days
	ret.avg = float32(float64(avg) * math.Exp(float64(-r/t_)*float64(days)) / float64(N))

	expected := float32(16.064522771)

	err_avg := math.Abs(float64((expected - ret.avg) / expected))

	// Error if >1%
	if err_avg > 0.01 {
		t.Errorf("Expected price %f got %f (%f)", expected, ret.avg, err_avg)
	}
}

func BenchmarkMonteCarlo(b *testing.B) {
	// run the simulation function b.N times

	// Initialize the reducer.empty
	ret := RetInit()

	rands := make(chan fixed.Int26_6)
	go Rands(123, rands)

	// Parameters
	p := Param{
		s0:         host.I26Float64(114.64),    // Actual price
		drift:      host.I26Float64(0.0016273), // Drft term (daily)
		volatility: host.I26Float64(0.088864),  // Volatility (daily)
		k:          host.I26Float64(100),       // Strike price
		days:       252,                        // Days until option expiration
	}
	b.SetBytes(5 * 4)

	for n := 0; n < b.N; n++ {
		ret = Payoff(ret, StocWalk(rands, p))
	}
}

func BenchmarkMonteCarloFloat(b *testing.B) {

	// Parameters
	p := ParamF{
		s0:         114.64,    // Actual price
		drift:      0.0016273, // Drft term (daily)
		volatility: 0.088864,  // Volatility (daily)
		k:          100,       // Strike price
		days:       252,       // Days until option expiration
	}
	b.SetBytes(5 * 4)

	b.RunParallel(func(pb *testing.PB) {
		// Initialize the reducer.empty
		ret := RetF{}
		for pb.Next() {
			rands := rand.New(rand.NewSource(42))
			ret = PayoffF(ret, StocWalkF(rands, p))
		}
	})
}
