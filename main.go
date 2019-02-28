package main

import (
	"flag"
	"fmt"

	"github.com/stefreak/terraform-state-store/auth/dummy"
	"github.com/stefreak/terraform-state-store/restapi"
	"github.com/stefreak/terraform-state-store/storage/inmemory"
)

func main() {
	var (
		port = flag.Int("listen-port", 8080, "HTTP Server listen port")
	)

	var store = inmemory.NewStateStore()
	var validator = dummy.NewValidator()

	restapi.Run(fmt.Sprintf(":%d", *port), store, validator)
}
