resource "aws_wafv2_web_acl" "default" {
  name        = local.name
  description = "Default WAF rule for ${local.name}"
  scope       = "REGIONAL"

  # TODO: should probably have a rule that only allows the paths we want and deny the rest?
  default_action {
    allow {}
  }

  # TODO: I had this working at some point but then it failed. Need to put in a rate limit rule, if the AWS ones don't have it already.
  #   rule {
  #     name     = "RateLimitRule"
  #     priority = 1
  #     action {
  #       count {}
  #     }
  #     statement {
  #       rate_based_statement {
  #         limit              = 10000
  #         aggregate_key_type = "IP"
  #       }
  #     }
  #     visibility_config {
  #       cloudwatch_metrics_enabled = true
  #       metric_name                = "hello-world-rate-limit"
  #       sampled_requests_enabled   = false
  #     }
  #   }
  rule {
    name     = "AWSManagedRulesCommonRuleSet"
    priority = 0

    override_action {
      count {}
    }

    statement {
      managed_rule_group_statement {
        name        = "AWSManagedRulesCommonRuleSet"
        vendor_name = "AWS"
      }
    }

    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "AWSManagedRulesCommonRuleSet"
      sampled_requests_enabled   = true
    }
  }
  # Recommended by checkov, includes log4shell check
  rule {
    name     = "AWS-AWSManagedRulesKnownBadInputsRuleSet"
    priority = 1
    override_action {
      none {}
    }
    statement {
      managed_rule_group_statement {
        name        = "AWSManagedRulesKnownBadInputsRuleSet"
        vendor_name = "AWS"
      }
    }
    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "AWSManagedRulesKnownBadInputsRuleSet"
      sampled_requests_enabled   = true
    }
  }

  visibility_config {
    cloudwatch_metrics_enabled = false
    metric_name                = "DefaultRuleSet"
    sampled_requests_enabled   = false
  }
  tags = merge(
    local.tags,
    {
      Name = local.name
    },
  )
}

resource "aws_wafv2_web_acl_association" "default" {
  web_acl_arn  = aws_wafv2_web_acl.default.arn
  resource_arn = aws_api_gateway_stage.default.arn
}


resource "aws_wafv2_web_acl_logging_configuration" "default" {
  log_destination_configs = [module.cwl_waf.cwl_arn]
  resource_arn            = aws_wafv2_web_acl.default.arn
}

resource "aws_cloudwatch_log_resource_policy" "waf" {
  policy_document = data.aws_iam_policy_document.waf.json
  policy_name     = "webacl-policy-waf"
}

data "aws_iam_policy_document" "waf" {
  version = "2012-10-17"
  statement {
    effect = "Allow"
    principals {
      identifiers = ["delivery.logs.amazonaws.com"]
      type        = "Service"
    }
    actions   = ["logs:CreateLogStream", "logs:PutLogEvents"]
    resources = ["${module.cwl_waf.cwl_arn}:*"]
    condition {
      test     = "ArnLike"
      values   = ["arn:aws:logs:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:*"]
      variable = "aws:SourceArn"
    }
    condition {
      test     = "StringEquals"
      values   = [tostring(data.aws_caller_identity.current.account_id)]
      variable = "aws:SourceAccount"
    }
  }
}