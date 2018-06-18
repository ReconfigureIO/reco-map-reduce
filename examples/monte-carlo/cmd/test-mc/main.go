package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/ReconfigureIO/fixed"
	"github.com/ReconfigureIO/fixed/host"
	"github.com/ReconfigureIO/sdaccel/xcl"
)

type Ret struct {
	Avg         fixed.Int26_6
	Zero_trials fixed.Int26_6
}

type Param struct {
	S0         fixed.Int26_6
	Drift      fixed.Int26_6
	Volatility fixed.Int26_6
	K          fixed.Int26_6
	Days       uint32
}

func main() {
	// Parameters

	beginI := flag.Uint("start", 4, "number of iterations to start at")
	endI := flag.Uint("end", 6, "number of iterations to end at")
	days := flag.Uint("days", 2, "number of days to run the sim")
	flag.Parse()

	p := Param{
		S0:         host.I26Float64(114.64),    // Actual price
		Drift:      host.I26Float64(0.0016273), // Drft term (daily)
		Volatility: host.I26Float64(0.088864),  // Volatility (daily)
		K:          host.I26Float64(100),       // Strike price
		Days:       uint32(*days),              // Days until option expiration
	}

	world := xcl.NewWorld()
	defer world.Release()

	// Import the kernel.
	// Right now these two idenitifers are hard coded as an output from the build process
	krnl := world.Import("kernel_test").GetKernel("reconfigure_io_sdaccel_builder_stub_0_1")
	defer krnl.Release()

	for n := *beginI; n < *endI; n++ {

		seeds := [4]uint32{}
		// Setup the seeds
		for i := range seeds {
			seeds[i] = rand.Uint32()
		}

		t_ := float64(365)  // Total periods in a year
		r := float64(0.033) // Risk free rate (yearly)

		length := 1 << n

		input := make([]Param, length, length)
		for j := 0; j < length; j++ {
			input[j] = p
		}

		// On the FGPA, allocated ReadOnly memory for the input to the kernel.
		inputBuff := world.Malloc(xcl.ReadOnly, uint(binary.Size(input)))

		// On the FGPA, allocated ReadOnly memory for the seeds as the context to the mappers kernel.
		contextBuff := world.Malloc(xcl.ReadOnly, uint(binary.Size(seeds)))

		// Allocate a buffer on the FPGA to store the return value of our computation
		// The output is a Ret, so we need 2 * 4 bytes to store it
		outputBuff := world.Malloc(xcl.WriteOnly, 8)

		// Pass the arguments to the kernel
		// Set the pointer to the first output buffer
		log.Printf("length=%d p=%+v", len(input), p)
		krnl.SetMemoryArg(0, inputBuff)
		krnl.SetMemoryArg(1, outputBuff)
		krnl.SetMemoryArg(2, contextBuff)
		krnl.SetArg(3, uint32(len(input)))

		// write our input to the kernel at the memory we've previously allocated
		binary.Write(inputBuff.Writer(), binary.LittleEndian, &input)
		binary.Write(contextBuff.Writer(), binary.LittleEndian, &seeds)

		// Run the kernel with the supplied arguments
		start := time.Now()
		krnl.Run(1, 1, 1)
		done := time.Now()

		// Read a Result type expected from the kernel
		var ret Ret
		err1 := binary.Read(outputBuff.Reader(), binary.LittleEndian, &ret)
		if err1 != nil {
			fmt.Println("binary.Read failed:", err1)
		}

		// Compute the final `avg` step
		avg := float64(ret.Avg) / float64(1<<6)
		avg_ := avg * math.Exp(-r/t_*float64(p.Days))
		price := avg_ / float64(length)

		// Print the value we got from the FPGA
		log.Printf("price=%f, zero_trials=%d ns=%d", price, uint32(ret.Zero_trials), done.Sub(start).Nanoseconds())

		inputBuff.Free()
		contextBuff.Free()
		outputBuff.Free()
	}
}
