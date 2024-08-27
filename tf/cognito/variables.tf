variable "app_domain_name" {
  description = "The FQDN of the app."
  type        = string
}

variable "tags" {
  description = "Default set of tags to be applied to all resources"
  type        = map(string)
  default = {
    "application" = "default-auth"
    "owner"       = "Engineering"
    "managed-by"  = "terraform"
  }
}

variable "user_pool_external_id" {
  description = "The external ID of the user pool."
  type        = string
  default     = "landlord_external_id"
}

variable "aws_region" {
    description = "The AWS region."
    type        = string
    default     = "us-east-2"
}

variable "environment" {
  description = "Environment Name, used for naming and tagging AWS resources."
  type        = string
  validation {
    condition     = can(regex("^[A-Z][a-zA-Z0-9]*$", var.environment))
    error_message = "Environment name must be alphanumeric and capitalized."
  }
}
