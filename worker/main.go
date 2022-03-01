package main

import (
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/mattchewone/temporal-retry-demo/demo"
)

func main() {
	// The client and worker are heavyweight objects that should be created once per process.
	c, err := client.NewClient(client.Options{
		HostPort: client.DefaultHostPort,
	})
	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	w := worker.New(c, "demo", worker.Options{
		MaxConcurrentActivityExecutionSize: 3,
	})

	w.RegisterWorkflow(demo.DemoWorkflow)
	w.RegisterActivity(demo.SendDataActivity)
	w.RegisterActivity(demo.SendEmailActivity)
	w.RegisterActivity(demo.CancelDataActivity)
	w.RegisterActivity(demo.ValidateDataActivity)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("Unable to start worker", err)
	}
}
