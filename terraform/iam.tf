#####################
### IAM Resources ###
#####################

resource "aws_iam_role" "critical_alert_teams_webhook" {
  name               = "critical-alert-teams-webhook-role"
  assume_role_policy = data.aws_iam_policy_document.critical_alert_teams_webhook_assume_policy.json
}

resource "aws_iam_role_policy" "critical_alert_teams_webhook_inline_logging" {
  name   = "LoggingPolicy"
  role   = aws_iam_role.critical_alert_teams_webhook.id
  policy = data.aws_iam_policy_document.critical_alert_teams_webhook_logging_policy.json
}

resource "aws_iam_role_policy" "critical_alert_teams_webhook_dynamodb_inline" {
  name   = "DdbPolicy"
  role   = aws_iam_role.critical_alert_teams_webhook.id
  policy = data.aws_iam_policy_document.critical_alert_teams_webhook_dynamodb_policy.json
}

resource "aws_iam_role_policy" "critical_alert_teams_webhook_sns_inline" {
  name   = "SnsPolicy"
  role   = aws_iam_role.critical_alert_teams_webhook.id
  policy = data.aws_iam_policy_document.critical_alert_teams_webhook_dynamodb_policy.json
}

resource "aws_iam_role_policy" "cloudwatch_alarms_policy" {
  name   = "CloudWatchAlarmsPolicy"
  role   = aws_iam_role.critical_alert_teams_webhook.id
  policy = data.aws_iam_policy_document.cloudwatch_alarms_policy.json
}

// Would be good to move the 4 iam policy resources into a for_each loop

data "aws_iam_policy_document" "critical_alert_teams_webhook_assume_policy" {
  statement {
    effect = "Allow"

    actions = ["sts:AssumeRole"]

    principals {
      type = "Service"
      identifiers = [
        "lambda.amazonaws.com",
        "events.amazonaws.com",
      ]
    }
  }
}

data "aws_iam_policy_document" "critical_alert_teams_webhook_logging_policy" {
  statement {
    sid    = "CloudWatchLogsPolicy"
    effect = "Allow"
    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]
    resources = ["*"]
  }
}

data "aws_iam_policy_document" "cloudwatch_alarms_policy" {
  statement {
    sid    = "CloudWatchAlarmsPolicy"
    effect = "Allow"
    actions = [
      "cloudwatch:SetAlarmState"
    ]
    resources = [for alarm in aws_cloudwatch_metric_alarm.heartbeat_watchdog_alarms : alarm.arn]
  }
}

data "aws_iam_policy_document" "critical_alert_teams_webhook_dynamodb_policy" {
  statement {
    sid    = "PubSubPolicy"
    effect = "Allow"
    actions = [
      "dynamodb:PutItem",
      "dynamodb:UpdateItem",
      "dynamodb:GetItem",
      "dynamodb:DescribeTable",
      "dynamodb:Query",
      "sns:Publish",
    ]
    resources = [aws_sns_topic.critical_alerting_topic.arn, data.aws_dynamodb_table.lambda_heartbeats.arn]
  }
}

data "aws_iam_policy_document" "scheduler_invoke_watchdog_lambda" {
  statement {
    effect    = "Allow"
    actions   = ["lambda:InvokeFunction"]
    resources = [aws_lambda_function.lambda_functions["lambda-heartbeat-watchdog"].arn]
  }
}

resource "aws_iam_role" "lambda_heartbeat_watchdog_scheduler_role" {
  name               = "lambda-heartbeat-watchdog-scheduler-role"
  assume_role_policy = data.aws_iam_policy_document.scheduler_assume_role.json
}

data "aws_iam_policy_document" "scheduler_assume_role" {
  statement {
    effect = "Allow"
    principals {
      type        = "Service"
      identifiers = ["scheduler.amazonaws.com"]
    }
    actions = ["sts:AssumeRole"]
  }
}

resource "aws_iam_role_policy" "scheduler_invoke_watchdog_lambda" {
  name   = "scheduler-invoke-watchdog-lambda"
  role   = aws_iam_role.lambda_heartbeat_watchdog_scheduler_role.id
  policy = data.aws_iam_policy_document.scheduler_invoke_watchdog_lambda.json
}
