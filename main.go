package main

import (
	"github.com/hashicorp/go-plugin"
	"github.com/marcinwyszynski/backendplugin"
)

func main() {
	plugin.Serve(backendplugin.Server(&FileBackend{}))
}
