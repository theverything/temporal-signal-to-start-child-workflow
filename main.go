package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"gitlab.oc.zillow.com/jeffrey/temporal-receive/process"
	temporal "go.temporal.io/sdk/client"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)

	exitCode := 0
	defer func() {
		stop()
		os.Exit(exitCode)
	}()

	temporalClient, err := temporal.Dial(temporal.Options{Logger: slog.Default()})
	if err != nil {
		slog.Error("FailedCreatingTemporalClient", "error", err.Error())
		exitCode = 1
		return
	}

	worker, err := process.StartWorker(temporalClient)
	if err != nil {
		slog.Error("FailedStartingWorker", "error", err.Error())
		exitCode = 1
		return
	}

	<-ctx.Done()

	worker.Stop()
}
