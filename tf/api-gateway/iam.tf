# This role and policy should be moved to a global resource, along with the section in apigateway.tf
resource "aws_iam_role" "cloudwatch" {
  name               = "api_gateway_cloudwatch_global_role"
  assume_role_policy = data.aws_iam_policy_document.cloudwatch_assume_role.json
}

data "aws_iam_policy_document" "cloudwatch_assume_role" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["apigateway.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}
data "aws_iam_policy_document" "cloudwatch" {
  #checkov:skip=CKV_AWS_111: Unrestricted writes needed by necessity for API Gateway
  #checkov:skip=CKV_AWS_356: Need a "*" in the statement by necessity for API Gateway
  statement {
    effect = "Allow"

    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:DescribeLogGroups",
      "logs:DescribeLogStreams",
      "logs:PutLogEvents",
      "logs:GetLogEvents",
      "logs:FilterLogEvents",
    ]

    resources = ["*"]
  }
}

resource "aws_iam_role_policy" "cloudwatch" {
  name   = "api_gateway_cloudwatch_global_policy"
  role   = aws_iam_role.cloudwatch.id
  policy = data.aws_iam_policy_document.cloudwatch.json
}