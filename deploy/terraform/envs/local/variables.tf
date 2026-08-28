variable "region" {
  description = "Region MiniStack emulates."
  type        = string
  default     = "eu-west-1"
}

variable "ministack_endpoint" {
  description = "MiniStack endpoint for all AWS services."
  type        = string
  default     = "http://localhost:4566"
}

variable "image" {
  description = "Locally-built API image MiniStack's ECS RunTask starts."
  type        = string
  default     = "onto-api:local"
}

variable "api_port" {
  description = "API container/target port."
  type        = number
  default     = 8090
}

variable "static_dir" {
  description = "Path to the SPA static assets (relative to this env directory)."
  type        = string
  default     = "../../../../internal/interface/web/static"
}

variable "api_base_url" {
  description = "Absolute API base baked into config.js for the split S3/ALB layout."
  type        = string
  default     = "http://api.onto.world"
}

# Local edge/DNS shim inputs — the pieces MiniStack can't provide that real AWS
# does (host-based routing + a resolver).
variable "edge_container" {
  description = "Name of the nginx edge container to reload after wiring api.conf."
  type        = string
  default     = "ministack-edge"
}

variable "docker_network" {
  description = "Docker network MiniStack attaches ECS task containers to."
  type        = string
  default     = "ministack_default"
}

variable "nginx_conf_dir" {
  description = "Path to the edge nginx conf.d directory (relative to this env directory)."
  type        = string
  default     = "../../../ministack/nginx/conf.d"
}
