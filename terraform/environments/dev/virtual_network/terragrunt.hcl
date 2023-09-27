terraform {
  source = "${get_repo_root()}/terraform/modules/vnet"
}

locals {
  common_vars = merge(yamldecode(file(find_in_parent_folders("dev-common.yaml"))))
}

dependency "rg" {
  config_path = "../resource_group"
  mock_outputs = {
    resource_group_name = "temp-rg"
  }
}

inputs = {
  location            = local.common_vars.location
  resource_group_name = dependency.rg.outputs.resource_group_name
  tags                = local.common_vars.tags
  suffix              = []
  purpose             = local.common_vars.app_name
  address_space       = local.common_vars.address_space
  subnets             = local.common_vars.subnets
}

