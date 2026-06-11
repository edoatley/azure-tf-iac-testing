variable "location" {
  type        = string
  description = "Location for the resources"
}

variable "tags" {
  type        = map(any)
  description = "Tags for the resources"
}

variable "suffix" {
  type        = list(any)
  description = "Suffix value for the naming"
}

variable "app_name" {
  type        = string
  description = "The Application name"
}