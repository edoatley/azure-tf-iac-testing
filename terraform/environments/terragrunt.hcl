generate "provider" {
  path      = "grunt_provider.tf"
  if_exists = "overwrite_terragrunt"
  contents  = <<EOF
provider "azurerm" {
  features {
    resource_group {
      prevent_deletion_if_contains_resources = false
    }
  }
}
provider "azuread" {
  # Configuration options
}

provider "local" {
  # Configuration options
}
EOF
}