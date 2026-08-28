# Route53 hosted zone and records. On AWS these are alias records pointing the
# apex at the SPA delivery (CloudFront, passed in) and api.* at the ALB. On
# MiniStack they are plain A records to 127.0.0.1 — purely informational, since
# the nginx edge does the real host routing there.
#
# When var.zone_id is empty (MiniStack) the module creates its own zone. When a
# zone_id is supplied (AWS, where registering the domain auto-created the zone)
# the module writes records into that existing zone instead.

resource "aws_route53_zone" "onto" {
  count = var.zone_id == "" ? 1 : 0
  name  = var.domain
}

locals {
  zone_id = var.zone_id != "" ? var.zone_id : aws_route53_zone.onto[0].zone_id

  # Apex alias defaults to the ALB when no dedicated SPA target is supplied.
  spa_alias_name    = var.spa_alias_name != "" ? var.spa_alias_name : aws_lb.api.dns_name
  spa_alias_zone_id = var.spa_alias_zone_id != "" ? var.spa_alias_zone_id : aws_lb.api.zone_id
}

resource "aws_route53_record" "api_alias" {
  count   = var.use_alias_records ? 1 : 0
  zone_id = local.zone_id
  name    = var.api_domain
  type    = "A"

  alias {
    name                   = aws_lb.api.dns_name
    zone_id                = aws_lb.api.zone_id
    evaluate_target_health = true
  }
}

resource "aws_route53_record" "spa_alias" {
  count   = var.use_alias_records ? 1 : 0
  zone_id = local.zone_id
  name    = var.domain
  type    = "A"

  alias {
    name                   = local.spa_alias_name
    zone_id                = local.spa_alias_zone_id
    evaluate_target_health = false
  }
}

resource "aws_route53_record" "api_plain" {
  count   = var.use_alias_records ? 0 : 1
  zone_id = local.zone_id
  name    = var.api_domain
  type    = "A"
  ttl     = 60
  records = [var.record_target_ip]
}

resource "aws_route53_record" "spa_plain" {
  count   = var.use_alias_records ? 0 : 1
  zone_id = local.zone_id
  name    = var.domain
  type    = "A"
  ttl     = 60
  records = [var.record_target_ip]
}
