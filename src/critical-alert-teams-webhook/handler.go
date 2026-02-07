package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var ddbClient *dynamodb.Client

func init() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("load aws config: %v", err.Error())
	}
	ddbClient = dynamodb.NewFromConfig(cfg)
}

func getFromDdbTableByFunctionName(ctx context.Context, tableName, functionName, property string) (string, error) {
	res, err := ddbClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"functionName": &types.AttributeValueMemberS{Value: functionName},
		},
	})
	if err != nil {
		return "", err
	}

	return res.Item[property].(*types.AttributeValueMemberS).Value, nil
}

func postToTeamsWebhook(ctx context.Context, detail events.CloudWatchAlarmSNSPayload) error {
	url := os.Getenv("WEBHOOK_URL")
	if url == "" {
		err := fmt.Errorf("WEBHOOK_URL environment variable not set")
		return err
	}

	fmt.Println("Posting message to Teams webhook")

	functionName := strings.TrimPrefix(detail.AlarmName, "HeartbeatWatchdog-")
	fmt.Printf("Function name: %s\n", functionName)
	lastSeenVal, err := getFromDdbTableByFunctionName(ctx, "lambda-heartbeats", functionName, "lastSeen")
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	lastSeen, err := time.Parse(time.RFC3339, lastSeenVal)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	timeSinceLastSeen := time.Since(lastSeen)

	message := FormatMessage(detail, functionName, lastSeenVal, timeSinceLastSeen)

	res, err := http.Post(url, "application/json", bytes.NewBuffer(message))
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		notOkErr := fmt.Errorf("failed to send message to teams")
		return notOkErr
	}

	fmt.Println("Message sent to Teams successfully")

	return nil
}

func Handler(ctx context.Context, event events.SNSEvent) {
	var wg sync.WaitGroup
	wg.Add(len(event.Records))
	defer wg.Wait()

	for _, record := range event.Records {
		go func(ctx context.Context, record events.SNSEventRecord) {
			defer wg.Done()

			snsRecord := record.SNS
			fmt.Printf("[%s] Message = %s\n", snsRecord.Timestamp, snsRecord.Message)

			var detail events.CloudWatchAlarmSNSPayload
			err := json.Unmarshal([]byte(snsRecord.Message), &detail)
			if err != nil {
				log.Fatalf("failed to unmarshal json sns message: %v", err)
			}

			webhookErr := postToTeamsWebhook(ctx, detail)
			if webhookErr != nil {
				log.Fatalf("error posting to teams webhook: %v", err)
			}
		}(ctx, record)
	}
}
