resource "aws_cognito_user_pool" "default" {
  name = "default-user-pool"

  email_verification_subject = "Your Verification Code"
  email_verification_message = "Please use the following code: {####}"
  # alias_attributes           = ["email"]
  username_attributes      = ["email"] # Enforce only email signups
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


  tags = var.tags
}

resource "aws_cognito_user_pool_client" "default" {
  name                = "default-user-pool-client"
  explicit_auth_flows = ["ALLOW_USER_PASSWORD_AUTH", "ALLOW_REFRESH_TOKEN_AUTH", "ALLOW_USER_SRP_AUTH"]

  user_pool_id = aws_cognito_user_pool.default.id
}

resource "aws_cognito_identity_pool" "default" {
  identity_pool_name               = "default-identity-pool"
  allow_unauthenticated_identities = false
  cognito_identity_providers {
    client_id               = aws_cognito_user_pool_client.default.id
    provider_name           = aws_cognito_user_pool.default.endpoint
    server_side_token_check = false
  }
}
