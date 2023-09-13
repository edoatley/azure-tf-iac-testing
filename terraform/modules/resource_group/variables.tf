variable "location" {
  type = string
  description = "Location for the resources"
}

variable "tags" {
  type = map
  description = "Tags for the resources"
}

variable "suffix" {
  type = list
  description = "Suffix value for the naming"
}

variable "app_name" {
  type = string
  description = "The Application name"
}