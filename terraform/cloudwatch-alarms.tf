#########################
### Cloudwatch Alarms ###
#########################

resource "aws_cloudwatch_metric_alarm" "heartbeat_watchdog_alarms" {
  for_each = local.function_names

  dimensions = {
    FunctionName = each.value
  }

  alarm_name          = "HeartbeatWatchdog-${each.value}"
  metric_name         = "Invocations"
  statistic           = "Sum"
  threshold           = 1
  treat_missing_data  = "breaching"
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = 1
  period              = 60
  namespace           = "AWS/Lambda"
  alarm_description   = "Monitors the heartbeat for the ${each.value} lambda. Alarm is manually controlled by the lambda-heartbeat-watchdog lambda."
  alarm_actions       = [aws_sns_topic.critical_alerting_topic.arn]
  ok_actions          = [aws_sns_topic.critical_alerting_topic.arn]
}