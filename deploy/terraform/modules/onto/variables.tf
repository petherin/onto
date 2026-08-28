# Inputs for the shared Onto module. Everything that genuinely differs between
# MiniStack (envs/local) and real AWS (envs/aws) is a variable here, so the same
# resource graph serves both targets.

variable "name_prefix" {
  description = "Prefix for resource names (bucket, cluster, ALB, target group)."
  type        = string
  default     = "onto"
}

variable "domain" {
  description = "Apex domain for the SPA (Route53 zone name)."
  type        = string
  default     = "onto.world"
}

variable "api_domain" {
  description = "Sub-domain for the API."
  type        = string
  default     = "api.onto.world"
}

# ── Static site (S3) ───────────────────────────────────────────────────────────

variable "static_dir" {
  description = "Path to the SPA's static assets directory."
  type        = string
}

variable "api_base_url" {
  description = "Absolute API base baked into config.js (e.g. https://api.onto.world). Empty = same-origin."
  type        = string
  default     = ""
}

# ── Networking (discovered by each env and passed in) ──────────────────────────

variable "vpc_id" {
  description = "VPC the ALB, target group and ECS tasks live in."
  type        = string
}

variable "subnet_ids" {
  description = "Subnets for the ALB and ECS tasks."
  type        = list(string)
}

variable "alb_security_group_ids" {
  description = "Security groups attached to the ALB."
  type        = list(string)
}

variable "task_security_group_ids" {
  description = "Security groups attached to the ECS tasks."
  type        = list(string)
}

variable "assign_public_ip" {
  description = "Assign a public IP to Fargate tasks (true for default/public subnets)."
  type        = bool
  default     = true
}

# ── ECS ────────────────────────────────────────────────────────────────────────

variable "image" {
  description = "Container image for the API (onto-api:local for MiniStack, ECR URL for AWS)."
  type        = string
}

variable "api_port" {
  description = "Container/target port the API listens on."
  type        = number
  default     = 8090
}

variable "cpu" {
  description = "Fargate task CPU units."
  type        = string
  default     = "256"
}

variable "memory" {
  description = "Fargate task memory (MiB)."
  type        = string
  default     = "512"
}

variable "execution_role_arn" {
  description = "Fargate task execution role ARN. Empty on MiniStack (IAM not enforced)."
  type        = string
  default     = ""
}

variable "create_service" {
  description = "Create a native aws_ecs_service. True on AWS; false on MiniStack (its service is phantom, so envs/local uses a run-task shim instead)."
  type        = bool
  default     = true
}

variable "desired_count" {
  description = "Desired task count for the ECS service."
  type        = number
  default     = 1
}

# ── ALB listener / TLS ─────────────────────────────────────────────────────────

variable "enable_https" {
  description = "Serve the API over HTTPS:443 (with an 80->443 redirect). False = plain HTTP:80."
  type        = bool
  default     = false
}

variable "certificate_arn" {
  description = "ACM certificate ARN for the HTTPS listener (required when enable_https = true)."
  type        = string
  default     = ""
}

# ── Route53 records ────────────────────────────────────────────────────────────

variable "zone_id" {
  description = "Existing Route53 hosted zone ID to write records into. Empty = create a zone for var.domain (MiniStack). On AWS this is the zone auto-created when the domain was registered."
  type        = string
  default     = ""
}

variable "use_alias_records" {
  description = "Use Route53 alias records to the ALB (AWS). False = plain A records (MiniStack, informational)."
  type        = bool
  default     = false
}

variable "record_target_ip" {
  description = "IP for plain A records when use_alias_records = false."
  type        = string
  default     = "127.0.0.1"
}

variable "spa_alias_name" {
  description = "Alias target DNS name for the apex/SPA record (e.g. a CloudFront domain). Empty = point apex at the ALB like the API."
  type        = string
  default     = ""
}

variable "spa_alias_zone_id" {
  description = "Hosted zone ID of the SPA alias target (e.g. CloudFront's Z2FDTNDATAQYW2)."
  type        = string
  default     = ""
}
