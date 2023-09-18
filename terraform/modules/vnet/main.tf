terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = ">= 3.0"
    }
  }
}

provider "azurerm" {
  features {}
}

module "naming" {
  source = "Azure/naming/azurerm"
  suffix = concat(var.suffix, [var.purpose])
}

resource "azurerm_virtual_network" "vnet" {
  name                = module.naming.virtual_network.name
  address_space       = var.address_space
  location            = var.location
  resource_group_name = var.resource_group_name
  tags                = var.tags

  lifecycle {
    ignore_changes = [
      tags
    ]
  }
}

resource "azurerm_subnet" "subnet" {
  for_each = { for subnet in var.subnets : subnet.name => subnet }

  name                                          = each.key
  resource_group_name                           = var.resource_group_name
  virtual_network_name                          = azurerm_virtual_network.vnet.name
  address_prefixes                              = each.value.address_prefixes
  private_endpoint_network_policies_enabled     = lookup(each.value, "private_endpoint_network_policies_enabled", false)
  private_link_service_network_policies_enabled = lookup(each.value, "private_link_service_network_policies_enabled", false)
}