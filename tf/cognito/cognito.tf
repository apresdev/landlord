resource "aws_cognito_user_pool" "default" {
  name = "landlord-default-user-pool"

  email_verification_subject = "Your Verification Code"
  email_verification_message = "Please use the following code: {####}"

  admin_create_user_config {
    invite_message_template {
      email_subject = format("Invitation for %s", var.app_domain_name)
      email_message = format(
        "Login here: https://%s with temporary password: {####}",
        var.app_domain_name)
      sms_message = format(
        "Login here: https://%s with temporary password: {####}",
        var.app_domain_name)
    }
  }

  username_attributes      = ["email"]
  auto_verified_attributes = ["email"]

  # We keep the password policy constraints to a minimum, and use the advice
  # from NIST to inform the user of their password strength. Requiring specific
  # character sets doesn't necessarily make a strong password.
  password_policy {
    minimum_length                   = 8
    require_lowercase                = false
    require_numbers                  = false
    require_symbols                  = false
    require_uppercase                = false
    temporary_password_validity_days = 7
  }

  username_configuration {
    case_sensitive = false
  }

  schema {
    attribute_data_type      = "String"
    developer_only_attribute = false
    mutable                  = true
    name                     = "email"
    required                 = true

    string_attribute_constraints {
      min_length = 5
      max_length = 256
    }
  }

  schema {
    attribute_data_type      = "String"
    developer_only_attribute = false
    mutable                  = true
    name                     = "tenant"
    required                 = false

    string_attribute_constraints {
      max_length = 64
    }
  }

  schema {
    attribute_data_type      = "String"
    developer_only_attribute = false
    mutable                  = true
    name                     = "impersonate_group"
    required                 = false

    string_attribute_constraints {
      max_length = 64
    }
  }

  schema {
    attribute_data_type      = "String"
    developer_only_attribute = false
    mutable                  = true
    name                     = "impersonate_user"
    required                 = false

    string_attribute_constraints {
      max_length = 128
    }
  }

  user_pool_add_ons {
    advanced_security_mode = "AUDIT"
  }

  tags = var.tags
}

resource "aws_cognito_user_pool_client" "default" {
  name                 = "landlord-default-user-pool-client"
  explicit_auth_flows  = ["ALLOW_USER_PASSWORD_AUTH", "ALLOW_REFRESH_TOKEN_AUTH", "ALLOW_USER_SRP_AUTH"]
  user_pool_id         = aws_cognito_user_pool.default.id
  default_redirect_uri = format("https://%s", var.app_domain_name)
  callback_urls        = [format("https://%s", var.app_domain_name)]
  logout_urls          = [format("https://%s/logout", var.app_domain_name)]
  generate_secret = true
  allowed_oauth_flows = ["code"]
  allowed_oauth_scopes = ["email", "openid"]
}
