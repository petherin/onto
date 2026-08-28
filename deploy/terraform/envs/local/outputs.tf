output "bucket_name" {
  value = module.onto.bucket_name
}

output "alb_dns_name" {
  value = module.onto.alb_dns_name
}

output "cluster_name" {
  value = module.onto.cluster_name
}

output "zone_id" {
  value = module.onto.zone_id
}

output "urls" {
  description = "Where to reach the stack (via the local edge; run 'make ministack-hosts' once)."
  value = {
    spa = "http://onto.world/"
    api = "http://api.onto.world/api/state"
  }
}
