variable "app_url" {
  description = "The base URL of the app with its protocol scheme"
  type        = string
  validation {
    condition = can(regex("^https?:\\/\\/[a-zA-Z0-9.-]+(:[0-9]{1,5})?$", var.app_url))
    error_message = "The app URL must include the scheme and domain, but no path information. Do not include a trailing slash."
  }
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
