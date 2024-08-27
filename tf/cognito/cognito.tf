resource "aws_cognito_user_pool" "default" {
  name = "landlord-default-user-pool"

  email_verification_subject = "Your Verification Code"
  email_verification_message = "Please use the following code: {####}"

  admin_create_user_config {
    allow_admin_create_user_only = true
    invite_message_template {
      email_subject = format("Invitation for %s", var.app_domain_name)
      email_message = format(
        "{username} login here: https://%s with temporary password: {####}",
        var.app_domain_name)
      sms_message = format(
        "{username} login here: https://%s with temporary password: {####}",
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

  mfa_configuration = "OPTIONAL"

  software_token_mfa_configuration {
    enabled = true
  }

  sms_configuration {
    external_id    = var.user_pool_external_id
    sns_caller_arn = aws_iam_role.landlord_sms_cognito_role.arn
    sns_region     = var.aws_region
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

  lambda_config {
    pre_token_generation_config {
      lambda_arn     = module.landlord_pre_token_generation_lambda.lambda_function_arn
      lambda_version = "V2_0"
    }
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

resource "aws_iam_role" "landlord_sms_cognito_role" {
  name = "LandlordSMSCognitoRole"

  force_detach_policies = true

  assume_role_policy = jsonencode({
    "Version" : "2012-10-17",
    "Statement" : [
      {
        "Sid" : "",
        "Effect" : "Allow",
        "Principal" : {
          "Service" : "cognito-idp.amazonaws.com"
        },
        "Action" : "sts:AssumeRole",
        "Condition" : {
          "StringEquals" : {
            "sts:ExternalId" : var.user_pool_external_id,
            "aws:SourceAccount": data.aws_caller_identity.current.account_id
          },
          "ArnLike" : {
            "aws:SourceArn": format(
              "arn:aws:cognito-idp:%s:%s:userpool/*",
              var.aws_region,
              data.aws_caller_identity.current.account_id)
          }
        }
      },
    ]
  })

  tags = var.tags
}

resource "aws_iam_role_policy" "landlord_sms_cognito_role_policy" {
  name = "landlord_sms_cognito_role_policy"
  role = aws_iam_role.landlord_sms_cognito_role.id
  policy = jsonencode({
    "Version" : "2012-10-17",
    "Statement" : [
      {
        "Effect" : "Allow",
        "Action" : [
          "sns:publish"
        ],
        "Resource" : [
          "*"
        ]
      }
    ]
  })
}
