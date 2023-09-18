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
  suffix = concat(var.suffix, [var.app_name])
}

resource "azurerm_resource_group" "network" {
  name     = module.naming.resource_group.name
  location = var.location
  tags = merge(var.tags, tomap({ "deploy-timestamp" = timestamp() }))

  lifecycle {
    ignore_changes = [
      tags["deploy-timestamp"]
    ]
  }
}
