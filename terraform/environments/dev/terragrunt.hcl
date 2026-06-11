# Tell this child configuration to include the root terragrunt.hcl file
include "root" {
  path = find_in_parent_folders()
}

remote_state {
  backend = "azurerm"
  generate = {
    path      = "grunt_backend.tf"
    if_exists = "overwrite_terragrunt"
  }
  config = {
    resource_group_name  = get_env("ARM_BACKEND_RESOURCE_GROUP", "rg-edo-terraform-state")
    storage_account_name = get_env("ARM_BACKEND_STORAGE_ACC", "edoterraformstate")
    container_name       = get_env("ARM_BACKEND_STORAGE_CONTAINER", "dev")
    key                  = "${path_relative_to_include()}/dev-tf-testing.tfstate"
    tenant_id            = get_env("ARM_TENANT_ID", "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx")
    subscription_id      = get_env("ARM_SUBSCRIPTION_ID", "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx")
  }
}