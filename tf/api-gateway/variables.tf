variable "name" {
  description = <<EOF
  Name of the ECS service, to be used for tagging and naming resources. It MUST be unique within the AWS account,
  as it is used for some global resources like IAM. The name will be capitalized in tagging and naming resources.
  EOF
  type        = string
}

variable "default_tags" {
  description = "Default tags to be applied to all resources"
  type        = map(string)
  default = {
    "owner"      = "Engineering"
    "managed-by" = "terraform"
  }
}

variable "environment" {
  description = "Environment Name, used for tagging AWS resources. It will be capitalized in tagging and name resources."
  type        = string
}

variable "application" {
  description = "Application Name, used for tagging AWS resources. It will be capitalized in tagging and name resources."
  type        = string
}

variable "cloudwatch_logs_retention_days" {
  description = "Number of days to retain logs in CloudWatch Logs"
  type        = number
  default     = 365
  validation {
    condition     = contains([1, 3, 5, 7, 14, 30, 60, 90, 120, 150, 180, 365, 400, 545, 731, 1096, 1827, 2192, 2557, 2922, 3288, 3653], var.cloudwatch_logs_retention_days)
    error_message = "Cloudwatch Logs retention days must be one of 1, 3, 5, 7, 14, 30, 60, 90, 120, 150, 180, 365, 400, 545, 731, 1096, 1827, 2192, 2557, 2922, 3288, 3653."
  }
}

variable "api_version" {
  description = "The version of the API, for example v1, used for the stage name and will be the first part of the API path, but is not passed along to the container."
  type        = string
  default     = "v1"
}

variable "openapi_spec_file_path" {
  description = <<EOF
  The path to the OpenAPI spec to use for the API Gateway.
  The following variables are required and will be substituted:
  * vpc_link_connection_id: The ID of the VPC Link to use for the API Gateway
  * nlb_uri:
    * If a REST API this will be "http://<dns-name-of-nlb>"
    * If an HTTP API this will be the ARN of the NLB listener to use for the API Gateway

  For example, the x-amazon-apigateway-integration section might look like (without the slashes):
      x-amazon-apigateway-integration:
        connectionId: "\$\{vpc_link_connection_id\}"
        connectionType: "VPC_LINK"
        httpMethod: "any"
        type: "http_proxy"
        uri: "\$\{nlb_uri\}"
        payloadFormatVersion: "1.0"
  EOF
  type        = string
}

variable "lb_dns_name" {
  type = string
}

variable "lb_arn" {
  type = string
}

variable "user_pool_arn" {
  description = "The ARN of the Cognito User Pool"
  type        = string
}

variable "cloudwatch_iam_arn" {
  description = "The ARN of the IAM role for CloudWatch Logs"
  type        = string
}
