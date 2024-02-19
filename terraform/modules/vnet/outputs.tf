output "vnet_name" {
  description = "Specifies the name of the virtual network"
  value       = azurerm_virtual_network.vnet.name
}

output "vnet_id" {
  description = "Specifies the resource id of the virtual network"
  value       = azurerm_virtual_network.vnet.id
}

output "vnet_address_space" {
  description = "Contains the address_space of the vnet"
  value       = azurerm_virtual_network.vnet.address_space
}

output "subnet_ids" {
  description = "Contains a map of the the IDs of the subnet and their names"
  value       = { for subnet in azurerm_subnet.subnet : subnet.name => subnet.id }
}

output "subnet_address_spaces" {
  description = "Contains a map of the the IP spaces of the subnets and their names"
  value       = { for subnet in azurerm_subnet.subnet : subnet.name => subnet.address_prefixes }
}