terraform {
  source = "${get_repo_root()}/terraform/modules/resource_group"
}

locals {
 common_vars = yamldecode(file(find_in_parent_folders("prod-common.yaml")))
}

inputs = {
  location = local.common_vars.location
  tags     = local.common_vars.tags
  suffix   = local.common_vars.suffix
  app_name = local.common_vars.app_name
}

