package process

import (
	"log/slog"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const (
	TaskQueue        = "TaskQueue"
	MainWorkflowName = "MainWorkflow"
	ChildChannelName = "child"
)

type MainWorkflowParams struct{}

func MainWorkflow(ctx workflow.Context, params MainWorkflowParams) error {
	childChan := workflow.GetSignalChannel(ctx, ChildChannelName)

	var children int

	for {
		if children >= 10 {

			for {
				var value string
				more := childChan.ReceiveAsync(&value)

				if !more {
					break
				}

				processChild(ctx, value)
				children++
			}

			break
		}

		var value string
		childChan.Receive(ctx, &value)

		processChild(ctx, value)
		children++
	}

	return workflow.NewContinueAsNewError(ctx, MainWorkflow, params)
}

func ChildWorkflow(ctx workflow.Context) error {
	return nil
}

func processChild(ctx workflow.Context, id string) {
	cctx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID:        id,
		ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
	})

	workflow.ExecuteChildWorkflow(cctx, ChildWorkflow)
}

func StartWorker(client client.Client) (worker.Worker, error) {
	w := worker.New(client, TaskQueue, worker.Options{OnFatalError: onFatalError})

	w.RegisterWorkflowWithOptions(MainWorkflow, workflow.RegisterOptions{Name: MainWorkflowName})
	w.RegisterWorkflow(ChildWorkflow)

	err := w.Start()
	if err != nil {
		return nil, err
	}

	return w, nil
}

func onFatalError(err error) {
	slog.Error("WorkerError", "error", err.Error())
}
