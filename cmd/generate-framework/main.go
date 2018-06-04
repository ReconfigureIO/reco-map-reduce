package main

import (
	"bytes"
	"flag"
	"go/format"
	"io/ioutil"
	"log"
	"text/template"

	yaml "gopkg.in/yaml.v2"
)

var program = `package main
        import (
                // Import the entire framework
                _ "github.com/ReconfigureIO/sdaccel"
                aximemory "github.com/ReconfigureIO/sdaccel/axi/memory"
                axiprotocol "github.com/ReconfigureIO/sdaccel/axi/protocol"
                arbitrate "github.com/ReconfigureIO/sdaccel/axi/arbitrate"
        )

        func Top(
        		inputData uintptr,
        		outputData uintptr,
                {{ if .Context }}
                contextData uintptr,
                {{ end }}
        		length uint32,
                // The second set of arguments will be the ports for interacting with memory
                memReadAddr chan<- axiprotocol.Addr,
                memReadData <-chan axiprotocol.ReadData,

                memWriteAddr chan<- axiprotocol.Addr,
                memWriteData chan<- axiprotocol.WriteData,
                memWriteResp <-chan axiprotocol.WriteResp) {



        {{ if .Context }}
        memReadAddrContext := make(chan axiprotocol.Addr)
        memReadDataContext := make(chan axiprotocol.ReadData)

        memReadAddrData := make(chan axiprotocol.Addr)
        memReadDataData := make(chan axiprotocol.ReadData)

        go arbitrate.ReadArbitrateX2(memReadAddr, memReadData, memReadAddrContext, memReadDataContext, memReadAddrData, memReadDataData)

        contextChan := make(chan uint32, 1)
        go aximemory.ReadBurstUInt32(
                memReadAddrContext, memReadDataContext, true, contextData, {{ .Mapper.Replicate }}, contextChan)

        {{ if .UseIntermediate }}

        // Intermediate chans
        {{ range $index, $spec := .Contexts -}}
        intermediateContext{{ $spec.Index }} := make(chan uint32, 1)
        {{ end }}

        go func(){
        	for i := 0 ; i < {{ .Mapper.Replicate }}; i++ {
        		el := <- contextChan
        		// For better case handling

                {{ range $index, $spec := .Contexts -}}
                   el{{ $spec.Index }} := el
                {{ end }}

        		switch i >> {{ $.ContextShift }} {
                {{ range $index, $spec := .Contexts -}}
                   case {{ $spec.Index }}:
                     intermediateContext{{ $spec.Index }} <- el{{ $spec.Index }}
                {{ end }}
        		}
        	}
        }()
       {{ end }}

        // create individual context channels
        {{ range $index, $spec := .Mappers -}}
        context{{ $spec.Index }} := make(chan {{ $.Context.Output }}, 1)
        {{ end }}

        {{ range $index, $spec := .Mappers -}}
        {{ if $.UseIntermediate }}
       	go {{ $.Context.Function }}(<-intermediateContext{{ $spec.ContextIndex }}, context{{ $spec.Index }})
        {{ else }}
       	go {{ $.Context.Function }}(<-contextChan, context{{ $spec.Index }})
        {{ end }}
        {{ end }}
        {{ end }}

        // Read all of the input data into a channel
        inputChan := make(chan uint32, {{ .Mapper.Replicate }})

        {{ if .Context }}
        go aximemory.ReadBurstUInt32(
                memReadAddrData, memReadDataData, true, inputData, length * ({{ .Mapper.TypeWidth }} / 32), inputChan)
        {{ else }}
        go aximemory.ReadBurstUInt32(
                memReadAddr, memReadData, true, inputData, length * ({{ .Mapper.TypeWidth }} / 32), inputChan)
        {{ end }}


        // Read all of the input data into a channel
        elementChan := make(chan {{ .Mapper.Type }}, 1)
        go {{ .Mapper.Deserialize }}(inputChan, elementChan)

        // Read all of the input data into a channel
        // dataChan := make(chan [{{ .Mapper.Replicate }}]{{ .Mapper.Type }}, 1)

        {{ range $index, $spec := .Mappers }}
        data{{ $spec.Index }} := make(chan {{ $.Mapper.Type }}, 1)
        {{ end }}


        // Dispatch
        go func() {
            for n := length; n != 0;  {
                for i := uint8(0); i < {{ .Mapper.Replicate }}; i++ {
                    var el {{ .Mapper.Type }}
                    if uint32(i) < n {
                        el = <-elementChan
                    }else{
                        el = [1]{{ .Mapper.Type}}{}[0]
                    }

                    {{ range $index, $spec := .Mappers }}
                       el{{ $spec.Index }} := el
                    {{ end }}

                    switch i {
                            {{ range $index, $spec := .Mappers }}

                                    case {{ $spec.Index }}:
                                           data{{ $spec.Index }} <- el{{ $spec.Index }}
                           {{ end }}
                    }
                }

                if n < {{ .Mapper.Replicate }} {
                   n = 0
                }else {
                  n -= {{ .Mapper.Replicate }}
                }
            }
        } ()

                val := make(chan {{ .Reducer.Type }}, 1)

                // Mapper part

                {{ range $index, $spec := .Mappers }}
              	c{{ $spec.Index }} := make(chan {{ $.Reducer.Type }}, 1)
            	go func() {
                    for {
                    {{ if $.Context }}
        	    	c{{ $spec.Index }} <- {{ $.Mapper.Function }}(context{{ $spec.Index }}, <-data{{ $spec.Index }})
                    {{ else }}
        	    	c{{ $spec.Index }} <- {{ $.Mapper.Function }}(<-data{{ $spec.Index }})
                    {{ end }}
                    }
            	}()
                {{ end }}

                // Reducer part
                {{ range $index, $element := .Reducers }}
                {{ range $index, $spec := $element }}
                    c{{ $spec.OutputIndex }} := make(chan {{ $.Reducer.Type }}, 1)
                 	go func() {
			for{
         	        	c{{ $spec.OutputIndex }} <- {{ $.Reducer.Function }}(<-c{{ $spec.InputA }}, <-c{{ $spec.InputB }})
			}
                 	}()
                     {{ end }}
                     {{ end }}

                	go func() {
			for{
                		val <- <-c{{ .LastIndex }}
            		}
			}()

        retChan := make(chan {{ .Reducer.Type }})
        outputDataChan := make(chan uint32)

        go func(){
            var ret {{ .Reducer.Type }}
            toRead := uint32({{ .Mapper.Replicate }})
            ret = {{ .Reducer.Empty }}()
            for n := length; n > 0; n -= toRead {
                if n < toRead {
                   toRead = n
                }
                ret = {{ .Reducer.Function }}(ret, <-val)
            }
            retChan <- ret
        }()

        go {{ .Reducer.Serialize }}(retChan, outputDataChan)

        // Write it back to the pointer the host requests
        aximemory.WriteBurstUInt32(
                memWriteAddr, memWriteData, memWriteResp, true, outputData, {{ .Reducer.TypeWidth }} / 32, outputDataChan)
        }
`

func main() {
	var filename = flag.String("output", "mapreduce.go", "output file name")
	var configPath = flag.String("config", "reco.yml", "config file location")
	flag.Parse()

	configFile, err := ioutil.ReadFile(*configPath)

	if err != nil {
		log.Fatal("Error opening config file", err)
	}

	var funcs = template.FuncMap{}

	var buffer bytes.Buffer

	var d Data

	err = yaml.Unmarshal(configFile, &d)
	if err != nil {
		log.Fatal("Error reading config file", err)
	}

	// Generate main()
	t := template.Must(template.New("main").Funcs(funcs).Parse(program))
	if err := t.Execute(&buffer, d); err != nil {
		log.Fatal("template", err)
	}

	data, err := format.Source(buffer.Bytes())
	if err != nil {
		log.Fatal("format ", err, string(buffer.Bytes()))
	}

	if err := ioutil.WriteFile(*filename, data, 0644); err != nil {
		panic(err)
	}
}
