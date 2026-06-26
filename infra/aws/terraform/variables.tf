variable "aws_region" {
  type        = string
  description = "AWS region for the deployment."
  default     = "eu-west-1"
}

variable "project_name" {
  type        = string
  description = "Project prefix used in AWS resources."
  default     = "team-pulse"
}

variable "environment" {
  type        = string
  description = "Environment name."
  default     = "dev"
}

variable "container_image_web_app" {
  type        = string
  description = "ECR image URI for the web app."
}

variable "container_image_api_gateway" {
  type        = string
  description = "ECR image URI for the API gateway."
}

variable "container_image_activity_service" {
  type        = string
  description = "ECR image URI for the activity service."
}

variable "container_image_payment_service" {
  type        = string
  description = "ECR image URI for the payment service."
}

variable "stripe_secret_key_secret_arn" {
  type        = string
  description = "Secrets Manager ARN that stores the Stripe secret key."
}

variable "stripe_price_starter" {
  type        = string
  description = "Stripe price ID for the starter plan."
  sensitive   = true
}

variable "stripe_price_club" {
  type        = string
  description = "Stripe price ID for the club plan."
  sensitive   = true
}

variable "stripe_price_pro" {
  type        = string
  description = "Stripe price ID for the pro plan."
  sensitive   = true
}
