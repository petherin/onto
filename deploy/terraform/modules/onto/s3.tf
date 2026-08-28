# SPA bucket + object uploads. The bucket is left private here; envs/aws attaches
# a CloudFront OAC bucket policy, while envs/local reaches objects path-style
# through the nginx edge. config.js is templated with the environment's API base.

locals {
  content_types = {
    html = "text/html"
    js   = "text/javascript"
    css  = "text/css"
    svg  = "image/svg+xml"
    json = "application/json"
  }

  # Every static asset except config.js, which is generated below with the
  # environment-specific API base URL baked in.
  static_files = toset([
    for f in fileset(var.static_dir, "*") : f if f != "config.js"
  ])
}

resource "aws_s3_bucket" "web" {
  bucket        = "${var.name_prefix}-web"
  force_destroy = true
}

resource "aws_s3_object" "assets" {
  for_each = local.static_files

  bucket       = aws_s3_bucket.web.id
  key          = each.value
  source       = "${var.static_dir}/${each.value}"
  etag         = filemd5("${var.static_dir}/${each.value}")
  content_type = lookup(local.content_types, reverse(split(".", each.value))[0], "application/octet-stream")
}

resource "aws_s3_object" "config" {
  bucket       = aws_s3_bucket.web.id
  key          = "config.js"
  content      = "window.ONTO_API_BASE = \"${var.api_base_url}\";\n"
  content_type = "text/javascript"
}
