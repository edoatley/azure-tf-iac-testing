variable "location" {
  type = string
  description = "Location for the resources"
}

variable "resource_group_name" {
  type = string
  description = "Resource Group for the resources"
}

variable "tags" {
  type = map
  description = "Tags for the resources"
}

variable "suffix" {
  type = list
  description = "Suffix value for the naming"
}

variable "purpose" {
  type = string
  description = "The Purpose of the virtual Network"
}

variable "address_space" {
  type = list
  description = "Address space for the virtual network"
}

variable "subnets" {
  type = list(object({
    name                                          = string
    address_prefixes                              = list(string)
    private_endpoint_network_policies_enabled     = optional(bool)
    private_link_service_network_policies_enabled = optional(bool)
  }))
  description = "Subnets for the virtual network"
}
