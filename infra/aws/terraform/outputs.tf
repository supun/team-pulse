output "alb_dns_name" {
  value       = aws_lb.public.dns_name
  description = "Public entrypoint for the TeamPulse application."
}

output "cloud_map_namespace" {
  value       = aws_service_discovery_private_dns_namespace.main.name
  description = "Private namespace used for ECS service discovery."
}

output "ecs_cluster_name" {
  value       = aws_ecs_cluster.main.name
  description = "ECS cluster name."
}
