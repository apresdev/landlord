module "landlord_dynamo" {
  source           = "git@github.com:apresdev/apres-terraform.git//modules/aws/dynamodb?ref=rel/dynamodb/1.0.0"
  environment      = var.environment
  application      = "Landlord"
  component        = "Auth"
  name             = "LandlordDynamoDB"
  owner            = var.owner
  stream_enabled   = true
  stream_view_type = "NEW_AND_OLD_IMAGES"
  billing_mode     = "PAY_PER_REQUEST"

  hash_key   = "pk"
  range_key  = "sk"
  attributes = [
    {
      name = "pk"
      type = "S"
    },
    {
      name = "sk"
      type = "S"
    }
  ,
    {
      name = "altName"
      type = "S"
    }
  ]

  global_secondary_indices = [
    {
      name = "altNameIndex"
      hash_key = "altName"
    }
  ]
}

# Create a publisher that will take DynamoDB CDC events from the DynamoDB stream and
# publish them to the SNS topic
module "landlord_cdc_publisher" {
  count = var.landlord_cdc_topic_arn != "none" ? 1 : 0
  #checkov:skip=CKV_TF_1:False positive, we are not using a hash because we use the tagged version.
  source = "git@github.com:apresdev/apres-terraform.git//modules/aws/dynamodb_sns_publisher?ref=rel/dynamodb_sns_publisher/0.2.0"

  name       = "landlord-cdc"
  topic_arn  = var.landlord_cdc_topic_arn
  stream_arn = module.landlord_dynamo.stream_arn

  environment  = var.environment
  owner        = var.owner
  application  = "Landlord"
  component    = "Auth"
  default_tags = var.default_tags
}

module "landlord_ecs" {
  source               = "git@github.com:apresdev/apres-terraform.git//modules/aws/ecs?ref=rel/ecs/2.0.1"
  application          = "Landlord"
  component            = "Auth"
  container_image_uri  = var.container_image_uri
  create_load_balancer = true
  deployment_target    = "FARGATE"
  environment          = var.environment
  name                 = "LandlordECS"
  vpc_environment_tag  = var.environment
  container_port       = 8080
  container_environment_variables = {
    LANDLORD_DB_TABLE_NAME     = module.landlord_dynamo.table_name
    LANDLORD_APP_URL           = var.app_url
    LANDLORD_APP_NAME          = var.app_name
    LANDLORD_APP_ADMIN_EMAIL   = var.app_admin_email
    LANDLORD_LISTEN_ADDR       = "0.0.0.0:8080"
    LANDLORD_DEFAULT_USER_POOL = "landlord-default-user-pool"
  }
  container_cpu_architecture   = "X86_64"
  ecs_task_iam_policy_document = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ses:SendEmail",
        "cognito-idp:AdminAddUserToGroup",
        "cognito-idp:AdminConfirmSignUp",
        "cognito-idp:AdminCreateUser",
        "cognito-idp:AdminDeleteUser",
        "cognito-idp:AdminGetUser",
        "cognito-idp:AdminListUserAuthEvents",
        "cognito-idp:AdminRemoveUserFromGroup",
        "cognito-idp:AdminResetUserPassword",
        "cognito-idp:AdminSetUserMFAPreference",
        "cognito-idp:AdminSetUserPassword",
        "cognito-idp:AdminUpdateUserAttributes",
        "cognito-idp:AssociateSoftwareToken",
        "cognito-idp:ConfirmForgotPassword",
        "cognito-idp:ConfirmSignUp",
        "cognito-idp:CreateGroup",
        "cognito-idp:CreateUserPool",
        "cognito-idp:CreateUserPoolClient",
        "cognito-idp:CreateUserPoolDomain",
        "cognito-idp:DeleteGroup",
        "cognito-idp:DeleteUserPool",
        "cognito-idp:DescribeUserPool",
        "cognito-idp:DescribeUserPoolClient",
        "cognito-idp:DescribeUserPoolDomain",
        "cognito-idp:ForgotPassword",
        "cognito-idp:GetGroup",
        "cognito-idp:InitiateAuth",
        "cognito-idp:ListUserPoolClients",
        "cognito-idp:ListUserPools",
        "cognito-idp:ListUsers",
        "cognito-idp:RespondToAuthChallenge",
        "cognito-idp:SignUp",
        "cognito-idp:VerifySoftwareToken"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "dynamodb:DeleteItem",
        "dynamodb:DescribeTable",
        "dynamodb:GetItem",
        "dynamodb:PutItem",
        "dynamodb:Query",
        "dynamodb:Scan"
      ],
      "Resource": "${module.landlord_dynamo.table_arn}"
    },
    {
      "Effect": "Allow",
      "Action": [
        "dynamodb:Query"
      ],
      "Resource": "${module.landlord_dynamo.table_arn}/index*"
    }
  ]
}
EOF
}

module "landlord_api_gateway" {
  source                 = "git@github.com:apresdev/apres-terraform.git//modules/aws/api_gateway_rest?ref=rel/api_gateway_rest/1.1.0"
  api_version            = "v1"
  openapi_spec_file_path = "../../api/landlord-openapi-ext.json"
  application            = "Landlord"
  component              = "Auth"
  environment            = var.environment
  name                   = "LandlordAPIGateway"
  openapi_spec_variables = {
    nlb_uri              = "http://${module.landlord_ecs.load_balancer_dns_name}"
    cognito_provider_arn = aws_cognito_user_pool.default.arn
  }
  load_balancer_arn      = module.landlord_ecs.load_balancer_arn
  attach_vpc_load_balancer = true
  depends_on = [
    module.landlord_ecs
  ]
}

module "landlord_waf" {
  source                 = "git@github.com:apresdev/apres-terraform.git//modules/aws/waf?ref=rel/waf/1.1.0"
  associate_resource_arn = module.landlord_api_gateway.apigw_stage_arn
  application            = "Landlord"
  component              = "Auth"
  environment            = var.environment
  name                   = "LandlordWAF"
  scope                  = "REGIONAL"
  depends_on = [
    module.landlord_api_gateway
  ]
}
