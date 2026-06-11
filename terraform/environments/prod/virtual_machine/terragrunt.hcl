terraform {
  source = "${get_repo_root()}/terraform/modules/vm"
}

locals {
  common_vars = yamldecode(file(find_in_parent_folders("prod-common.yaml")))
}

dependency "rg" {
  config_path = "../resource_group"
  mock_outputs = {
    resource_group_name = "temp-rg"
  }
}

dependency "vnet" {
  config_path = "../virtual_network"
  mock_outputs = {
    subnet_ids = {
      "subnet1" = "temp-subnet1-id"
      "subnet2" = "temp-subnet2-id"
    }
  }
}

inputs = {
  location            = local.common_vars.location
  tags                = local.common_vars.tags
  suffix              = local.common_vars.suffix
  resource_group_name = dependency.rg.outputs.resource_group_name
  subnet_id           = dependency.vnet.outputs.subnet_ids["subnet1"]
  vm_size             = local.common_vars.vm.size
  vm_name             = local.common_vars.vm.name
  admin_name          = local.common_vars.vm.admin_name
  admin_password      = local.common_vars.vm.admin_password
  public_ip_required  = true
}
