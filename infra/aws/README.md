# AWS Deployment

This project is scaffolded for AWS ECS Fargate deployment with:

- one public Application Load Balancer
- one public `web-app` ECS service
- one public-facing `api-gateway` ECS service behind `/api/*`
- private `activity-service` and `payment-service` ECS services
- Cloud Map service discovery for internal service-to-service calls
- CloudWatch log groups for each service

## Expected request flow

- `/*` -> `web-app`
- `/api/*` -> `api-gateway`
- `api-gateway` -> `activity-service` and `payment-service` through Cloud Map DNS

## Files

- `infra/aws/terraform`: VPC, ECS, ALB, security groups, Cloud Map, task definitions, and services
- `build/Dockerfile.*`: container build definitions for each deployable service

## Example build and push flow

Create ECR repositories first, then build and push:

```bash
docker build -f build/Dockerfile.web-app -t team-pulse-web-app .
docker build -f build/Dockerfile.api-gateway -t team-pulse-api-gateway .
docker build -f build/Dockerfile.activity-service -t team-pulse-activity-service .
docker build -f build/Dockerfile.payment-service -t team-pulse-payment-service .
```

Tag them for your ECR registry and push them, then copy `terraform.tfvars.example` to `terraform.tfvars` and fill in the image URIs.

## Stripe secret handling

The payment service expects `STRIPE_SECRET_KEY` from AWS Secrets Manager, referenced through `stripe_secret_key_secret_arn`.

Price IDs are passed as normal task environment variables because they are identifiers, not secrets.

## Apply

```bash
cd infra/aws/terraform
terraform init
terraform plan
terraform apply
```

The main output is the public ALB DNS name.
