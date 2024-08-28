package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"gitlab.oc.zillow.com/jeffrey/temporal-receive/process"
	temporal "go.temporal.io/sdk/client"
)

func main() {
	client, err := temporal.Dial(temporal.Options{Logger: slog.Default()})
	if err != nil {
		slog.Error("FailedCreatingTemporalClient", "error", err.Error())
		return
	}

	defer client.Close()

	wOpts := temporal.StartWorkflowOptions{TaskQueue: process.TaskQueue}

	w, err := client.ExecuteWorkflow(context.Background(), wOpts, process.MainWorkflowName)
	if err != nil {
		slog.Error("FailedStartingWorkflow", "error", err.Error())
		return
	}

	wid := w.GetID()
	rid := w.GetRunID()

	childRun := func(v string) error {
		return client.SignalWorkflow(context.Background(), wid, rid, "child", v)
	}

	wg := sync.WaitGroup{}

	for i := 0; i < 300; i++ {
		i := i

		wg.Add(1)

		go func(v int) {
			defer wg.Done()

			childRun(fmt.Sprintf("child_%d", v))
		}(i)
	}

	wg.Wait()
}
