provider "aws" {
  region = "eu-west-1"

  assume_role {
    role_arn     = "arn:aws:iam::<account_id>:role/<role_name>"
    session_name = "terraform"
  }

  default_tags {
    tags = {
      owner                 = "Platform Team"
      solution              = "Monitoring"
      owner-contact         = "team-mail@company.com"
      deployer              = "Terraform"
      management-repository = "lambda-heartbeat-watchdog"
      environment           = "Production"
    }
  }
}
