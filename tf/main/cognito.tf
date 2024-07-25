resource "aws_cognito_user_pool" "default" {
  name = "default-user-pool"

  email_verification_subject = "Your Verification Code"
  email_verification_message = "Please use the following code: {####}"

  admin_create_user_config {
    invite_message_template {
      email_subject = format("Invitation to join %s", var.app_domain_name)
      email_message = format(
        "Your username is {username} and temporary password is {####}\n\nClick here to sign in: https://%s",
        var.app_domain_name)
      sms_message = "Your username is {username} and temporary password is {####}"
    }
  }

  username_attributes      = ["email"]
  auto_verified_attributes = ["email"]

  # TODO: this is not secure, just for the example!
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
      min_length = 7
      max_length = 256
    }
  }

  schema {
    attribute_data_type      = "String"
    developer_only_attribute = false
    mutable                  = true
    name                     = "name"
    required                 = true

    string_attribute_constraints {
      min_length = 3
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
    name                     = "impersonate"
    required                 = false

    string_attribute_constraints {
      max_length = 64
    }
  }

  tags = var.tags
}

resource "aws_cognito_user_pool_client" "default" {
  name                 = "default-user-pool-client"
  explicit_auth_flows  = ["ALLOW_USER_PASSWORD_AUTH", "ALLOW_REFRESH_TOKEN_AUTH", "ALLOW_USER_SRP_AUTH"]
  user_pool_id         = aws_cognito_user_pool.default.id
  default_redirect_uri = format("https://%s", var.app_domain_name)
  callback_urls        = [format("https://%s", var.app_domain_name)]
  logout_urls          = [format("https://%s/logout", var.app_domain_name)]
  generate_secret = true
  allowed_oauth_flows = ["code"]
  allowed_oauth_scopes = ["email", "openid"]
}

resource "aws_cognito_identity_pool" "default" {
  identity_pool_name               = "default-identity-pool"
  allow_unauthenticated_identities = false
  cognito_identity_providers {
    client_id               = aws_cognito_user_pool_client.default.id
    provider_name           = aws_cognito_user_pool.default.endpoint
    server_side_token_check = false
  }
  tags = var.tags
}
