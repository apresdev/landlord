# ECS Module

Creates a single ECS service running on EC2 or Fargate with logging enabled to CloudWatch Logs, with an NLB in front.
A future version will include an API Gateway providing the public API as an HTTP API.

The network path APIGW -> NLB -> ECS is in clear text. To mitigate this, the path is contained within the VPC (not on the internet) and the Security Groups are setup so nothing else can see that traffic. That is sufficient in most cases.

## Fargate Alert

While much of this module supports Fargate, it's not fully working due to the way volumes are handled. That needs to be fixed. The best possible way is once the AWS provider fully supports attaching EBS volumes.

## Environment Variables
The following environment variables are passed to the container by default:

| Name | Description |
| -----| ----------- |
| AWS_REGION | AWS Region, like us-east-2, where the container is running |
| AWS_ACCOUNT_ID | Account ID where the container is running |
| ENVIRONMENT | Environment passed into the stack, used to tag evertyhing |
| APPLICATION | Application passed into the stack, used to tag everything |
| COMPONENT | Component passed into the stack, used to tag everything |

## Future
* Add Prometheus to the cluster
* Fully support EC2 and Fargate

<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.6.0, <2.0.0 |
| <a name="requirement_aws"></a> [aws](#requirement\_aws) | >= 5.0.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | 5.55.0 |

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_cwl_apigateway"></a> [cwl\_apigateway](#module\_cwl\_apigateway) | git@github.com:apresdev/apres-terraform.git//modules/aws/cloudwatchlogs | rel/cloudwatchlogs/1.0.0 |
| <a name="module_cwl_waf"></a> [cwl\_waf](#module\_cwl\_waf) | git@github.com:apresdev/apres-terraform.git//modules/aws/cloudwatchlogs | rel/cloudwatchlogs/1.0.0 |

## Resources

| Name | Type |
|------|------|
| [aws_api_gateway_account.default](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/api_gateway_account) | resource |
| [aws_api_gateway_deployment.default](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/api_gateway_deployment) | resource |
| [aws_api_gateway_method_settings.default](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/api_gateway_method_settings) | resource |
| [aws_api_gateway_rest_api.rest_api](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/api_gateway_rest_api) | resource |
| [aws_api_gateway_stage.default](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/api_gateway_stage) | resource |
| [aws_api_gateway_vpc_link.default](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/api_gateway_vpc_link) | resource |
| [aws_cloudwatch_log_resource_policy.waf](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/cloudwatch_log_resource_policy) | resource |
| [aws_cognito_user_pool.default](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/cognito_user_pool) | resource |
| [aws_cognito_user_pool_client.default](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/cognito_user_pool_client) | resource |
| [aws_iam_role.cloudwatch](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role) | resource |
| [aws_iam_role_policy.cloudwatch](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy) | resource |
| [aws_wafv2_web_acl.default](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/wafv2_web_acl) | resource |
| [aws_wafv2_web_acl_association.default](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/wafv2_web_acl_association) | resource |
| [aws_wafv2_web_acl_logging_configuration.default](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/wafv2_web_acl_logging_configuration) | resource |
| [aws_caller_identity.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/caller_identity) | data source |
| [aws_iam_policy_document.cloudwatch](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_iam_policy_document.cloudwatch_assume_role](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_iam_policy_document.waf](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_region.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/region) | data source |
| [aws_subnet.private](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/subnet) | data source |
| [aws_subnets.private](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/subnets) | data source |
| [aws_vpc.default](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/vpc) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_api_version"></a> [api\_version](#input\_api\_version) | The version of the API, for example v1, used for the stage name and will be the first part of the API path, but is not passed along to the container. | `string` | `"v1"` | no |
| <a name="input_application"></a> [application](#input\_application) | Application Name, used for tagging AWS resources. It will be capitalized in tagging and name resources. | `string` | n/a | yes |
| <a name="input_cloudwatch_logs_retention_days"></a> [cloudwatch\_logs\_retention\_days](#input\_cloudwatch\_logs\_retention\_days) | Number of days to retain logs in CloudWatch Logs | `number` | `365` | no |
| <a name="input_default_tags"></a> [default\_tags](#input\_default\_tags) | Default tags to be applied to all resources | `map(string)` | <pre>{<br>  "managed-by": "terraform",<br>  "owner": "Engineering"<br>}</pre> | no |
| <a name="input_environment"></a> [environment](#input\_environment) | Environment Name, used for tagging AWS resources. It will be capitalized in tagging and name resources. | `string` | n/a | yes |
| <a name="input_lb_arn"></a> [lb\_arn](#input\_lb\_arn) | n/a | `string` | n/a | yes |
| <a name="input_lb_dns_name"></a> [lb\_dns\_name](#input\_lb\_dns\_name) | n/a | `string` | n/a | yes |
| <a name="input_name"></a> [name](#input\_name) | Name of the ECS service, to be used for tagging and naming resources. It MUST be unique within the AWS account,<br>  as it is used for some global resources like IAM. The name will be capitalized in tagging and naming resources. | `string` | n/a | yes |
| <a name="input_openapi_spec_file_path"></a> [openapi\_spec\_file\_path](#input\_openapi\_spec\_file\_path) | The path to the OpenAPI spec to use for the API Gateway.<br>  The following variables are required and will be substituted:<br>  * vpc\_link\_connection\_id: The ID of the VPC Link to use for the API Gateway<br>  * nlb\_uri:<br>    * If a REST API this will be "http://<dns-name-of-nlb>"<br>    * If an HTTP API this will be the ARN of the NLB listener to use for the API Gateway<br><br>  For example, the x-amazon-apigateway-integration section might look like (without the slashes):<br>      x-amazon-apigateway-integration:<br>        connectionId: "\$\{vpc\_link\_connection\_id\}"<br>        connectionType: "VPC\_LINK"<br>        httpMethod: "any"<br>        type: "http\_proxy"<br>        uri: "\$\{nlb\_uri\}"<br>        payloadFormatVersion: "1.0" | `string` | n/a | yes |

## Outputs

No outputs.
<!-- END_TF_DOCS -->
