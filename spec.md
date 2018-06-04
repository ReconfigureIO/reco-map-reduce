# reco.yml

## mapper

Instantiate `replicate` number of the `function`. An appropriate routing infrastructure will be automatically instantiated to push to automatically fetch and send data elements to any available `mappers`.

## reducer

Instantiate a tree of `function` of depth `depth`. Each `reducer` will receive the output of a `mapper` or a previous `reducer`.

Final output will be written back to memory.
