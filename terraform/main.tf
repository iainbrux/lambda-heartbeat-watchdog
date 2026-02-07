##############
### Locals ###
##############

locals {
  function_names = toset(var.lambda_functions)
  monitors       = toset(var.monitor_handlers)

  lambda_configs = {
    "lambda-heartbeat-watchdog" : {
      description = "A watchdog that will poll the 'lambda-heartbeats' dynamo table, and trigger the associated lambda function alarms if no pulse has been detected >= 10 mins"
      memory_size = 256
    }
    "critical-alert-teams-webhook" : {
      description = "Takes a CloudWatch Alarm event and posts the event details to the Microsoft Teams channel"
    }
  }
}

#############################
### EventBridge Resources ###
#############################

resource "aws_scheduler_schedule" "lambda_heartbeat_watchdog_rule" {
  name       = "lambda-heartbeat-watchdog-rule"
  group_name = "default"
  state      = "DISABLED"

  flexible_time_window {
    mode = "OFF"
  }

  schedule_expression = "rate(1 minute)"

  target {
    arn      = aws_lambda_function.lambda_functions["lambda-heartbeat-watchdog"].arn
    role_arn = aws_iam_role.lambda_heartbeat_watchdog_scheduler_role.arn
  }
}

#################
### SNS Topic ###
#################

resource "aws_sns_topic" "critical_alerting_topic" {
  name = "CriticalAlertsTeamsWebhook"
}

#######################
### Archive creator ###
#######################

data "archive_file" "lambda_bootstrap_files" {
  for_each = local.monitors

  type        = "zip"
  source_dir  = "${path.module}/../artifacts/${each.value}"
  output_path = "${path.module}/../artifacts/${each.value}.zip"
}

########################
### Lambda resources ###
########################

data "aws_dynamodb_table" "lambda_heartbeats" {
  name = "lambda-heartbeats"
}

resource "aws_lambda_function" "lambda_functions" {
  for_each = local.monitors

  function_name = each.value
  description   = lookup(lookup(local.lambda_configs, each.value, {}), "description", "Deployed by Terraform. No description provided.")

  role        = aws_iam_role.critical_alert_teams_webhook.arn
  runtime     = "provided.al2023"
  handler     = "bootstrap"
  memory_size = lookup(lookup(local.lambda_configs, each.value, {}), "memory_size", 128)
  timeout     = 30

  filename         = "${path.module}/../artifacts/${each.value}.zip"
  source_code_hash = data.archive_file.lambda_bootstrap_files[each.value].output_base64sha256
  publish          = true

  environment {
    variables = merge(
      tomap({
        WEBHOOK_URL = var.webhook_url
      }),
      each.value == "lambda-heartbeat-watchdog" ? tomap({
        FUNCTION_NAMES  = jsonencode(local.function_names)
        ALERT_TOPIC_ARN = aws_sns_topic.critical_alerting_topic.arn
      }) : tomap({})
    )
  }
}

resource "aws_lambda_permission" "critical_alert_teams_webhook" {
  statement_id  = "AllowInvokeFromSns"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.lambda_functions["critical-alert-teams-webhook"].function_name
  principal     = "sns.amazonaws.com"
  source_arn    = aws_sns_topic.critical_alerting_topic.arn
}

resource "aws_lambda_permission" "allow_lambda_heartbeat_schedule_to_invoke_lambda" {
  statement_id  = "AllowInvokeFromEventBridge"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.lambda_functions["lambda-heartbeat-watchdog"].function_name
  principal     = "scheduler.amazonaws.com"
  source_arn    = aws_scheduler_schedule.lambda_heartbeat_watchdog_rule.arn
}

resource "aws_cloudwatch_log_group" "lambda_functions_cloudwatch_logs_groups" {
  for_each = local.monitors

  name              = join("", ["/aws/lambda/", aws_lambda_function.lambda_functions[each.value].function_name])
  retention_in_days = 30
}

resource "aws_sns_topic_subscription" "critical_alert_teams_webhook_subscription" {
  topic_arn = aws_sns_topic.critical_alerting_topic.arn
  protocol  = "lambda"
  endpoint  = aws_lambda_function.lambda_functions["critical-alert-teams-webhook"].arn
}