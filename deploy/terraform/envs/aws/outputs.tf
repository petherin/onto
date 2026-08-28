output "bucket_name" {
  value = module.onto.bucket_name
}

output "alb_dns_name" {
  value = module.onto.alb_dns_name
}

output "cloudfront_domain_name" {
  value = aws_cloudfront_distribution.spa.domain_name
}

output "cluster_name" {
  value = module.onto.cluster_name
}

output "zone_id" {
  description = "Route53 hosted zone ID the stack writes records into (the zone auto-created at domain registration)."
  value       = module.onto.zone_id
}

output "urls" {
  value = {
    spa = "https://${var.domain}/"
    api = "https://${var.api_domain}/api/state"
  }
}
