output "bucket_name" {
  description = "S3 bucket hosting the SPA."
  value       = aws_s3_bucket.web.id
}

output "bucket_regional_domain_name" {
  description = "Regional domain name of the SPA bucket (CloudFront origin)."
  value       = aws_s3_bucket.web.bucket_regional_domain_name
}

output "bucket_arn" {
  description = "ARN of the SPA bucket (for OAC bucket policies)."
  value       = aws_s3_bucket.web.arn
}

output "cluster_name" {
  description = "ECS cluster name."
  value       = aws_ecs_cluster.this.name
}

output "task_definition_arn" {
  description = "ECS task definition ARN (used by the local run-task shim)."
  value       = aws_ecs_task_definition.api.arn
}

output "task_definition_family" {
  description = "ECS task definition family."
  value       = aws_ecs_task_definition.api.family
}

output "alb_dns_name" {
  description = "ALB DNS name (the host MiniStack's ALB data plane answers to)."
  value       = aws_lb.api.dns_name
}

output "alb_arn" {
  description = "ALB ARN."
  value       = aws_lb.api.arn
}

output "target_group_arn" {
  description = "API target group ARN (used by the local register-targets shim)."
  value       = aws_lb_target_group.api.arn
}

output "zone_id" {
  description = "Route53 hosted zone ID for the domain (created or supplied)."
  value       = local.zone_id
}
