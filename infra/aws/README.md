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

## GitHub Actions deployment

The repository includes a GitHub Actions workflow at `.github/workflows/deploy.yml` that:

- runs `go test ./...`
- builds and pushes all four service images to Amazon ECR
- initializes Terraform with an S3 backend
- applies the ECS/Fargate infrastructure and service updates

### Required GitHub repository secrets

- `AWS_ROLE_TO_ASSUME`: IAM role assumed by GitHub Actions through OIDC
- `TF_STATE_BUCKET`: S3 bucket that stores Terraform state
- `TF_LOCK_TABLE`: DynamoDB table used for Terraform state locking
- `STRIPE_SECRET_KEY_SECRET_ARN`: Secrets Manager ARN for the Stripe secret key
- `STRIPE_PRICE_STARTER`
- `STRIPE_PRICE_CLUB`
- `STRIPE_PRICE_PRO`

### Optional GitHub repository variables

- `AWS_REGION`: defaults to `eu-west-1`
- `PROJECT_NAME`: defaults to `team-pulse`
- `DEPLOY_ENVIRONMENT`: defaults to `dev`

### Notes

- The workflow creates the expected ECR repositories if they do not exist yet.
- Terraform state must be stored remotely for CI-driven deployments to remain consistent across runners.
