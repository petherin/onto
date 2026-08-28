# Real AWS environment. Same shared module as local, wired for production:
# a native ECS service, HTTPS on the ALB via ACM, and alias Route53 records
# pointing the apex at CloudFront and api.* at the ALB.

module "onto" {
  source = "../../modules/onto"

  name_prefix = "onto"
  domain      = var.domain
  api_domain  = var.api_domain
  static_dir  = abspath("${path.root}/${var.static_dir}")

  api_base_url = var.api_base_url

  vpc_id                  = data.aws_vpc.default.id
  subnet_ids              = data.aws_subnets.default.ids
  alb_security_group_ids  = [aws_security_group.alb.id]
  task_security_group_ids = [aws_security_group.task.id]
  assign_public_ip        = true

  image    = var.image
  api_port = var.api_port

  # Native ECS service registers the task with the ALB automatically.
  create_service     = true
  desired_count      = var.desired_count
  execution_role_arn = aws_iam_role.execution.arn

  # HTTPS on :443 with an 80->443 redirect, using the Terraform-issued and
  # DNS-validated regional certificate.
  enable_https    = true
  certificate_arn = aws_acm_certificate_validation.alb.certificate_arn

  # Write records into the zone auto-created when the domain was registered.
  zone_id = data.aws_route53_zone.onto.zone_id

  # Alias records: apex -> CloudFront, api.* -> ALB. Z2FDTNDATAQYW2 is
  # CloudFront's fixed hosted zone ID for alias targets.
  use_alias_records = true
  spa_alias_name    = aws_cloudfront_distribution.spa.domain_name
  spa_alias_zone_id = "Z2FDTNDATAQYW2"
}
