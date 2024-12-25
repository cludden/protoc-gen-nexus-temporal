# protoc-gen-nexus-temporal

A Protobuf plugin for generating Temporal Nexus code.

**⚠️ EXPERIMENTAL: Generated code structure is subject to change as feedback is collected. ⚠️**

Supported languages:

- Golang
- Java (TBD)

## Installation

### From GitHub releases (recommended)

1. Download an archive from the [latest release](https://github.com/bergundy/protoc-gen-nexus-temporal/releases/latest).
2. Extract and add to your system's path.

### Using go install

```
go install github.com/bergundy/protoc-gen-nexus-temporal/cmd/protoc-gen-nexus-temporal@latest
```

## Usage

### Install protoc-gen-nexus

Follow instruction in the repo's [README](https://github.com/bergundy/protoc-gen-nexus?tab=readme-ov-file#installation).

### Create a proto file

> NOTE: the directory structure here determines the directory structure of the generated files.

`example/v1/service.proto`

```protobuf
syntax="proto3";

package example.v1;

option go_package = "github.com/bergundy/greet-nexus-example/gen/example/v1;example";

message GreetInput {
  string name = 1;
}

message GreetOutput {
  string greeting = 1;
}

service Greeting {
  rpc Greet(GreetInput) returns (GreetOutput) {
  }
}
```

### Create `buf` config files

> NOTE: Alternatively you may use protoc directly.

`buf.yaml`

```yaml
version: v2
modules:
  - path: .
lint:
  use:
    - BASIC
  except:
    - FIELD_NOT_REQUIRED
    - PACKAGE_NO_IMPORT_CYCLE
breaking:
  use:
    - FILE
  except:
    - EXTENSION_NO_DELETE
    - FIELD_SAME_DEFAULT
```

`buf.gen.yaml`

```yaml
version: v2
clean: true
managed:
  enabled: true
plugins:
  - remote: buf.build/protocolbuffers/go
    out: gen
    opt:
      - paths=source_relative
  - local: protoc-gen-nexus
    out: gen
    strategy: all
    opt:
      - paths=source_relative
      - lang=go
  - local: protoc-gen-nexus-temporal
    out: gen
    strategy: all
    opt:
      - paths=source_relative
      - lang=go
```

### Generate code 

```
buf generate
```

### Implement a service handler and register it with a Temporal worker

```go
import (
	"context"
	"log"

	example "github.com/bergundy/greet-nexus-example/gen/example/v1"
	examplenexus "github.com/bergundy/greet-nexus-example/gen/example/v1/examplenexus"
	"github.com/nexus-rpc/sdk-go/nexus"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporalnexus"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func GreetWorkflow(ctx workflow.Context, input *example.GreetInput) (*example.GreetOutput, error) {
	return &example.GreetOutput{Greeting: "Hello " + input.Name}, nil
}

type greetingHandler struct {
	examplenexus.UnimplementedGreetingNexusServiceHandler
}

func (*greetingHandler) Greet(name string) nexus.Operation[*example.GreetInput, *example.GreetOutput] {
	return temporalnexus.NewWorkflowRunOperation(
		// The name of the Greet operation as defined in the proto.
		name,
		// Workflow to expose as the operation.
		// Input must match the operation input using this builder. See `NewWorkflowRunOperationWithOptions` for
		// exposing workflows with alternative signatures.
		GreetWorkflow,
		func(ctx context.Context, input *example.GreetInput, options nexus.StartOperationOptions) (client.StartWorkflowOptions, error) {
			return client.StartWorkflowOptions{
				ID: meaningfulBusinessID(input),
			}, nil
		})
}

func main() {
	c, err := client.Dial(client.Options{HostPort: "localhost:7233"})
	if err != nil {
		log.Fatal(err)
	}
	w := worker.New(c, "example", worker.Options{})
	// All operations will automatically be registered on the service.
	oneWaySvc, err := examplenexus.NewGreetingNexusService(&greetingHandler{})
	if err != nil {
		log.Fatal(err)
	}
	w.RegisterNexusService(oneWaySvc)
	// Workflows need to be registered separately.
	w.RegisterWorkflow(GreetWorkflow)

	if err = w.Run(worker.InterruptCh()); err != nil {
		log.Fatal(err)
	}
}
```

### Invoke an operation from a workflow

#### Synchronous Call

```go
import (
	"github.com/bergundy/greet-nexus-example/gen/example/v1"
	"github.com/bergundy/greet-nexus-example/gen/example/v1/examplenexustemporal"
	"go.temporal.io/sdk/workflow"
)

func CallerWorkflow(ctx workflow.Context) error {
	c := examplenexustemporal.NewGreetingNexusClient("example-endpoint")
	output, err := c.Greet(ctx, &example.GreetInput{Name: "World"}, workflow.NexusOperationOptions{})
	if err != nil {
		return err
	}
	workflow.GetLogger(ctx).Info("Got greeting", "greeting", output.Greeting)
	return nil
}
```

#### Asynchronous Call

```go
func CallerWorkflow(ctx workflow.Context) error {
	c := examplenexustemporal.NewGreetingNexusClient("example-endpoint")
	fut := c.GreetAsync(ctx, &example.GreetInput{Name: "World"}, workflow.NexusOperationOptions{})
	exec := workflow.NexusOperationExecution{}
	// Wait for operation to be started.
	if err := fut.GetNexusOperationExecution().Get(ctx, &exec); err != nil {
		return err
	}
	output, err := fut.GetTyped(ctx)
	if err != nil {
		return err
	}
	workflow.GetLogger(ctx).Info("Got greeting", "greeting", output.Greeting)
	return nil
}
```

## Contributing

### Prerequisites

- Go >=1.23
- [Buf](https://buf.build/docs/installation/)

### Build the plugin

```
go build ./cmd/...
```

### Generate code

```
rm -rf ./gen && PATH=${PWD}:${PATH} buf generate
```

### Run sanity tests

```
go test ./...
```

### Lint code

[Install](https://golangci-lint.run/welcome/install/) the latest version of `golangci-lint` and run:

```
golangci-lint run ./...
```
