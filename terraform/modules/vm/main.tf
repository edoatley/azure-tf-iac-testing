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

resource "azurerm_public_ip" "vm_public_ip" {
  count = var.public_ip_required ? 1 : 0
  name                = "${var.vm_name}-public-ip"
  location            = var.location
  resource_group_name = var.resource_group_name
  allocation_method   = "Static"
  lifecycle {
    # If this resource is to be associated with a resource that requires disassociation 
    # before destruction (such as azurerm_network_interface) it is recommended to set the 
    # lifecycle argument create_before_destroy = true.  Otherwise, it can fail to disassociate
    # on destruction.
    create_before_destroy = true
  }
}

resource "azurerm_network_interface" "vm_nic" {
  name                = "${var.vm_name}-network-interface"
  location            = var.location
  resource_group_name = var.resource_group_name

  ip_configuration {
    name                          = "${var.vm_name}-ip-configuration"
    subnet_id                     = var.subnet_id
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id = var.public_ip_required ? azurerm_public_ip.vm_public_ip.id : null
  }
}

resource "azurerm_linux_virtual_machine" "vm" {
  name                  = var.vm_name
  location              = var.location
  resource_group_name   = var.resource_group_name 
  size                  = var.vm_size
  admin_username        = var.admin_name
  admin_password        = var.admin_password
  network_interface_ids = [azurerm_network_interface.vm_nic.id]

  source_image_reference {
    publisher = var.source_image_reference.publisher
    offer     = var.source_image_reference.offer
    sku       = var.source_image_reference.sku
    version   = var.source_image_reference.version
  }
  os_disk {
    name                 = "example-os-disk"
    caching              = "ReadWrite"
    storage_account_type = "Standard_LRS"
  }

  computer_name = var.vm_name
  tags          = var.tags
}
