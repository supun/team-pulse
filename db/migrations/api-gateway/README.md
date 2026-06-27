# API Gateway

`api-gateway` is stateless in the current Team Pulse architecture.

It proxies requests to `activity-service` and `payment-service`, so it does not
own persistent tables or SQL migrations at this stage.
