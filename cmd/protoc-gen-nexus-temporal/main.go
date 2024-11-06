package main

import (
	"flag"
	"fmt"
	"runtime"

	"github.com/bergundy/protoc-gen-nexus-temporal/internal/plugin"
	"google.golang.org/protobuf/compiler/protogen"
)

var (
	version = "dev"
	commit  = "latest"
)

func main() {
	flag.Parse()
	showVersion := flag.Bool("version", false, "print the version and exit")
	if *showVersion {
		fmt.Printf("protoc-gen-nexus-temporal: %s\n", version)
		fmt.Printf("go: %s\n", runtime.Version())
		return
	}

	p := plugin.New(version, commit)

	opts := protogen.Options{
		ParamFunc: p.Param,
	}

	opts.Run(p.Run)
}
