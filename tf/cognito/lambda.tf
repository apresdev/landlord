module "landlord_pre_token_generation_lambda" {
  source = "git::git@github.com:apresdev/apres-terraform.git//modules/aws/lambda?ref=rel/lambda/0.2.0"
  name = "landlord_pre_token_generation"
  environment = var.environment
  binary_path = "landlord_pre_token_generation_lambda.mjs"
  runtime = "nodejs20.x"
  handler = "landlord_pre_token_generation_lambda.handler"
}

resource "aws_lambda_permission" "with_sns" {
  statement_id = "AllowExecuteFromCognito"
  action = "lambda:InvokeFunction"
  function_name = module.landlord_pre_token_generation_lambda.lambda_function_arn
  principal = "cognito-idp.amazonaws.com"
  source_arn = aws_cognito_user_pool.default.arn
}
