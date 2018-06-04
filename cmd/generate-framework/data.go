package main

type Context struct {
	Output   string
	Function string
}

type Mapper struct {
	Type        string
	TypeWidth   int `yaml:"typeWidth"`
	Deserialize string
	Function    string
	Replicate   int
}

type Reducer struct {
	Type      string
	TypeWidth int `yaml:"typeWidth"`
	Serialize string
	Function  string
	Depth     int
	Empty     string
}

type Data struct {
	Context *Context
	Mapper  Mapper
	Reducer Reducer
}

type MapperSpec struct {
	Index        int
	ContextIndex int
}

type ContextSpec struct {
	Index int
}

const MaxContextLog = 4

// How much to shift the context by
func (d Data) ContextShift() int {
	return MaxContextLog
}

// Conditional on whether a template should use intermediate contexts
func (d Data) UseIntermediate() bool {
	return (d.Mapper.Replicate >> MaxContextLog) > 0
}

func (d Data) Contexts() []ContextSpec {
	length := d.Mapper.Replicate >> MaxContextLog
	ret := make([]ContextSpec, length, length)
	for i := range ret {
		ret[i] = ContextSpec{Index: i}
	}
	return ret

}

func (d Data) Mappers() []MapperSpec {
	ret := make([]MapperSpec, d.Mapper.Replicate, d.Mapper.Replicate)
	for i := range ret {
		ret[i] = MapperSpec{Index: i, ContextIndex: i >> MaxContextLog}
	}
	return ret
}

type ReducerSpec struct {
	Last        bool
	OutputIndex int
	InputA      int
	InputB      int
}

func (d Data) Reducers() [][]ReducerSpec {
	ret := [][]ReducerSpec{}

	x := 0
	p := 0
	replicate := d.Mapper.Replicate
	for i := 0; i < d.Reducer.Depth; i++ {
		lastBlock := i == d.Reducer.Depth-1
		inner := []ReducerSpec{}
		for j := 0; j < replicate; j++ {
			inner = append(inner, ReducerSpec{lastBlock && j == replicate-2, p + replicate, j + x, j + x + 1})
			j++
			p++
		}
		// Arg Index
		x = x + replicate
		// Chan index
		p = p + replicate/2
		replicate = replicate / 2
		ret = append(ret, inner)
	}
	return ret
}

func (d Data) LastIndex() int {
	p := 0
	replicate := d.Mapper.Replicate
	for i := 0; i < d.Reducer.Depth; i++ {
		for j := 0; j < replicate; j++ {
			j++
			p++
		}
		// Chan index
		p = p + replicate/2
		replicate = replicate / 2
	}
	return p
}
