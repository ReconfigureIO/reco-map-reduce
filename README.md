# reco-map-reduce

[![Build Status](https://travis-ci.org/ReconfigureIO/reco-map-reduce.svg?branch=master)](https://travis-ci.org/ReconfigureIO/reco-map-reduce)

An implementation of MapReduce for FPGAs for use with [Reconfigure.io](https://reconfigure.io). You specify your mapper and reducer and use our framework to produce FPGA code that provides parallel processing.

Our MapReduce framework is provided as a way of scaling up simple functions, and as a template for other parallel patterns.

For more background info, see our [blog post]().

## Installing

```
$ go get github.com/ReconfigureIO/reco-map-reduce/cmd/generate-framework
$ go get golang.org/x/tools/cmd/bundle
```

## Usage

* Create an `input.go` file to define your mapper and reducer, as well as the necessary types.
* Create a `reco.yml` file and specify the mapper and reducer's information ([example](example/max/reco.yml))
* Run `generate-framework -output mapreduce.go` to create the `Top` function of your FPGA code.
* Run `bundle -prefix " " -o main.go .` to bundle both your `input.go`, and the generated `Top` function into a single `main.go`
* Use the `reco` tool as normal to simulate, build and deploy your program.

## Requirements

MapReduce is a framework for processing problems with the potential for parallelism across large datasets using a number of nodes. This usually means multiple computers in a network cluster or spread out geographically in a grid, but in the context of Reconfigure.io, our nodes are individual elements of circuitry on the same FPGA. Put simply, you write the functions required to process the data on one node and MapReduce farms this out to multiple nodes.

Reconfigure.io MapReduce projects have this initial structure:

    cmd
    │   └── test
    │       └── main.go
    ├── input.go
    ├── reco.yml

`input.go` contains all the user-definable functions that will be used in the generated FPGA code.

### Mappers

Mappers are used to farm out data to multiple workers. You choose how many instances of the your mapper function to create, and a routing mechanism is automatically set up to fetch and send data elements to available mappers.

### Reducers

Each reducer takes two inputs from a previous mapper or reducer stage, and generates one output. You choose how many reducer phases there should be and a tree of reducers is automatically created.

### reco.yml

`reco.yml` contains all the settings required to generate the MapReduce framework for a project. The basic structure looks like this:

```
  mapper:
    type:
    typeWidth:
    deserialize:
    function:
    replicate:
  reducer:
    type:
    typeWidth:
    serialize:
    function:
    depth:
    empty:
```

* `type` and `typeWidth` just set the type and width of the data we'll be dealing with.
* `deserialize` `serialize`, `function` and `empty` are set to refer to functions that are defined within `input.go`.
* `deserialize` and `serialize` are used to pipe data elements into and out of the fabric of the FPGA.
* Mapper `function` defines what each mapper does with its sample data element.
* Reducer `function` defines how each reducer processes it's two inputs to create a single output.
* `replicate` is the number of mapper instances you want to create.
* `depth` is the number of reducer stages to include (max=log(mappers)).
* `empty` is a function defined to generate a suitable initial value for the project, this will be used to feed empty inputs to reducers.

## Scope

There are a number of constraints around the kind of example for which MapReduce is a good fit:

* Mappers need to work in a single input element and produce a single output element.
* Reducers need to combine two output elements into a single output element, in a way that is associative (e.g. `max(max(a,b), c) == max(a, max(b, c))` )
* The Reducer also needs an initial value - for this familiar with abstract algebra is called a ‘monoid’.


## Contributing

Pull requests & issues are enthusiastically accepted!

By participating in this project you agree to follow our [Code of Conduct](CODE_OF_CONDUCT.md).
