resource "azurerm_resource_group" "network" {
  name     = "${module.naming.resource_group.name_unique}-net"
  location = var.location
  tags     = module.tags.tags
  lifecycle {
    ignore_changes = [
      tags
    ]
  }
}