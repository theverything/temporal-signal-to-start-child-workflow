package process

import (
	"context"
	"log/slog"
	"time"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const (
	TaskQueue        = "TaskQueue"
	MainWorkflowName = "MainWorkflow"
)

type MainWorkflowParams struct{}

func MainWorkflow(ctx workflow.Context, params MainWorkflowParams) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("MainWorkflowStarted")

	options := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			MaximumInterval:    time.Second * 100,
			BackoffCoefficient: 2.0,
		},
	}

	ctx = workflow.WithActivityOptions(ctx, options)

	var children int

	processChild := func(value string) {
		children++
		logger.Info("NewChild", "children", children)

		logger.Info("MainWorkflowReceivedSignal")

		cctx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID:        value,
			ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
		})

		workflow.ExecuteChildWorkflow(cctx, ChildWorkflow, value)
	}

	childChan := workflow.GetSignalChannel(ctx, "child")

	for {
		if children >= 100 {
			logger.Info("MainWorkflowLoopBreak", "children", children)

			for {
				var value string
				more := childChan.ReceiveAsync(&value)

				if !more {
					break
				}

				processChild(value)
			}

			break
		}

		var value string
		childChan.Receive(ctx, &value)

		processChild(value)

		logger.Info("MainWorkflowLoopDone", "children", children)
	}

	logger.Info("MainWorkflowContinueAsNew", "children", children)

	return workflow.NewContinueAsNewError(ctx, MainWorkflow, params)
}

func ChildWorkflow(ctx workflow.Context, value string) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("ChildWorkflowStarted")

	options := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			MaximumInterval:    time.Second * 100,
			BackoffCoefficient: 2.0,
		},
	}

	err := workflow.Sleep(ctx, time.Second*5)
	if err != nil {
		logger.Error("ChildWorkflowSleep", "error", err.Error())

		return err
	}

	ctx = workflow.WithActivityOptions(ctx, options)

	var nextValue string
	err = workflow.ExecuteActivity(ctx, Activity, value).Get(ctx, &nextValue)
	if err != nil {
		logger.Error("ChildWorkflowActivity", "error", err.Error())

		return err
	}

	logger.Info("ChildWorkflowDone", "value", nextValue)

	return nil
}

func Activity(ctx context.Context, value string) (string, error) {
	logger := activity.GetLogger(ctx)

	time.Sleep(time.Second * 2)

	logger.Info("Activity", "value", value)

	return value + "activity", nil
}

func StartWorker(client client.Client) (worker.Worker, error) {
	w := worker.New(client, TaskQueue, worker.Options{OnFatalError: onFatalError})

	w.RegisterWorkflowWithOptions(MainWorkflow, workflow.RegisterOptions{Name: MainWorkflowName})
	w.RegisterWorkflow(ChildWorkflow)
	w.RegisterActivity(Activity)

	err := w.Start()
	if err != nil {
		return nil, err
	}

	return w, nil
}

func onFatalError(err error) {
	slog.Error("WorkerError", "error", err.Error())
}
