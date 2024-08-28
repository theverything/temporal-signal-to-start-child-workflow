# Temporal Signal to Start Child Workflow POC

## Instructions

Run each of the following commands in a new terminal.

1. Run `temporal server start-dev --namespace default` to start a local temporal server.
2. Run `go run ./` to start the temporal worker.
3. Run `go run ./cmd/client` to send 10 signals to the main workflow.