module "landlord_webapp_ecs" {
  source                  = "git@github.com:apresdev/apres-terraform.git//modules/aws/ecs?ref=rel/ecs/2.1.2"
  application             = "Landlord"
  component               = "Auth"
  container_image_uri     = var.container_image_uri

  load_balancer_is_public    = true
  create_load_balancer       = true
  load_balancer_ssl_cert_arn = var.load_balancer_ssl_cert_arn
  load_balancer_port         = 443

  deployment_target       = "FARGATE"
  environment             = var.environment
  name                    = "LandlordWebAppECS"
  vpc_environment_tag     = var.environment
  container_port          = 8080
  container_environment_variables = {
    LANDLORD_DB_TABLE_NAME     = module.landlord_dynamo.table_name
    LANDLORD_APP_URL           = var.app_url
    LANDLORD_CONSOLE_URL       = var.console_url
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
