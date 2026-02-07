terraform {
  required_version = ">= 1.1.9, < 2.0.0"
  backend "s3" {
    bucket         = ""
    encrypt        = true
    key            = "terraform.tfstate"
    region         = "eu-west-1"
    dynamodb_table = ""

    workspace_key_prefix = "lambda-heartbeat-watchdog"
    role_arn             = ""
  }
}
