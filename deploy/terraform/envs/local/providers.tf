# MiniStack provider: fake credentials, path-style S3, and every service endpoint
# pointed at the local emulator (http://localhost:4566). The skip_* flags stop the
# provider from calling the real STS/metadata endpoints during init.
terraform {
  required_version = ">= 1.5"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
  }
}

provider "aws" {
  region     = var.region
  access_key = "test"
  secret_key = "test"

  s3_use_path_style           = true
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true

  endpoints {
    s3         = var.ministack_endpoint
    ec2        = var.ministack_endpoint
    ecs        = var.ministack_endpoint
    elbv2      = var.ministack_endpoint
    elb        = var.ministack_endpoint
    route53    = var.ministack_endpoint
    iam        = var.ministack_endpoint
    sts        = var.ministack_endpoint
    cloudwatch = var.ministack_endpoint
    logs       = var.ministack_endpoint
  }
}
