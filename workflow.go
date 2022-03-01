package demo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const WorkflowID = "workflow-demo"
const UpdateSignalName = "update-signal"

var defaultActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 1 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		MaximumAttempts: 1,
	},
}

type Data struct {
	Name     string
	Status   string
	Sent     bool
	Attempts int
}

// DemoWorkflow workflow definition
func DemoWorkflow(ctx workflow.Context, d Data) (string, error) {
	logger := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, defaultActivityOptions)
	if err := workflow.ExecuteActivity(ctx, ValidateDataActivity, d).Get(ctx,
		&d); err != nil {
		return "failedValidation", err
	}

	timeoutCtx, cancelHandler := workflow.WithCancel(ctx)
	selector := workflow.NewSelector(timeoutCtx)

	shouldFailWorkflow := false
	dataTimeout := workflow.NewTimer(timeoutCtx, time.Minute*1)
	attemptSend, sendSettable := workflow.NewFuture(timeoutCtx)

	sendInProgress := true
	selector.AddFuture(dataTimeout, func(f workflow.Future) {
		logger.Info("Timer fired")
		CancelData(ctx)
		workflow.ExecuteActivity(ctx, SendEmailActivity)
		shouldFailWorkflow = true

		if sendInProgress {
			cancelHandler()
		}
	}).AddFuture(attemptSend, func(f workflow.Future) {
		_ = f.Get(timeoutCtx, nil)
		sendInProgress = false
	})

	workflow.Go(timeoutCtx, func(ctx workflow.Context) {
		res, err := SendData(ctx, &d)

		sendSettable.Set(res, err)
	})

	updCh := workflow.GetSignalChannel(ctx, UpdateSignalName)
	selector.AddReceive(updCh, func(_ workflow.ReceiveChannel, _ bool) {
		cancelHandler()
	})

	selector.Select(timeoutCtx)

	if shouldFailWorkflow {
		return "failed", errors.New("workflow failed")
	}

	var activityName string
	for {
		updCh.Receive(ctx, &activityName)
		logger.Info("Received request for update", "metadata.activity_name", activityName)
		if err := workflow.ExecuteActivity(ctx, activityName, d).Get(ctx, &d); err != nil {
			logger.Error(
				"Failed to update status",
				"error", err,
			)
			return "update failed", err
		}

		if d.Status == "CLOSED" {
			return "complete", nil
		}
	}
}

func ValidateDataActivity(ctx context.Context, d Data) (Data, error) {
	d.Status = "Checked"
	return d, nil
}

func CancelData(ctx workflow.Context) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Canceling some data..")

	if err := workflow.ExecuteActivity(ctx, CancelDataActivity).Get(ctx, nil); err != nil {
		logger.Error("CancelData failed")
		return err
	}

	return nil
}

func CancelDataActivity(ctx context.Context) error {
	activity.GetLogger(ctx).Info("CancelDataActivity cancelation.")
	return nil
}

func SendEmailActivity(ctx context.Context) error {
	activity.GetLogger(ctx).Info("SendEmailActivity sending notification email as the process takes long time.")
	return nil
}

type SendDataResult struct {
	Response string
	Time     *time.Time
	Data     *Data
}

func SendData(ctx workflow.Context, d *Data) (*Data, error) {
	logger := workflow.GetLogger(ctx)

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2,
			MaximumInterval:    time.Hour * 12,
		},
	})

	for {
		var sendDataResult SendDataResult

		if err := workflow.ExecuteActivity(ctx, SendDataActivity, &d).
			Get(ctx, &sendDataResult); err != nil {
			logger.Error("SendDataActivity failed", "error", err.Error())
			return d, err
		}

		if strings.EqualFold(sendDataResult.Response, "Never") {
			logger.Info("ResendData Ended with Never")
			break
		}

		if sendDataResult.Time != nil {
			sleepErr := workflow.Sleep(ctx, time.Until(*sendDataResult.Time))
			if sleepErr != nil {
				logger.Info("Aborted Due to Canceled Context")
				break
			}
		}
		continue
	}

	return d, nil
}

type File struct {
	Response string `json:"response"`
}

func SendDataActivity(ctx context.Context, d *Data) (SendDataResult, error) {
	jsonFile, err := os.Open("res.json")
	if err != nil {
		fmt.Println(err)
		return SendDataResult{
			Response: "Error",
			Data:     d,
		}, errors.New("failed to open file")
	}
	defer jsonFile.Close()
	byteValue, _ := ioutil.ReadAll(jsonFile)

	f := File{}
	json.Unmarshal(byteValue, &f)

	// Send the data...
	if strings.EqualFold(f.Response, "never") {
		return SendDataResult{
			Response: "Never",
			Data:     d,
		}, nil
	}

	if strings.EqualFold(f.Response, "error") {
		return SendDataResult{
			Response: "Error",
			Data:     d,
		}, errors.New("failed to process")
	}

	t := time.Now().Local().Add(time.Minute * 1)
	return SendDataResult{
		Response: "Sleep",
		Data:     d,
		Time:     &t,
	}, nil
}
