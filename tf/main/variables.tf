variable "tags" {
  description = "Default set of tags to be applied to all resources"
  type        = map(string)
  default = {
    "application" = "default-auth"
    "owner"       = "Engineering"
    "managed-by"  = "terraform"
  }
}