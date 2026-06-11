# Using Terragrunt to deploy Infrastructure as Code

- [Using Terragrunt to deploy Infrastructure as Code](#using-terragrunt-to-deploy-infrastructure-as-code)
  - [Introduction](#introduction)
    - [Step 1 - Initialise the project and install tools](#step-1---initialise-the-project-and-install-tools)
    - [Step 2 - Setup the backend for state management](#step-2---setup-the-backend-for-state-management)
    - [Step 3 - Provider configuration](#step-3---provider-configuration)
    - [Step 4 - Create some terraform modules](#step-4---create-some-terraform-modules)
    - [Step 5 - deploy our first resource with Terragrunt](#step-5---deploy-our-first-resource-with-terragrunt)
    - [Step 6 - Make Terragrunt configuration driven](#step-6---make-terragrunt-configuration-driven)
    - [Step 7 - Production configuration](#step-7---production-configuration)
    - [Step 8 - Add a dependent module](#step-8---add-a-dependent-module)
  - [Conclusions](#conclusions)


## Introduction

Infrastructure as Code (IaC) is the process of defining your infrastructure resources in source code that can be versioned
and managed like any other software. This allows you to automate the provisioning of your infrastructure in a repeatable and
consistent manner and allows changes to be tracked and audited. When adopting cloud providers, like Azure and AWS, this becomes
even more important as the number of resources you are managing can grow exponentially and the ability to manage them manually
becomes impossible.

In the 'Azure world' the primary management plane control is Azure Resource Manager (ARM). We can create ARM templates and 
deploy these in Azure but I would argue these are not particularly human readable. Terraform benefits from being widely used, 
multi-cloud and declarative so I will focus on this toolset here. I will explore some new tooling that will hopefully help 
improve my terraform workflows. There are two key challenges I am interested in addressing:

- How to structure the Terraform code to make it easier to manage multiple configurations
- How to test terraform code so that we are confident it works and are very happy to destroy our infrastructure safe in the 
  knowledge we can recreate it

I am going to tackle these questions in a 3 part series:

- **Part 1** - setting up and utilising terragrunt to manage the terraform code and configuration
- **Part 2** - using terratest to test the terraform code
- **Part 3** - using spock to test the deployed

In a nutshell the key challenge I am looking at here is how to configure test Terraform code to ensure it is working as 
expected across all the environments it executes. This is not a trivial task as a handful of terraform resources will lead 
to the execution of dozens of API requests to ARM to both check the current state. I hope you enjoy my exploration of this 
topic and find it useful.

In this first article I will focus on getting a terragrunt project set up and running and give it a test drive to see how it can help
us manage our terraform code. The project will be a simple application deployed to a VM inside a virtual network.

### Step 1 - Initialise the project and install tools

Before we can get started we need to install the tooling and do some basic project initialisation. We can install the tools we need
by following the following instructions:

- [Install Terraform](https://learn.hashicorp.com/terraform/getting-started/install)
- [Install Terragrunt](https://terragrunt.gruntwork.io/docs/getting-started/install/)

We can then initialise the project with some basic commands:

```bash
mkdir -p terraform/environments/{dev,prod}
touch terraform/environments/{dev,prod}/terragrunt.hcl
mkdir -p terraform/modules/{resource_group,vnet,vm,app_gateway}
touch terraform/modules/{resource_group,vnet,vm,app_gateway}/{main.tf,outputs.tf,variables.tf}
```

### Step 2 - Setup the backend for state management

Terraform relies on having a persisted [state file](https://developer.hashicorp.com/terraform/language/state) that allows it to understand
what should exist and identify any changes it needs to make to bring your infrastructure into the desired state. This state file can be
stored locally but this is not recommended and we will store it in an Azure storage account.

To quickly create the storage account and containers we can use the Azure CLI:

```bash
az group create --name rg-edo-terraform-state --location northeurope
az storage account create --name edoterraformstate --resource-group rg-edo-terraform-state --location northeurope --sku Standard_LRS
az storage container create --name dev --account-name edoterraformstate
az storage container create --name prod --account-name edoterraformstate
```

With these created we can then define a Terragrunt backend configuration in the root `terragrunt.hcl` in `terraform/environments/dev` 
directory:

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
    container_name       = "dev"
    key                  = "${path_relative_to_include()}/terraform.tfstate"
    tenant_id            = get_env("ARM_TENANT_ID", "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx")
    subscription_id      = get_env("ARM_SUBSCRIPTION_ID", "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx")
  }
}
```

If you are familiar with a backend configuration in terraform then a lot of this is familiar but there are a few differences. 

Firstly, we have the `generate` block which instructs Terragrunt to generate a file based on the configuration with the 
`if_exists = "overwrite_terragrunt"` meaning to overwrite previous terragrunt backend configurations but error if another 
`backend.tf` exists. For the path I am using a prefix of `grunt_` for the terragrunt generated files so I can easily use 'gitignore' 
with the pattern: `grunt_*.tf`

Secondly, here we are using `get_env()` to retrieve the Azure tenant and subscription ids from the environment which simplifies 
deployments when you are targetting multiple subscriptions in dev, prod etc. We also use the `path_relative_to_include()` function
to ensure the state file is stored in a folder structure that matches the terragrunt configuration.

With our backend all configured we can initialise the configuration to check it is working:

```bash
❯ export ARM_SUBSCRIPTION_ID=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
❯ export ARM_TENANT_ID=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
❯ cd terraform/environments/dev
❯ terragrunt init
...
Initializing the backend...

Successfully configured the backend "azurerm"! Terraform will automatically
use this backend unless the backend configuration changes.
```

### Step 3 - Provider configuration

Providers in terraform are the plugins that interact with the cloud provider APIs to actually apply the configuration and 
create your infrastructure resources. Now we have our backend working, the next step is to add provider configuration to 
a `terragrunt.hcl` file under `terraform/environments`. We can place it in this folder as it will be common to all environments.

The configuration looks like this:

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
EOF
}
```

You will recognise the `generate` configuration from the backend configuration. Here we are generating a file called `grunt_provider.tf`.
The bulk of the configuration is a [heredoc](https://en.wikipedia.org/wiki/Here_document) that defines the actual providers we wish to use.
In our case we only need the Azure provider but you can add as many as you need.

With this defined we now need to reference it in the `terragrunt.hcl` file in each environment e.g. `terraform/environments/dev`. To do 
this we can use the `include` block:

```hcl
include "root" {
  path = find_in_parent_folders()
}
```

This completes the provider configuration so we can initialise again to check it is working:

```bash
❯ cd terraform/environments/dev
❯ terragrunt init -upgrade

Initializing the backend...

Initializing provider plugins...
- Finding latest version of hashicorp/azurerm...
- Using previously-installed hashicorp/azurerm v3.77.0

Terraform has been successfully initialized!
```

In the elided output you can see the provider downloads and installs have succeeded.

### Step 4 - Create some terraform modules

With the Terragrunt plumbing in place we can now define some terraform modules to use. I need 3 modules:

- Resource Group - `terraform/modules/resource_group`
- Virtual Network - `terraform/modules/vnet`
- Virtual Machine - `terraform/modules/vm`

I will share all the source code when I have completed the other articles in this series so will skip over that for now and focus 
on how to use them with Terragrunt.

### Step 5 - deploy our first resource with Terragrunt

With our terraform modules in place we can now set about using them with Terragrunt. We will start with the resource group
which is nice and straightforward. We create a `terragrunt.hcl` file in the `terraform/environments/dev/resource_group` folder
as follows:

```hcl
terraform {
  source = "${get_repo_root()}/terraform/modules/resource_group"
}

inputs = {
  location = "northeurope"
  tags     = { "environment" = "dev", project = "edoterraform" }
  suffix   = ["edo", "dev"]
  app_name = "testapp"
}
```

We can now try this out and see if it works. I use the terragrunt `run-all` command to run the same command in all the modules
which will be helpful once we start to add more shortly.

```bash
❯ cd terraform/environments/dev
❯ terragrunt run-all apply
...
Plan: 3 to add, 0 to change, 0 to destroy.

Changes to Outputs:
  + resource_group_name = "rg-edo-dev-testapp"
module.naming.random_string.first_letter: Creating...
module.naming.random_string.main: Creating...
module.naming.random_string.first_letter: Creation complete after 0s [id=g]
module.naming.random_string.main: Creation complete after 0s [id=rgsm3x4mbq03v77hb6qt9dme4yuobggu5104auz0es9yhctbup6gwrdv03ob]
azurerm_resource_group.network: Creating...
azurerm_resource_group.network: Creation complete after 1s [id=/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxx/resourceGroups/rg-edo-dev-testapp]

Apply complete! Resources: 3 added, 0 changed, 0 destroyed.

Outputs:

resource_group_name = "rg-edo-dev-testapp"
```

Success! We have deployed our resource group with Terragrunt.

### Step 6 - Make Terragrunt configuration driven

Although we have been able to deploy everything successfully we have not really made use of the power of Terragrunt yet. Critically none of the code
we have can easily handle configuration changes. For example, if we wanted to change the location of the resource group we would need to update
the file  `terraform/environments/dev/resource_group/terragrunt.hcl`. This is not ideal as we would need to update the same configuration in all
the environments and modules. This will get increasingly complicated, time-consuming and error-prone as the project grows.

Let's fix this by first externalising the common variables into a yaml file. We can create a `dev-common.yaml` file in the `terraform/environments/dev`
and populate it with the values that may change between environments, deployments or be shared between modules:

```yaml
location: northeurope

tags:
  environment: dev
  project: edoterraform

suffix:
  - edo
  - dev

app_name: testapp
```

To use this configuration we can update the `terragrunt.hcl` file in the `terraform/environments/dev/resource_group` folder to read the yaml file
and save them as local variables:

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

We can now benefit from two options to better manage the configuration going forward:

1. we can update the configuration in one place and it will be used by all the modules
2. we can override configurations with terraform variables (either passed in on the command line or set in the environment)

Note: the locals block is scoped to the module and so unfortunately I don't believe there is a way to define this at the `dev` environment level.

### Step 7 - Production configuration  

With our configurable dev environment working we can now configure the prod environment. To do so we create a `prod-common.yaml` file in 
the `terraform/environments/prod` folder:

```yaml
location: northeurope

tags:
  environment: prod
  project: edoterraform

suffix:
  - edo
  - prod

app_name: testapp
```

and then add our backend configuration by defining the `terragrunt.hcl` file in `terraform/environments/prod` as follows:

```hcl
include "root" {
  path = find_in_parent_folders()
}

remote_state {
  backend = "azurerm"
  generate = {
    path      = "grunt_backend.tf"
    if_exists = "overwrite_terragrunt"
  }
  config = {
    resource_group_name  = "rg-edo-terraform-state"
    storage_account_name = "edoterraformstate"
    container_name       = "prod"
    key                  = "${path_relative_to_include()}/terraform.tfstate"
    tenant_id            = get_env("ARM_TENANT_ID", "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx")
    subscription_id      = get_env("ARM_SUBSCRIPTION_ID", "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx")
  }
}
```

Finally, we amend the `terragrunt.hcl` file in `terraform/environments/prod/resource_group` folder to match the `dev` version only replacing 
`prod-common.yaml` for `dev-common.yaml`
```bash
❯ cd terraform/environments/prod
❯ terragrunt run-all apply

Plan: 3 to add, 0 to change, 0 to destroy.

Changes to Outputs:
  + resource_group_name = "rg-edo-prod-testapp"
module.naming.random_string.main: Creating...
module.naming.random_string.first_letter: Creating...
module.naming.random_string.first_letter: Creation complete after 0s [id=u]
module.naming.random_string.main: Creation complete after 0s [id=fgu13x066v1lw44589j8fi529wcjrkdtxgxu5dlyhkt9l2djttiq55f4oss5]
azurerm_resource_group.network: Creating...
azurerm_resource_group.network: Creation complete after 1s [id=/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxx/resourceGroups/rg-edo-prod-testapp]

Apply complete! Resources: 3 added, 0 changed, 0 destroyed.
```

Great, our prod and dev configurations both work and give the expected results.

### Step 8 - Add a dependent module

In the real world it is quite likely no matter how self-contained you try to make your modules that you will need to reference some value
from another module. For example, we may want to create a virtual network and then create a virtual machine inside that virtual network.

In our case the next thing we want to do is add the implementation calling the virtual network module. One of the variables that module 
requires is the resource group name. We could pass this in as a variable but it would be better if we could get it from the resource group
so if the `resource_group` module changes we don't need to update the way the `vnet` module is configured.

In terragrunt we can do this by using the `dependency` block. This allows us to reference the output of another module. Our definition
`terraform/environments/dev/virtual_network/terragrunt.hcl` will look like this:

```hcl
terraform {
  source = "${get_repo_root()}/terraform/modules/vnet"
}

locals {
  common_vars = yamldecode(file(find_in_parent_folders("dev-common.yaml")))
}

dependency "rg" {
  config_path = "../resource_group"
  mock_outputs = {
    name = "temp-rg"
  }
}

inputs = {
  location            = local.common_vars.location
  resource_group_name = dependency.rg.outputs.name
  tags                = local.common_vars.tags
  suffix              = local.common_vars.suffix
  purpose             = local.common_vars.app_name
  address_space       = local.common_vars.address_space
  subnets             = local.common_vars.subnets
}
```

The `dependency` block allows us to reference the output of the `resource_group` module which we do in our inputs block
for the virtual network. For simplicity for the moment I have treated the address space and subnets as common variables
and updated the `dev-common.yaml` with:

```yaml
address_space:
  - 10.0.0.0/16

subnets:
  - name: subnet1
    address_prefixes: 
      - 10.0.1.0/24
  - name: subnet2
    address_prefixes: 
      - 10.0.2.0/24
```

There are a few key points to understand here:

- give a name `rg` to the dependency on the `resource_group` module and specify where it is located relative to the `virtual_network` module
- reference the `name` output `resource_group` creates using the reference `dependency.rg.outputs.name`
- define `mock_outputs` so that when the `resource_group` module is not applied we can still test the configuration. This is
  necessary as when you are still planning the resources `terraform` will not have created the outputs yet meaning the `name` output
  the `vnet` module requires will not exist and we will get an error like this:
  
> terraform/environments/dev/resource_group/terragrunt.hcl is a dependency of terraform/environments/dev/vnet/terragrunt.hcl but detected no outputs.
> Either the target module has not been applied yet, or the module has no outputs. If this is expected, set the skip_outputs flag to true on the dependency block.

This mock_outputs approach is [documented here](https://terragrunt.gruntwork.io/docs/features/execute-terraform-commands-on-multiple-modules-at-once/#unapplied-dependency-and-mock-outputs)

With this configuration defined we can again run `terragrunt run-all plan` to check that runs ok before we try an apply:

```bash
❯ cd terraform/environments/dev
❯ terragrunt run-all apply
<< ... Detailed output shortened for brevity ...>>

Apply complete! Resources: 5 added, 0 changed, 0 destroyed.

Outputs:

subnet_address_spaces = {
  "subnet1" = tolist([
    "10.0.1.0/24",
  ])
  "subnet2" = tolist([
    "10.0.2.0/24",
  ])
}
subnet_ids = {
  "subnet1" = "/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxx/resourceGroups/rg-edo-dev-testapp/providers/Microsoft.Network/virtualNetworks/vnet-edo-dev-testapp/subnets/subnet1"
  "subnet2" = "/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxx/resourceGroups/rg-edo-dev-testapp/providers/Microsoft.Network/virtualNetworks/vnet-edo-dev-testapp/subnets/subnet2"
}
vnet_address_space = tolist([
  "10.0.0.0/16",
])
vnet_id = "/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxx/resourceGroups/rg-edo-dev-testapp/providers/Microsoft.Network/virtualNetworks/vnet-edo-dev-testapp"
vnet_name = "vnet-edo-dev-testapp"
```

## Conclusions

In this first article I have explored how to set up and use Terragrunt to manage the configuration of a terraform project. To date my experience of doing this has been
through having environment specific: GitHub actions configuration, tfvars files and command line -var options. This has worked fine to be honest and a typical project has
usually looked very similar to the terragrunt project I have built here so I am not seeing massive benefits yet.

That said I do really like the way you can read environment variables into configuration, for example, to specify the backend for a given environment. I also love being 
able to define configuration in yaml which I find much simpler to read and work with than tfvars files. Finally I think the hard dependencies between modules could be useful
to enforce sequencing of deployments. On occassion I have found terraform modules (even with a `depend_on`) run in parallel and not in the sequence I need for success.

Overall, I find Terragrunt an interesting tool that I will certainly need to try some more to get the best of and understand the key use cases where it excels.

In my next article I will look at testing the terraform code using Terratest.