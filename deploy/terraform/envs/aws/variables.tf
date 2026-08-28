variable "region" {
  description = "AWS region to deploy into."
  type        = string
  default     = "eu-west-1"
}

variable "image" {
  description = "Fully-qualified API image (e.g. <acct>.dkr.ecr.<region>.amazonaws.com/onto-api:latest)."
  type        = string
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

variable "domain" {
  description = "Apex domain for the SPA."
  type        = string
  default     = "onto.world"
}

variable "api_domain" {
  description = "Sub-domain for the API."
  type        = string
  default     = "api.onto.world"
}

variable "api_base_url" {
  description = "Absolute API base baked into config.js."
  type        = string
  default     = "https://api.onto.world"
}

variable "desired_count" {
  description = "Number of API tasks the ECS service keeps running."
  type        = number
  default     = 1
}
