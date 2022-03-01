package main

import (
	"context"
	"log"

	"go.temporal.io/sdk/client"

	"github.com/mattchewone/temporal-retry-demo/demo"
)

func main() {
	// The client is a heavyweight object that should be created once per process.
	c, err := client.NewClient(client.Options{
		HostPort: client.DefaultHostPort,
	})
	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	workflowOptions := client.StartWorkflowOptions{
		ID:        demo.WorkflowID,
		TaskQueue: "demo",
	}

	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, demo.DemoWorkflow, demo.Data{
		Name:     "error",
		Attempts: 0,
	})
	if err != nil {
		log.Fatalln("Unable to execute workflow", err)
	}
	log.Println("Started workflow", "WorkflowID", we.GetID(), "RunID", we.GetRunID())
}
