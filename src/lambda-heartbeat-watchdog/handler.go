package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var ddbClient *dynamodb.Client
var cwClient *cloudwatch.Client

func init() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("load aws config: %v", err.Error())
	}
	ddbClient = dynamodb.NewFromConfig(cfg)
	cwClient = cloudwatch.NewFromConfig(cfg)
}

func triggerAlarm(ctx context.Context, cwClient *cloudwatch.Client, alarmName string, functionName string) error {
	message := fmt.Sprintf("%s has not registered a pulse for >= 10 minutes. This likely means a critical failure within the architecture and customers are affected. Immediate attention required.\nManually triggered by: lambda-heartbeat-watchdog", functionName)
	_, err := cwClient.SetAlarmState(ctx, &cloudwatch.SetAlarmStateInput{
		AlarmName:   aws.String(alarmName),
		StateValue:  cwTypes.StateValueAlarm,
		StateReason: aws.String(message),
	})
	return err
}

func settleAlarm(ctx context.Context, cwClient *cloudwatch.Client, alarmName string, functionName string) error {
	message := fmt.Sprintf("%s has registered a pulse within the last minute - moved alarm back in to OK state.\nManually triggered by: lambda-heartbeat-watchdog", functionName)
	_, err := cwClient.SetAlarmState(ctx, &cloudwatch.SetAlarmStateInput{
		AlarmName:   aws.String(alarmName),
		StateValue:  cwTypes.StateValueOk,
		StateReason: aws.String(message),
	})
	return err
}

func getFunctionNames() ([]string, error) {
	raw, ok := os.LookupEnv("FUNCTION_NAMES")
	if !ok || raw == "" {
		return nil, errors.New("FUNCTION_NAMES environment variable not set")
	}

	var functionNames []string
	if err := json.Unmarshal([]byte(raw), &functionNames); err != nil {
		return nil, err
	}

	return functionNames, nil
}

func queryHeartbeatByFunctionName(ctx context.Context, functionName string) error {
	res, err := ddbClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String("lambda-heartbeats"),
		Key: map[string]ddbTypes.AttributeValue{
			"functionName": &ddbTypes.AttributeValueMemberS{Value: functionName},
		},
	})
	if err != nil {
		return err
	}

	lastSeenVal := res.Item["lastSeen"].(*ddbTypes.AttributeValueMemberS).Value
	lastSeen, err := time.Parse(time.RFC3339, lastSeenVal)
	if err != nil {
		return err
	}

	timeSinceLastSeen := time.Since(lastSeen)
	alarmName := fmt.Sprintf("HeartbeatWatchdog-%s", functionName)
	if timeSinceLastSeen >= 10*time.Minute {
		alarmErr := triggerAlarm(ctx, cwClient, alarmName, functionName)
		if alarmErr != nil {
			return err
		}

		return nil
	} else {
		settleErr := settleAlarm(ctx, cwClient, alarmName, functionName)
		if settleErr != nil {
			return err
		}

		return nil
	}
}

func Handler(ctx context.Context) {
	functionNames, err := getFunctionNames()
	if err != nil {
		log.Fatalf("error getting function names: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(len(functionNames))
	defer wg.Wait()

	for _, functionName := range functionNames {
		go func(functionName string) {
			defer wg.Done()

			err := queryHeartbeatByFunctionName(ctx, functionName)
			if err != nil {
				log.Fatalf("error querying lambda %s heartbeat: %v", functionName, err)
			}

			fmt.Printf("lambda %s heartbeat checked successfully\n", functionName)
		}(functionName)
	}
}
