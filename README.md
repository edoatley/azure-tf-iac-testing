# Testing Terraform Infrastructure as Code

- [Testing Terraform Infrastructure as Code](#testing-terraform-infrastructure-as-code)
  - [Introduction](#introduction)
  - [Tooling](#tooling)
  - [Setup up a basic `terragrunt` project](#setup-up-a-basic-terragrunt-project)

## Introduction

Infrastructure as Code (IaC) is the process of defining your infrastructure resources in source code that can be versioned 
and managed like any other software. This allows you to automate the provisioning of your infrastructure in a repeatable and 
consistent manner and allows changes to be tracked and audited. In the Azure world this can be achieved using:- [Testing Terraform Infrastructure as Code](#testing-terraform-infrastructure-as-code)
- [Testing Terraform Infrastructure as Code](#testing-terraform-infrastructure-as-code)
  - [Introduction](#introduction)
  - [Tooling](#tooling)
  - [Setup up a basic `terragrunt` project](#setup-up-a-basic-terragrunt-project)

Realistically Azure CLI is going to be hard to define in an idempotent way and ARM templates are not as flexible as Terraform
which is widely used and declarative. Therefore this document will focus on Terraform.

The challenge I am looking at here is how to test the Terraform code to ensure it is working as expected. This is not a trivial
task as a handful of terraform resources will lead to the execution of dozens of API requests to ARM to both check the current state.

## Tooling

There are various tools that we are using here to make our code cleaner and easier to test:

- [Terraform](https://www.terraform.io/) - a tool for building, changing, and versioning infrastructure safely and efficiently
- [Terragrunt](https://terragrunt.gruntwork.io/) - a thin wrapper around Terraform that provides some useful features
- [Terratest](https://terratest.gruntwork.io/) - a Go library that makes it easier to write automated tests for your infrastructure code
- [Spock](http://spockframework.org/) - a testing and specification framework for Java and Groovy applications

## Setup up a basic `terragrunt` project

First we will set up a project. I am using terragrunt to manage the terraform code as it provides a number of useful features
we will use later. The project will be a simple web application deployed to a VM behind an application gateway.

1. [Install Terraform](https://learn.hashicorp.com/terraform/getting-started/install).

2. [Install Terragrunt](https://terragrunt.gruntwork.io/docs/getting-started/install/)

3. Create the project structure

```bash
mkdir -p terraform/environments/{dev,prod}
touch terraform/environments/{dev,prod}/terragrunt.hcl
mkdir -p terraform/modules/{resource_group,vnet,vm,app_gateway}
touch terraform/modules/{resource_group,vnet,vm,app_gateway}/{main.tf,outputs.tf,variables.tf}
```

4. Create a storage account and container to store the terraform state data:

```bash
az group create --name rg-edo-terraform-state --location northeurope
az storage account create --name edoterraformstate --resource-group rg-edo-terraform-state --location northeurope --sku Standard_LRS
az storage container create --name terraform-state --account-name edoterraformstate
```

5. Get a basic terragrunt configuration in place:

in the root `terragrunt.hcl` in `terraform/environments/dev` file:

```hcl
remote_state {
  backend = "azurerm"
  generate = {
    path      = "grunt_backend.tf"
    if_exists = "overwrite_terragrunt"
  }
  config = {
    resource_group_name  = "rg-edo-terraform-state"
    storage_account_name = "edoterraformstate"
    container_name       = "terraform-state"
    key                  = "${path_relative_to_include()}/terraform.tfstate"
    tenant_id            = get_env("ARM_TENANT_ID", "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx")
    subscription_id      = get_env("ARM_SUBSCRIPTION_ID", "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx")
  }
}
```

Note: I am using a prefix of `grunt_` for the terragrunt generated files so I can easily 'gitignore' them with the pattern: `grunt_*.tf`

6. Initialise the configuration to check it is working:

```bash
❯ cd terraform/environments/dev
❯ terragrunt init

Initializing the backend...

Successfully configured the backend "azurerm"! Terraform will automatically
use this backend unless the backend configuration changes.

Initializing provider plugins...

Terraform has been successfully initialized!

You may now begin working with Terraform. Try running "terraform plan" to see
any changes that are required for your infrastructure. All Terraform commands
should now work.

If you ever set or change modules or backend configuration for Terraform,
rerun this command to reinitialize your working directory. If you forget, other
commands will detect it and remind you to do so if necessary.
```

Note, for this to work you need to either update the subscription and tenant ids in the root `terragrunt.hcl` file or set the
`ARM_SUBSCRIPTION_ID` and `ARM_TENANT_ID` environment variables.

7. Add the provider configuration to a new parent root `terragrunt.hcl` under `terraform/environments`:

```hcl
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
```

and then in each child `terragrunt.hcl` file in `terraform/environments/<environment name>` reference the parent by adding:

```hcl
include "root" {
  path = find_in_parent_folders()
}

we can then check this applies correctly:

```bash
❯ cd terraform/environments/dev
❯ terragrunt init -upgrade
...
Initializing provider plugins...
- Finding latest version of hashicorp/azuread...
- Finding latest version of hashicorp/local...
- Finding latest version of hashicorp/azurerm...
- Installing hashicorp/azurerm v3.72.0...
- Installed hashicorp/azurerm v3.72.0 (signed by HashiCorp)
- Installing hashicorp/azuread v2.41.0...
- Installed hashicorp/azuread v2.41.0 (signed by HashiCorp)
- Installing hashicorp/local v2.4.0...
- Installed hashicorp/local v2.4.0 (signed by HashiCorp)
```

In the elided output you can see the provider downloads and installs.

8. Create a basic resource group module:

```hcl
terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = ">= 2.0"
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
```

9. Utilise the module in the `dev` environment by updating the file `terraform/environments/dev/resource_group/terragrunt.hcl`:

```hcl
terraform {
  source = "${get_repo_root()}/terraform/modules/resource_group"
}

inputs = {
  location = "northeurope"
  tags     = { "environment" = "dev" }
  suffix   = "edo"
  app_name = "app"
}
```

we can then run plan to see what will be created:

```bash
❯ cd terraform/environments/dev
❯ terragrunt run-all apply
```

Note I am using the `run-all` command to ensure all the modules are applied.


10. now that is working 


```hcl
terraform {
  source = "${get_repo_root()}/terraform/modules/resource_group"
}

locals {
  common_vars = yamldecode(file(find_in_parent_folders("dev-common.yaml")))
}

inputs = {
  location = local.common_vars.location
  tags     = local.common_vars.tags
  suffix   = local.common_vars.suffix
  app_name = local.common_vars.app_name
}
```