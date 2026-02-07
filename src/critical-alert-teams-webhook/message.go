package main

import (
	"fmt"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

func FormatMessage(detail events.CloudWatchAlarmSNSPayload, functionName string, lastSeen string, timeSinceLastSeen time.Duration) []byte {
	var message []byte

	switch detail.NewStateValue {
	case "ALARM":
		message = []byte(fmt.Sprintf(`
	   {
		  "type": "message",
		  "attachments": [
			{
			  "contentType": "application/vnd.microsoft.card.adaptive",
			  "content": {
				"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
				"type": "AdaptiveCard",
				"version": "1.4",
				"body": [
				  {
					"type": "TextBlock",
					"text": "🚨 [ACTIVATED] %s",
					"weight": "Bolder",
					"size": "Large",
					"color": "Attention",
					"wrap": true
				  },
				  {
					"type": "TextBlock",
					"text": "**%s detected no pulse for > 10 minutes**",
					"wrap": true,
					"spacing": "Small"
				  },
				  {
					"type": "FactSet",
					"facts": [
					  {
						"title": "Service",
						"value": "%s"
					  },
					  {
						"title": "Last heartbeat",
						"value": "%s"
					  },
					  {
						"title": "Silence duration",
						"value": "%s"
					  },
					  {
						"title": "Environment",
						"value": "Development"
					  }
					]
				  },
				  {
					"type": "TextBlock",
					"text": "This likely indicates a critical ingestion failure and both customers and business are affected. Immediate investigation required.",
					"wrap": true,
					"color": "Warning",
					"spacing": "Medium"
				  }
				],
				"actions": [
				  {
					"type": "Action.OpenUrl",
					"title": "View logs",
					"url": "https://console.aws.amazon.com/cloudwatch/home"
				  },
				  {
					"type": "Action.OpenUrl",
					"title": "Acknowledge",
					"url": "https://your-runbook-url"
				  }
				]
			  }
			}
		  ]
		}
	`, detail.AlarmName, detail.AlarmName, functionName, lastSeen, timeSinceLastSeen))
	case "OK":
		message = []byte(fmt.Sprintf(`
	   {
		  "type": "message",
		  "attachments": [
			{
			  "contentType": "application/vnd.microsoft.card.adaptive",
			  "content": {
				"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
				"type": "AdaptiveCard",
				"version": "1.4",
				"body": [
				  {
					"type": "TextBlock",
					"text": "✅ [CLOSED] %s",
					"weight": "Bolder",
					"size": "Large",
					"color": "Good",
					"wrap": true
				  },
				  {
					"type": "TextBlock",
					"text": "**%s has detected a pulse and is now OK**",
					"wrap": true,
					"spacing": "Small"
				  },
				  {
					"type": "FactSet",
					"facts": [
					  {
						"title": "Service",
						"value": "%s"
					  },
					  {
						"title": "Last heartbeat",
						"value": "%s"
					  },
					  {
						"title": "Environment",
						"value": "Development"
					  }
					]
				  },
				  {
					"type": "TextBlock",
					"text": "The alert has been automatically closed by CloudWatch Alarms via the heartbeat watchdog.",
					"wrap": true,
					"color": "Accent",
					"spacing": "Medium"
				  }
				],
				"actions": [
				  {
					"type": "Action.OpenUrl",
					"title": "View logs",
					"url": "https://console.aws.amazon.com/cloudwatch/home"
				  }
				]
			  }
			}
		  ]
		}
	`, detail.AlarmName, detail.AlarmName, functionName, lastSeen))
	}

	return message
}
