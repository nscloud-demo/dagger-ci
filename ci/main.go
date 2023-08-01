package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"dagger.io/dagger"
)

var registry = flag.String("registry", "", "Docker registry where to push images")
var commit = flag.String("commit", "", "Commit ID")

func main() {
	flag.Parse()

	ctx := context.Background()

	// initialize Dagger client
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// use a golang:1.19 container
	// get version
	// execute
	golang := client.Container().From("golang:1.19").WithExec([]string{"go", "version"})

	version, err := golang.Stdout(ctx)
	if err != nil {
		panic(err)
	}

	// print output
	fmt.Println("Hello from Dagger and " + version)

	// use a node:16-slim container
	// mount the source code directory on the host
	// at /src in the container
	source := client.Container().
		From("node:16-slim").
		WithDirectory("/src", client.Host().Directory("."), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"node_modules/", "ci/"},
		})

		// set the working directory in the container
		// install application dependencies
	runner := source.WithWorkdir("/src").
		WithExec([]string{"npm", "install"})

		// run application tests
	test := runner.WithExec([]string{"npm", "test", "--", "--watchAll=false"})

	// build application
	// write the build output to the host
	buildDir := test.WithExec([]string{"npm", "run", "build"}).
		Directory("./build")

	_, err = buildDir.Export(ctx, "./build")
	if err != nil {
		panic(err)
	}

	e, err := buildDir.Entries(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Printf("build dir contents:\n %s\n", e)

	ref, err := client.Container().
		From("nginx:1.23-alpine").
		WithDirectory("/usr/share/nginx/html", client.Host().Directory("./build")).
		Publish(ctx, fmt.Sprintf("%s/dagger-app:v%s", *registry, *commit))
	if err != nil {
		panic(err)
	}

	fmt.Printf("Published image to: %s\n", ref)
}
