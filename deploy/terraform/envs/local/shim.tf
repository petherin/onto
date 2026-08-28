# LOCAL-ONLY glue for the two things MiniStack can't model that real AWS gives
# for free:
#   1. A running task behind the ALB. MiniStack's ECS *service* is phantom, so we
#      run-task imperatively, find the container's IP, and register it with the
#      target group (on AWS the aws_ecs_service does this natively).
#   2. Host-based DNS routing. There's no resolver mapping api.onto.world to the
#      ALB, and the ALB data plane only answers to its generated *.elb DNS name,
#      so we rewrite the nginx edge's api.conf with that name and reload it.
#
# Runs on every apply (timestamp trigger) so a rebuilt image is redeployed.
resource "null_resource" "runtime" {
  triggers = {
    always = timestamp()
  }

  depends_on = [module.onto]

  provisioner "local-exec" {
    interpreter = ["bash", "-c"]
    command     = "${path.module}/shim.sh"

    environment = {
      ONTO_ENDPOINT         = var.ministack_endpoint
      AWS_DEFAULT_REGION    = var.region
      AWS_ACCESS_KEY_ID     = "test"
      AWS_SECRET_ACCESS_KEY = "test"

      CLUSTER     = module.onto.cluster_name
      TASK_FAMILY = module.onto.task_definition_family
      SUBNET      = data.aws_subnets.default.ids[0]
      TG_ARN      = module.onto.target_group_arn
      ALB_DNS     = module.onto.alb_dns_name
      API_PORT    = var.api_port

      EDGE_CONTAINER = var.edge_container
      NGINX_CONF_DIR = abspath("${path.root}/${var.nginx_conf_dir}")
    }
  }
}
