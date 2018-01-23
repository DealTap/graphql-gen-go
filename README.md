# graphql-gen-go

**Early Stage**

Generates boiler-plate code for [golang graphql](https://github.com/neelance/graphql-go) from graphql schema files.

## Install from source

### Prerequisites

1. make (http://www.gnu.org/software/make/)
1. Go and Go workspace
1. glide (https://github.com/Masterminds/glide)

### Download and Build

1. `go get -u github.com/DealTap/graphql-gen-go`
1. Change current directory `cd $GOPATH/src/github.com/dealtap/graphql-gen-go`
1. Run `make install build`

## Getting Started

1. Download the latest binary from [releases](https://github.com/DealTap/graphql-gen-go/releases) or build from the source
1. To generate code, run `graphql-gen-go SOME_FILE --out_dir SOME_PATH --pkg SOME_PACKAGE`. You can pass more than one file
1. Run `graphql-gen-go --help` to get usage details

## Parameters

* `out_dir` - destination directory of the generated files. Default is current directory
* `pkg` - package name of the generated files. Default is `main`

## Notes

* Resolver function is not generated for a GraphQL type which has a property with arguments. 
It is assumed that such a property would require additional logic; so, it should be implemented manually. 
Take a look at `sample/api/api-extra.go` for an example
* A `server.gql.go` file is also generated which implements a custom http handler and runs a GraphQL server. 
You can use this or your own http handler or built in one in [graphql-go](https://github.com/neelance/graphql-go)

## How to Use Generated Code

**With generated http handler**

A sample test server is included: `sample/test-server.go`. 
You can start this sample server by running `make run-sample` and it can be queried like
```
curl -XPOST localhost:7050/graphql -H "Content-Type: application/graphql" \
-d '{ person(id: "1000") { name email friends { name email } } }'
```

**With http handler from graphql-go**

```
var schema *graphql.Schema
type resolver struct{}

func (r *resolver) Person(request api.PersonRequest) api.PersonResolver {
  ...
}

func init() {
  schema, err = graphql.ParseSchema(api.Schema, &resolver{})
  	if err != nil {
  		panic(err)
  	}
}

func main() {
  http.Handle("/graphql", &relay.Handler{Schema: schema})
  err := http.ListenAndServe(":7050", nil)
  if err != nil {
    panic(err)
  }
}
```

## Status

* [x] Minimal Api
* [x] Generate Code for graphql types
  * [x] query
  * [x] mutation
  * [x] interface
  * [x] object
  * [x] enum
  * [x] input object
  * [x] union
* [ ] Custom http handler
  * [x] minimal implementation 
  * [ ] Add facebook data loader in custom http handler for batching
  * [ ] option to specify middleware/interceptors that will run before/after executing incoming graphql request i.e., auth check
* [ ] Improve api/code
  * [ ] refactor duplicate code
  * [ ] handle imports dynamically instead of hard-coding
  * [ ] option to disable generation of custom http handler   
  * [ ] option to disable generation of custom http handler   

## Credit

Hard fork of [graphql-gen-go](https://github.com/euforic/graphql-gen-go) and inspired by [protoc-gen-go](https://github.com/golang/protobuf/protoc-gen-go).
Thanks to [neelance](https://github.com/neelance) for the great [graphql-go](https://github.com/neelance/graphql-go) library. 
