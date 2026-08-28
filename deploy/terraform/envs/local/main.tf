# Local (MiniStack) environment. Discovers the emulator's default VPC/subnets and
# stands up the shared Onto module with local settings: plain HTTP, no ECS service
# (the run-task shim in shim.tf handles that), no IAM role, informational DNS.

data "aws_vpc" "default" {
  default = true
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

module "onto" {
  source = "../../modules/onto"

  name_prefix = "onto"
  static_dir  = abspath("${path.root}/${var.static_dir}")

  # config.js points the SPA at the API's own subdomain (served via the edge).
  api_base_url = var.api_base_url

  # MiniStack's default VPC/subnets; no security groups needed (not enforced).
  vpc_id                  = data.aws_vpc.default.id
  subnet_ids              = data.aws_subnets.default.ids
  alb_security_group_ids  = []
  task_security_group_ids = []
  assign_public_ip        = true

  image    = var.image
  api_port = var.api_port

  # MiniStack's ECS service is phantom, so the container is started imperatively
  # by shim.tf rather than a native service; IAM isn't enforced here.
  create_service     = false
  execution_role_arn = ""

  # Plain HTTP:80, matching the local edge proxy exactly.
  enable_https = false

  # Informational A records; the nginx edge does the real host routing locally.
  use_alias_records = false
  record_target_ip  = "127.0.0.1"
}
