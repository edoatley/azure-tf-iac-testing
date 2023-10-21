variable "location" {
  type = string
  description = "Location for the resources"
}

variable "tags" {
  type = map
  description = "Tags for the resources"
}

variable "resource_group_name" {
  type = string
  description = "Name of resource group"
}

variable "subnet_id" {
  type = string
  description = "Identifier of subnet"
}

variable "vm_size" {
  type = string
  description = "The size of the VM"
  default = "Standard_B1s"
}

variable "vm_name" {
  type = string
  description = "The name of the VM"
}

variable "source_image_reference" {
  type = map
  description = "The source image reference"
  default = {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "18.04-LTS"
    version   = "latest"
  }
}

variable "admin_name" {
  type = string
  description = "The name of the administrator"
}

variable "admin_password" {
  type = string
  description = "The password of the administrator"
}

variable "public_ip_required" {
  type = bool
  description = "Whether a public IP is required"
  default = true
}
