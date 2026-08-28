# Real AWS provider — standard credentials from the environment / shared config
# (AWS_PROFILE, env vars, or an assumed role). No endpoint overrides.
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
  region = var.region
}

# CloudFront only accepts ACM certificates from us-east-1, so the CDN cert is
# created through this aliased provider regardless of var.region.
provider "aws" {
  alias  = "us_east_1"
  region = "us-east-1"
}
