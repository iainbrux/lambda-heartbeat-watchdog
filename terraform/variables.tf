variable "webhook_url" {
  type        = string
  description = "The URL to send the notification to"
}

variable "lambda_functions" {
  type        = list(string)
  description = "The list of lambda functions that will be monitored by the lambda-heartbeat-watchdog function"
}

variable "monitor_handlers" {
  type        = list(string)
  description = "The list of lambda functions that will be deployed by Terraform"
  default = ["critical-alert-teams-webhook", "lambda-heartbeat-watchdog"]
}