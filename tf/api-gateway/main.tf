locals {
  name        = title(var.name)
  component   = title(var.application)
  application = title(var.application)
  environment = title(var.environment)

  tags = merge(
    var.default_tags,
    tomap({
      environment = title(var.environment),
      component   = title(var.name),
      application = title(var.name)
    })
  )

}
