package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"testing"

	"github.com/ReconfigureIO/fixed"
	"github.com/ReconfigureIO/fixed/host"
	"github.com/ReconfigureIO/sdaccel/xcl"
)

type Ret struct {
	avg         fixed.Int26_6
	zero_trials fixed.Int26_6
}

type Param struct {
	S0         fixed.Int26_6
	Drift      fixed.Int26_6
	Volatility fixed.Int26_6
	K          fixed.Int26_6
	Days       uint32
}

func main() {
	world := xcl.NewWorld()
	defer world.Release()

	program := world.Import("kernel_test")
	defer program.Release()

	krnl := program.GetKernel("reconfigure_io_sdaccel_builder_stub_0_1")
	defer krnl.Release()

	f := func(B *testing.B) {
		doit(world, krnl, B)
	}

	bm := testing.Benchmark(f)
	fmt.Printf("%s\t%s\t%s\n", "monte-carlo", bm, bm.MemString())
}

func doit(world xcl.World, krnl *xcl.Kernel, B *testing.B) {
	B.SetBytes(20)
	B.ReportAllocs()

	seeds := [128]uint32{}
	// Setup the seeds
	for i := range seeds {
		seeds[i] = rand.Uint32()
	}

	// Parameters
	p := Param{
		S0:         host.I26Float64(114.64),    // Actual price
		Drift:      host.I26Float64(0.0016273), // Drft term (daily)
		Volatility: host.I26Float64(0.088864),  // Volatility (daily)
		K:          host.I26Float64(100),       // Strike price
		Days:       252,                        // Days until option expiration
	}

	input := make([]Param, B.N, B.N)
	for i := 0; i < B.N; i++ {
		input[i] = p
	}

	// On the FGPA, allocated ReadOnly memory for the input to the kernel.
	inputBuff := world.Malloc(xcl.ReadOnly, uint(binary.Size(input)))
	defer inputBuff.Free()

	// On the FGPA, allocated ReadOnly memory for the seeds as the context to the mappers kernel.
	contextBuff := world.Malloc(xcl.ReadOnly, uint(binary.Size(seeds)))
	defer contextBuff.Free()

	// Allocate a buffer on the FPGA to store the return value of our computation
	// The output is a Ret, so we need 2 * 4 bytes to store it
	outputBuff := world.Malloc(xcl.WriteOnly, 8)
	defer outputBuff.Free()

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

	B.ResetTimer()
	krnl.Run(1, 1, 1)
	B.StopTimer()

	// Read a Result tyoe expected from the kernel
	var ret [2]fixed.Int26_6
	err1 := binary.Read(outputBuff.Reader(), binary.LittleEndian, &ret)
	if err1 != nil {
		fmt.Println("binary.Read failed:", err1)
	}
}
