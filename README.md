# Testing Terraform Infrastructure as Code

## Introduction

Infrastructure as Code (IaC) is the process of defining your infrastructure resources in source code that can be versioned 
and managed like any other software. This allows you to automate the provisioning of your infrastructure in a repeatable and 
consistent manner and allows changes to be tracked and audited. In the Azure world this can be achieved using:

- [Terraform](https://www.terraform.io/)
- [Azure Resource Manager (ARM) Templates](https://docs.microsoft.com/en-us/azure/azure-resource-manager/templates/overview)
- [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/?view=azure-cli-latest)

Realistically Azure CLI is going to be hard to define in an idempotent way and ARM templates are not as flexible as Terraform
which is widely used and declarative. Therefore this document will focus on Terraform.

The challenge I am looking at here is how to test the Terraform code to ensure it is working as expected. This is not a trivial
task as a handful of terraform resources will lead to the execution of dozens of API requests to ARM to both check the current state.

## Setup the project

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
