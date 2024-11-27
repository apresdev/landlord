# BEGIN_COMMON_VARS
variable "environment" {
  description = "Environment name, used for tagging AWS resources, and in the bucket name."
  type        = string
  validation {
    condition     = can(regex("^[A-Z][a-zA-Z0-9]*$", var.environment))
    error_message = "Environment name must be alphanumeric and capitalized."
  }
}

variable "application" {
  description = "Application name, used for tagging AWS resources."
  type        = string
  validation {
    condition     = can(regex("^[A-Z][a-zA-Z0-9]*$", var.application))
    error_message = "Application name must be alphanumeric and capitalized."
  }
}

variable "component" {
  description = "Component name, used for tagging AWS resources."
  type        = string
  validation {
    condition     = can(regex("^[A-Z][a-zA-Z0-9]*$", var.component))
    error_message = "Component name must be alphanumeric and capitalized."
  }
}

variable "owner" {
  description = "Owner of the resources, used for tagging AWS resources."
  type        = string
  validation {
    condition     = can(regex("^[A-Z][a-zA-Z0-9]*$", var.owner))
    error_message = "Owner must be alphanumeric and capitalized."
  }
}

variable "default_tags" {
  description = "Default set of tags to be applied to all resources"
  type        = map(string)
  default     = {}
}
# END_COMMON_VARS

variable "app_url" {
  description = "The base URL of the app with its protocol scheme"
  type        = string
  validation {
    condition = can(regex("^https?:\\/\\/[a-zA-Z0-9.-]+(:[0-9]{1,5})?$", var.app_url))
    error_message = "The app URL must include the scheme and domain, but no path information. Do not include a trailing slash."
  }
}

variable "app_name" {
  description = "The name of the app. Used in various UI displays when authenticating"
  type        = string
}

variable "app_admin_email" {
  description = "The email address of an app administrator, used when sending alert notifications"
  type        = string
  validation {
    condition = can(regex("^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$", var.app_admin_email))
    error_message = "The app admin email must be a valid email address."
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

# See https://docs.aws.amazon.com/cognito/latest/developerguide/cognito-user-pools-app-ui-customization.html
variable "hosted_ui_css_filename" {
    description = "Custom CSS to be applied to the hosted UI"
    type        = string
    default     = ""
}

variable "hosted_ui_logo_filename" {
  description = <<-EOF
    The uploaded logo image for the UI customization, provided as a base64-encoded String.
    Drift detection is not possible for this argument."
  EOF
  type        = string
  default     = ""
}

variable "container_image_uri" {
  description = <<EOF
    URI of the container, should include the tag!
    An example using ECR:: 012345668901.dkr.ecr.us-east-2.amazonaws.com/service:46c2c244158828aa06c90655f58f1cc55b641234
  EOF
  type        = string
}

variable "custom_domain_prefix" {
  description = "The custom domain prefix for the Cognito Hosted UI"
  type        = string
  validation {
    condition     = can(regex("^[a-z[a-z0-9-]+$", var.custom_domain_prefix))
    error_message = "The custom domain prefix must be lower case alphanumeric, can contain hyphens, and has leading letter."
  }
}

variable "landlord_cdc_topic_arn" {
  description = "The ARN of the SNS topic to publish CDC events to"
  type        = string
}
