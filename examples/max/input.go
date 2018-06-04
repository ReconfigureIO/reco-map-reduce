package main

// copy one channel to another - More functionality is required here for more complex examples
func Deserialize(inputChan <-chan uint32, outputChan chan<- uint32) {
	for {
		outputChan <- <-inputChan
	}
}

// copy one channel to another - More functionality is required here for more complex examples
func Serialize(inputChan <-chan uint32, outputChan chan<- uint32) {
	for {
		outputChan <- <-inputChan
	}
}

// return 0 to be used as an empty state for mappers and reducers
func Uint32Init() uint32 {
	return 0
}

// functionality for mappers - return each individual integer from the sample data
func Identity(a uint32) uint32 {
	return a
}

// functionality for reducers - return the higher value from two inputs
func Max(a uint32, b uint32) uint32 {
	if a > b {
		return a
	} else {
		return b
	}
}

// So that test will still be able to run
func main() {}
