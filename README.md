# Testing Terraform Infrastructure as Code

- [Testing Terraform Infrastructure as Code](#testing-terraform-infrastructure-as-code)
  - [Introduction](#introduction)
  - [Tooling](#tooling)
  - [Setup up a basic `terragrunt` project](#setup-up-a-basic-terragrunt-project)
    - [Step 1. Install Terraform](#step-1-install-terraform)
    - [Step 2. Install Terragrunt](#step-2-install-terragrunt)
    - [Step 3. Create the project structure](#step-3-create-the-project-structure)
    - [Step 4. Setup the backend](#step-4-setup-the-backend)
    - [Step 5. Terragrunt backend configuration](#step-5-terragrunt-backend-configuration)
    - [Step 6. Terragrunt initialize](#step-6-terragrunt-initialize)
    - [Step 7. Provider configuration](#step-7-provider-configuration)
    - [Step 8. Create a module](#step-8-create-a-module)
    - [Step 9. Apply the module in `dev`](#step-9-apply-the-module-in-dev)
    - [Step 10. Externalise common configuration](#step-10-externalise-common-configuration)
    - [Step 11. Production configuration](#step-11-production-configuration)
    - [Step 12. Add vnet module](#step-12-add-vnet-module)
    - [Step 13. Update production configuration](#step-13-update-production-configuration)
  - [Testing the project with terratest](#testing-the-project-with-terratest)
    - [Step 1. Install go](#step-1-install-go)
    - [Step 2. Create a basic test](#step-2-create-a-basic-test)
    - [Step 3. Execute the test](#step-3-execute-the-test)
    - [Step 4. Add a test for the vnet module](#step-4-add-a-test-for-the-vnet-module)
    - [Step 5. Making a more useful clean test](#step-5-making-a-more-useful-clean-test)
    - [Step 6. (ASIDE) refactor getOutputMap](#step-6-aside-refactor-getoutputmap)
    - [Step 6. Aren't we building and destroying the actual dev environment!](#step-6-arent-we-building-and-destroying-the-actual-dev-environment)
  - [Testing deployed infrastructure](#testing-deployed-infrastructure)
    - [Why?](#why)
    - [Setting up the integration test](#setting-up-the-integration-test)
      - [Step 1. Create a new directory and initialise gradle](#step-1-create-a-new-directory-and-initialise-gradle)
      - [Step 2. Configure the gradle build](#step-2-configure-the-gradle-build)
      - [Step 3. Create a basic test](#step-3-create-a-basic-test)
    - [Step 4. Try the test out to check it fails as expected](#step-4-try-the-test-out-to-check-it-fails-as-expected)
    - [Step 5. Fix the test](#step-5-fix-the-test)
    - [Creating more configurable tests](#creating-more-configurable-tests)
    - [Creating more realistic / useful tests](#creating-more-realistic--useful-tests)

## Introduction

Infrastructure as Code (IaC) is the process of defining your infrastructure resources in source code that can be versioned 
and managed like any other software. This allows you to automate the provisioning of your infrastructure in a repeatable and 
consistent manner and allows changes to be tracked and audited. In the Azure world this can be achieved using:- [Testing Terraform Infrastructure as Code](#testing-terraform-infrastructure-as-code)

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

### Step 1. Install Terraform

Follow the [Install Terraform](https://learn.hashicorp.com/terraform/getting-started/install) instructions.

### Step 2. Install Terragrunt

Follow the [Install Terragrunt](https://terragrunt.gruntwork.io/docs/getting-started/install/) instructions.

### Step 3. Create the project structure

Execute the following commands:

```bash
mkdir -p terraform/environments/{dev,prod}
touch terraform/environments/{dev,prod}/terragrunt.hcl
mkdir -p terraform/modules/{resource_group,vnet,vm,app_gateway}
touch terraform/modules/{resource_group,vnet,vm,app_gateway}/{main.tf,outputs.tf,variables.tf}
```

### Step 4. Setup the backend

Create a storage account and container to store the terraform state data:

```bash
az group create --name rg-edo-terraform-state --location northeurope
az storage account create --name edoterraformstate --resource-group rg-edo-terraform-state --location northeurope --sku Standard_LRS
az storage container create --name dev --account-name edoterraformstate
az storage container create --name prod --account-name edoterraformstate
```

### Step 5. Terragrunt backend configuration

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
    container_name       = "dev"
    key                  = "${path_relative_to_include()}/terraform.tfstate"
    tenant_id            = get_env("ARM_TENANT_ID", "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx")
    subscription_id      = get_env("ARM_SUBSCRIPTION_ID", "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx")
  }
}
```

Note: I am using a prefix of `grunt_` for the terragrunt generated files so I can easily 'gitignore' them with the pattern: `grunt_*.tf`

### Step 6. Terragrunt initialize

Initialise the configuration to check it is working:

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

### Step 7. Provider configuration

Add the provider configuration to a new parent root `terragrunt.hcl` under `terraform/environments`:

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

### Step 8. Create a module

Create a basic resource group module:

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

### Step 9. Apply the module in `dev`

Utilise the module in the `dev` environment by updating the file `terraform/environments/dev/resource_group/terragrunt.hcl`:

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

we can then run apply to create:

<details>
<summary>Terragrunt dev apply output<summary>

```bash
❯ cd terraform/environments/dev
❯ terragrunt run-all apply
INFO[0000] The stack at /home/edoatley/source/edoatley/azure-tf-iac-testing/terraform/environments/dev will be processed in the following order for command apply:
Group 1
- Module /home/edoatley/source/edoatley/azure-tf-iac-testing/terraform/environments/dev/resource_group
 
Are you sure you want to run 'terragrunt apply' in each folder of the stack described above? (y/n) y

Terraform used the selected providers to generate the following execution
plan. Resource actions are indicated with the following symbols:
  + create

Terraform will perform the following actions:

  # azurerm_resource_group.network will be created
  + resource "azurerm_resource_group" "network" {
      + id       = (known after apply)
      + location = "northeurope"
      + name     = "rg-edo-dev-testapp"
      + tags     = (known after apply)
    }

  # module.naming.random_string.first_letter will be created
  + resource "random_string" "first_letter" {
      + id          = (known after apply)
      + length      = 1
      + lower       = true
      + min_lower   = 0
      + min_numeric = 0
      + min_special = 0
      + min_upper   = 0
      + number      = false
      + numeric     = false
      + result      = (known after apply)
      + special     = false
      + upper       = false
    }

  # module.naming.random_string.main will be created
  + resource "random_string" "main" {
      + id          = (known after apply)
      + length      = 60
      + lower       = true
      + min_lower   = 0
      + min_numeric = 0
      + min_special = 0
      + min_upper   = 0
      + number      = true
      + numeric     = true
      + result      = (known after apply)
      + special     = false
      + upper       = false
    }

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

</details>

Note: I am using the `run-all` command to ensure all the modules are applied.

### Step 10. Externalise common configuration

Now that `dev` is working we can refactor to pull out the common variables into a yaml file so we do not need to repeat them in the other modules.
To do so we first create a `dev-common.yaml` file in the `terraform/environments/dev` folder:

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

we can then update the `terragrunt.hcl` file in the `terraform/environments/dev/resource_group` folder to use the common variables:

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

Note: the locals block is scoped to the module and so unfortunately I don't believe there is a way to define this at the `dev` environment level.

### Step 11. Production configuration  

Now dev is working we can check production works as well by creating a `prod-common.yaml` file in the `terraform/environments/prod` folder:

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

Next, update the `terragrunt.hcl` file in `terraform/environments/prod` to define a backend:

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

Finally amend the `terragrunt.hcl` file in `terraform/environments/prod/resource_group` folder to match the `dev` version
and we can test it out.

<details>
<summary>Terragrunt apply prod output<summary>

```bash
❯ cd terraform/environments/prod
❯ terragrunt run-all apply
INFO[0000] The stack at /home/edoatley/source/edoatley/azure-tf-iac-testing/terraform/environments/prod will be processed in the following order for command apply:
Group 1
- Module /home/edoatley/source/edoatley/azure-tf-iac-testing/terraform/environments/prod/resource_group
 
Are you sure you want to run 'terragrunt apply' in each folder of the stack described above? (y/n) y

Terraform used the selected providers to generate the following execution
plan. Resource actions are indicated with the following symbols:
  + create

Terraform will perform the following actions:

  # azurerm_resource_group.network will be created
  + resource "azurerm_resource_group" "network" {
      + id       = (known after apply)
      + location = "northeurope"
      + name     = "rg-edo-prod-testapp"
      + tags     = (known after apply)
    }

  # module.naming.random_string.first_letter will be created
  + resource "random_string" "first_letter" {
      + id          = (known after apply)
      + length      = 1
      + lower       = true
      + min_lower   = 0
      + min_numeric = 0
      + min_special = 0
      + min_upper   = 0
      + number      = false
      + numeric     = false
      + result      = (known after apply)
      + special     = false
      + upper       = false
    }

  # module.naming.random_string.main will be created
  + resource "random_string" "main" {
      + id          = (known after apply)
      + length      = 60
      + lower       = true
      + min_lower   = 0
      + min_numeric = 0
      + min_special = 0
      + min_upper   = 0
      + number      = true
      + numeric     = true
      + result      = (known after apply)
      + special     = false
      + upper       = false
    }

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

Outputs:

resource_group_name = "rg-edo-prod-testapp"
```

</details>

### Step 12. Add vnet module

Next we will add the vnet module which will require us passing the resource group name in. To do this 
we need to use the output `resource_group_name` from the module `resource_group` and pass that into the inputs
for the `vnet` module. To do this we need to create a `terragrunt.hcl` file in `terraform/environments/dev/vnet`:

```hcl
terraform {
  source = "${get_repo_root()}/terraform/modules/vnet"
}

locals {
  common_vars = yamldecode(file(find_in_parent_folders("dev-common.yaml")))
}

dependency "rg" {
  config_path = "../resource_group"
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

We can now plan and then apply the configuration to check it works:

<details>
<summary>Terragrunt plan dev output<summary>

```bash
❯ cd terraform/environments/dev
❯ terragrunt run-all plan
❯ terragrunt run-all plan
INFO[0000] The stack at /home/edoatley/source/edoatley/azure-tf-iac-testing/terraform/environments/dev will be processed in the following order for command plan:
Group 1
- Module /home/edoatley/source/edoatley/azure-tf-iac-testing/terraform/environments/dev/resource_group

Group 2
- Module /home/edoatley/source/edoatley/azure-tf-iac-testing/terraform/environments/dev/vnet
 
<< ... Removed resource group output for brevity ... >>

Note: You didn't use the -out option to save this plan, so Terraform can't
guarantee to take exactly these actions if you run "terraform apply" now.
ERRO[0015] Module /home/edoatley/source/edoatley/azure-tf-iac-testing/terraform/environments/dev/vnet has finished with an error: /home/edoatley/source/edoatley/azure-tf-iac-testing/terraform/environments/dev/resource_group/terragrunt.hcl is a dependency of /home/edoatley/source/edoatley/azure-tf-iac-testing/terraform/environments/dev/vnet/terragrunt.hcl but detected no outputs. Either the target module has not been applied yet, or the module has no outputs. If this is expected, set the skip_outputs flag to true on the dependency block.  prefix=[/home/edoatley/source/edoatley/azure-tf-iac-testing/terraform/environments/dev/vnet] 
ERRO[0015] 1 error occurred:
        * /home/edoatley/source/edoatley/azure-tf-iac-testing/terraform/environments/dev/resource_group/terragrunt.hcl is a dependency of /home/edoatley/source/edoatley/azure-tf-iac-testing/terraform/environments/dev/vnet/terragrunt.hcl but detected no outputs. Either the target module has not been applied yet, or the module has no outputs. If this is expected, set the skip_outputs flag to true on the dependency block.
 
ERRO[0015] Unable to determine underlying exit code, so Terragrunt will exit with error code 1
```

</details>

Note: the error is expected as we have not applied the `resource_group` module yet and you cannot access outputs of an un-applied terraform module.
We can work around this by adding a `mock_outputs` block to the `dependency` block which will be used when modules are not applied as described in the official [documentation](https://terragrunt.gruntwork.io/docs/features/execute-terraform-commands-on-multiple-modules-at-once/#unapplied-dependency-and-mock-outputs).
Practically this means updating the `terragrunt.hcl` file in `terraform/environments/dev/vnet` making the `dependency` block now:

```hcl
dependency "rg" {
  config_path = "../resource_group"
  mock_outputs = {
    name = "temp-rg"
  }
}
```

The `terragrunt run-all plan` runs ok and we can try an apply:

<details>
<summary>Terragrunt apply dev output<summary>

```bash
❯ cd terraform/environments/dev
❯ terragrunt run-all apply
❯ terragrunt run-all apply
INFO[0000] The stack at /home/edoatley/source/edoatley/azure-tf-iac-testing/terraform/environments/dev will be processed in the following order for command apply:
Group 1
- Module /home/edoatley/source/edoatley/azure-tf-iac-testing/terraform/environments/dev/resource_group

Group 2
- Module /home/edoatley/source/edoatley/azure-tf-iac-testing/terraform/environments/dev/vnet
 
Are you sure you want to run 'terragrunt apply' in each folder of the stack described above? (y/n) y

Terraform used the selected providers to generate the following execution
plan. Resource actions are indicated with the following symbols:
  + create

Terraform will perform the following actions:

<< ... Detailed output shortened for brevity ...>>

Plan: 5 to add, 0 to change, 0 to destroy.

Changes to Outputs:
  + subnet_address_spaces = {
      + subnet1 = [
          + "10.0.1.0/24",
        ]
      + subnet2 = [
          + "10.0.2.0/24",
        ]
    }
  + subnet_ids            = {}
  + vnet_address_space    = [
      + "10.0.0.0/16",
    ]
  + vnet_id               = (known after apply)
  + vnet_name             = "vnet-edo-dev-testapp"
module.naming.random_string.first_letter: Creating...
module.naming.random_string.main: Creating...
module.naming.random_string.first_letter: Creation complete after 0s [id=m]
module.naming.random_string.main: Creation complete after 0s [id=biskuph6vvdb2nl0n0u6v4vlyv9z10gctu9ngpjl4vfobm08zkqzsslbsyuc]
azurerm_virtual_network.vnet: Creating...
azurerm_virtual_network.vnet: Creation complete after 4s [id=/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxx/resourceGroups/rg-edo-dev-testapp/providers/Microsoft.Network/virtualNetworks/vnet-edo-dev-testapp]
azurerm_subnet.subnet["subnet2"]: Creating...
azurerm_subnet.subnet["subnet1"]: Creating...
azurerm_subnet.subnet["subnet1"]: Creation complete after 4s [id=/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxx/resourceGroups/rg-edo-dev-testapp/providers/Microsoft.Network/virtualNetworks/vnet-edo-dev-testapp/subnets/subnet1]
azurerm_subnet.subnet["subnet2"]: Creation complete after 8s [id=/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxx/resourceGroups/rg-edo-dev-testapp/providers/Microsoft.Network/virtualNetworks/vnet-edo-dev-testapp/subnets/subnet2]

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

</details>

### Step 13. Update production configuration

We can now make the same changes to the production configuration by creating a `terragrunt.hcl` file in `terraform/environments/prod/vnet`:
which is identical to the `dev` version other than the common variables file reference.

## Testing the project with terratest

Now we have a basic project set up we can look at testing it. We will use terratest to do this. Terratest is a Go library that makes it easier to write automated tests for your infrastructure code. It provides a number of helper functions to make it easier to test the terraform code.

### Step 1. Install go

Follow the [Install Go](https://golang.org/doc/install) instructions.

### Step 2. Create a basic test

First we create a new directory `test` and add a file `terraform_test.go` with the following contents:

```go
package test

import (
  "testing"
  "github.com/gruntwork-io/terratest/modules/terraform"
  "github.com/stretchr/testify/assert"
)

// constant to define terraform directory we want to test
var terraformParentDir string = "../terraform/environments/dev"

// An example of how to test the our simple Terraform resource_group module
func TestTerraformBasicExample(t *testing.T) {
  t.Parallel()

  // Construct the terraform options setting the path to the Terraform code we want to test and and specifying the terragrunt binary.
  terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
    TerraformDir: terraformParentDir,
    TerraformBinary: "terragrunt",
  })
  
  // At the end of the test, run `terragrunt run-all destroy` to clean up any resources that were created.
  defer terraform.TgDestroyAll(t, terraformOptions)

  // Run `terragrunt run-all apply`. Fail the test if there are any errors.
  terraform.TgApplyAll(t, terraformOptions)

  // Call our helper to get the resource group name created by the Terraform code.
  rgName := getOutput(t, terraformOptions, "/resource_group", "resource_group_name")

  // Verify that our Resource Group has the right name
  assert.Equal(t, "rg-edo-dev-testapp", rgName)
}

// helper function to simplify fetching the outputs when using terragrunt run-all
func getOutput(t *testing.T, terraformOptions *terraform.Options, dir string, output string) string {
  terraformOptions.TerraformDir = terraformParentDir + dir
  outputValue := terraform.Output(t, terraformOptions, output)
  terraformOptions.TerraformDir = terraformParentDir
  return outputValue
}
```

The code is commented to explain in detail what is happening. The key points are:
- run `terragrunt run-all apply` to apply the terraform code
- extract the output of the resource group module
- verify the output is as expected

### Step 3. Execute the test

To actually run our go code we need to initialize the module first

```bash
cd test
go mod init github.com/edoatley/azure-tf-iac-testing/test
go mod tidy
```

and then we can run the test:

```bash
go test -v -run TestTerraformBasicExample -timeout 10m
```

So picking out the key parts of the output, we see the test:


1. Running the `terragrunt run-all apply` command:

```bash
```bash
TestTerraformBasicExample 2023-09-22T13:01:06+01:00 retry.go:91: terragrunt [run-all apply -input=false -auto-approve -lock=false --terragrunt-non-interactive]
TestTerraformBasicExample 2023-09-22T13:01:06+01:00 logger.go:66: Running command terragrunt with args [run-all apply -input=false -auto-approve -lock=false --terragrunt-non-interactive]
TestTerraformBasicExample 2023-09-22T13:01:07+01:00 logger.go:66: time=2023-09-22T13:01:07+01:00 level=info msg=The stack at /home/edoatley/source/edoatley/azure-tf-iac-testing/terraform/environments/dev will be processed in the following order for command apply:
TestTerraformBasicExample 2023-09-22T13:01:07+01:00 logger.go:66: Group 1
TestTerraformBasicExample 2023-09-22T13:01:07+01:00 logger.go:66: - Module /home/edoatley/source/edoatley/azure-tf-iac-testing/terraform/environments/dev
TestTerraformBasicExample 2023-09-22T13:01:07+01:00 logger.go:66: - Module /home/edoatley/source/edoatley/azure-tf-iac-testing/terraform/environments/dev/resource_group
TestTerraformBasicExample 2023-09-22T13:01:07+01:00 logger.go:66: 
TestTerraformBasicExample 2023-09-22T13:01:07+01:00 logger.go:66: Group 2
TestTerraformBasicExample 2023-09-22T13:01:07+01:00 logger.go:66: - Module /home/edoatley/source/edoatley/azure-tf-iac-testing/terraform/environments/dev/vnet
```

2. Applying ok and outputting the expected resource_group_name

```bash
TestTerraformBasicExample 2023-09-22T13:01:35+01:00 logger.go:66: Apply complete! Resources: 3 added, 0 changed, 0 destroyed.
TestTerraformBasicExample 2023-09-22T13:01:35+01:00 logger.go:66: 
TestTerraformBasicExample 2023-09-22T13:01:35+01:00 logger.go:66: Outputs:
TestTerraformBasicExample 2023-09-22T13:01:35+01:00 logger.go:66: 
TestTerraformBasicExample 2023-09-22T13:01:35+01:00 logger.go:66: resource_group_name = "rg-edo-dev-testapp"
```

3. Getting the output of the resource group module

```bash
TestTerraformBasicExample 2023-09-22T13:02:14+01:00 retry.go:91: terragrunt [output -no-color -json resource_group_name --terragrunt-non-interactive]
TestTerraformBasicExample 2023-09-22T13:02:14+01:00 logger.go:66: Running command terragrunt with args [output -no-color -json resource_group_name --terragrunt-non-interactive]
TestTerraformBasicExample 2023-09-22T13:02:14+01:00 logger.go:66: "rg-edo-dev-testapp"
```

4. Successfully destroying the resources

```bash
TestTerraformBasicExample 2023-09-22T13:02:14+01:00 retry.go:91: terragrunt [run-all destroy -auto-approve -input=false -lock=false --terragrunt-non-interactive]
TestTerraformBasicExample 2023-09-22T13:02:14+01:00 logger.go:66: Running command terragrunt with args [run-all destroy -auto-approve -input=false -lock=false --terragrunt-non-interactive]
...
TestTerraformBasicExample 2023-09-22T13:04:05+01:00 logger.go:66: azurerm_resource_group.network: Destroying... [id=/subscriptions/22222222-2222-2222-2222-222222222222/resourceGroups/rg-edo-dev-testapp]
TestTerraformBasicExample 2023-09-22T13:04:15+01:00 logger.go:66: azurerm_resource_group.network: Still destroying... [id=/subscriptions/4a0c9b39-20e2-47b5-a648-...05e5/resourceGroups/rg-edo-dev-testapp, 10s elapsed]
TestTerraformBasicExample 2023-09-22T13:04:21+01:00 logger.go:66: azurerm_resource_group.network: Destruction complete after 16s
TestTerraformBasicExample 2023-09-22T13:04:21+01:00 logger.go:66: module.naming.random_string.first_letter: Destroying... [id=l]
TestTerraformBasicExample 2023-09-22T13:04:21+01:00 logger.go:66: module.naming.random_string.main: Destroying... [id=6hlumrzrmcy3lndmiz2t03kisjq6pe1nhfe00tjk0xiwi8pvurokmoz4d62p]
TestTerraformBasicExample 2023-09-22T13:04:21+01:00 logger.go:66: module.naming.random_string.main: Destruction complete after 0s
TestTerraformBasicExample 2023-09-22T13:04:21+01:00 logger.go:66: module.naming.random_string.first_letter: Destruction complete after 0s
TestTerraformBasicExample 2023-09-22T13:04:21+01:00 logger.go:66: 
TestTerraformBasicExample 2023-09-22T13:04:21+01:00 logger.go:66: Destroy complete! Resources: 3 destroyed.
TestTerraformBasicExample 2023-09-22T13:04:21+01:00 logger.go:66: 
```

5. Confirming that the test has passed:

```bash
TestTerraformBasicExample 2023-09-22T13:04:21+01:00 logger.go:66: Destroy complete! Resources: 3 destroyed.
TestTerraformBasicExample 2023-09-22T13:04:21+01:00 logger.go:66: 
--- PASS: TestTerraformBasicExample (194.80s)
PASS
ok      github.com/edoatley/azure-tf-iac-testing        194.814s
```

### Step 4. Add a test for the vnet module

We now have a working test but it is not fast! It takes around 3 minutes to run. This is because we are applying the whole stack. 

Let's try creating a test that just runs the resource_group_module. To do this we will add another test to our `terraform_test.go` file:

```go
func TestTerraformResourceGroup(t *testing.T) {
  t.Parallel()

  terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
    TerraformDir: terraformParentDir + "/resource_group",
    TerraformBinary: "terragrunt",
  })
  

  defer terraform.Destroy(t, terraformOptions)

  terraform.InitAndApply(t, terraformOptions)
  rgName := terraform.Output(t, terraformOptions, "resource_group_name")
  assert.Equal(t, "rg-edo-dev-testapp", rgName)
}
```

This is a lot faster and could be beneficial in some cases:

```bash
--- PASS: TestTerraformResourceGroup (83.21s)
PASS
ok      github.com/edoatley/azure-tf-iac-testing        83.220s
```

but if what you want to test deploying all the modules then you are likely better off using the `run-all` command.

### Step 5. Making a more useful clean test

The long running nature of an apply-all test does make for an interesting challenge where you either:

- run a long running test many times 
- have a single long running test that needs to check many things rather than the ideal of making a single assertion per test

To refactor around this a little I created the following test this:

```go
func TestTerraformRunAll(t *testing.T) {
  t.Parallel()

  terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
    TerraformDir: terraformParentDir,
    TerraformBinary: "terragrunt",
  })
  
  // At the end of the test, run `terragrunt run-all destroy` to clean up any resources that were created.
  defer terraform.TgDestroyAll(t, terraformOptions)

  // Run `terragrunt run-all apply`. Fail the test if there are any errors.
  terraform.TgApplyAll(t, terraformOptions)

  validateResourceGroup(t, terraformOptions)

  validateVirtualNetwork(t, terraformOptions)
}

// helper function to validate the resource group
func validateResourceGroup(t *testing.T, terraformOptions *terraform.Options) {
  rgName := getOutput(t, terraformOptions, "/resource_group", "resource_group_name")
  assert.Equal(t, "rg-edo-dev-testapp", rgName)
}

// helper function to validate the vnet name and CIDR ranges
func validateVirtualNetwork(t *testing.T, terraformOptions *terraform.Options) {
  vnetName := getOutput(t, terraformOptions, "/virtual_network", "vnet_name")
  assert.Equal(t, "vnet-edo-dev-testapp", vnetName)

  vnetAddressSpaces := getOutput(t, terraformOptions, "/virtual_network", "vnet_address_space")
  assert.Equal(t, "10.0.0.0/16", vnetAddressSpaces[0])

  subnetAddressSpaces := getOutputMap(t, terraformOptions, "/virtual_network", "subnet_address_spaces")
  assert.Equal(t, "10.0.1.0/24", subnetAddressSpaces["subnet1"][0])
  assert.Equal(t, "10.0.2.0/24", subnetAddressSpaces["subnet2"][0])
}
```

Though we are breaking the 'one assertion rule' I would argue that:

a) the outcome we are measuring is whether the terraform we will use to deploy our infrastructure works correctly
b) running lots of identical tests with many assertions is going to take a lot of time and slow down feedback

### Step 6. (ASIDE) refactor getOutputMap

Now it was all working I took the opportunity to improve and refactor a bit. I observed that there is a nice method in terratest 
called `terraform.OutputStructE` which I could use to simplify the `validateVirtualNetwork` method:

```go
func getOutput(t *testing.T, terraformOptions *terraform.Options, dir string, outputRequested string, output interface{}) {
  terraformOptions.TerraformDir = terraformParentDir + dir
	err := terraform.OutputStructE(t, terraformOptions, outputRequested, output)
  if err != nil {
    t.Fatalf("Failed to fetch output %s: %v", output, err)
  }
}
```

in this you pass in a pointer to the object you wish it to populate which for the subnet ranges means I can call it as follows:

```go
  var subnetAddressSpaces map[string][]string
  getOutput(t, terraformOptions, "/virtual_network", "subnet_address_spaces", &subnetAddressSpaces)
  fmt.Println("XXX -> subnetAddressSpaces: " + fmt.Sprintf("%v", subnetAddressSpaces))
```

### Step 6. Aren't we building and destroying the actual dev environment!

In the test above you may notice we are using the real dev environment. This is not ideal as we are actually deploying
and destroying the real infrastructure so we are not testing the code in isolation.

The challenge here is to allow the variables in the *-common.yaml to be overridden. This seems to be a difficult task and so
I think moving to `tfvars` files instead is a better option. Fortunately that will be easier now we have tests!

So the first step is to override the suffix variable as this is used to handle the naming:

```go
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir:    terraformParentDir,
		TerraformBinary: terraformBinary,
		Vars: map[string]interface{}{
			"suffix": []string{"terratest", "edo"},
		},
	})
```

I then see my test failing with the following errors:

```bash
--- FAIL: TestTerraformRunAll (137.78s)
    --- FAIL: TestTerraformRunAll/Resource_Group (0.64s)
        terraform_test.go:55: 
                Error Trace:    /home/edoatley/source/edoatley/azure-tf-iac-testing/test/terraform_test.go:55
                                                        /home/edoatley/source/edoatley/azure-tf-iac-testing/test/terraform_test.go:43
                Error:          Not equal: 
                                expected: "rg-edo-dev-testapp"
                                actual  : "rg-terratest-edo-testapp"
                            
                                Diff:
                                --- Expected
                                +++ Actual
                                @@ -1 +1 @@
                                -rg-edo-dev-testapp
                                +rg-terratest-edo-testapp
                Test:           TestTerraformRunAll/Resource_Group
    --- FAIL: TestTerraformRunAll/Virtual_Network (3.47s)
        terraform_test.go:61: 
                Error Trace:    /home/edoatley/source/edoatley/azure-tf-iac-testing/test/terraform_test.go:61
                                                        /home/edoatley/source/edoatley/azure-tf-iac-testing/test/terraform_test.go:48
                Error:          Not equal: 
                                expected: "vnet-edo-dev-testapp"
                                actual  : "vnet-terratest-edo-testapp"
                            
                                Diff:
                                --- Expected
                                +++ Actual
                                @@ -1 +1 @@
                                -vnet-edo-dev-testapp
                                +vnet-terratest-edo-testapp
                Test:           TestTerraformRunAll/Virtual_Network
```

This is actually exactly what we want we just need to correct our assertions to expect the new names:

```go
// update RG assertion
assert.Equal(t, "rg-terratest-edo-testapp", rgName)

// update vnet assertions
assert.Equal(t, "vnet-terratest-edo-testapp", vnetName)
```

## Testing deployed infrastructure

### Why?

The terratest code we have written gives us confdence that the inputs and outputs of the terraform code are as expected. However, if there are checks 
that terraform cannot make or are based on attributes not visible to terraform then we need to test the deployed infrastructure more directly.

One approach to do this is to utilise the Azure Resource Manager to validate what has been deployed matches expectations. This is the approach I have taken here.

### Setting up the integration test

#### Step 1. Create a new directory and initialise gradle

```bash
> mkdir it && cd it
> gradle init
Starting a Gradle Daemon (subsequent builds will be faster)

Select type of project to generate:
  1: basic
  2: application
  3: library
  4: Gradle plugin
Enter selection (default: basic) [1..4] 1

Select build script DSL:
  1: Groovy
  2: Kotlin
Enter selection (default: Groovy) [1..2] 1

Project name (default: it): 

> Task :init
Get more help with your project: Learn more about Gradle by exploring our samples at https://docs.gradle.org/6.7/samples

BUILD SUCCESSFUL in 17s
2 actionable tasks: 2 executed
```

#### Step 2. Configure the gradle build

Firstly we shall update the gradle version by running

```bash
./gradlew wrapper --gradle-version 8.4
```

Now we can populate the generated build.gradle file:

```groovy
plugins {
    id 'java'
    id 'groovy'
    id("com.adarshr.test-logger") version "3.2.0"
    id("io.freefair.lombok") version "8.3"
}

group = 'uk.edoatley'
version = '0.0.1-SNAPSHOT'
sourceCompatibility = JavaVersion.VERSION_17

repositories {
    mavenCentral()
    jcenter()
}

dependencies {
    // Azure deps
    implementation 'com.azure:azure-identity:1.8.2'
    implementation 'com.azure:azure-core-http-netty:1.13.1'
    implementation 'com.azure.resourcemanager:azure-resourcemanager:2.30.0'
    implementation 'com.azure.resourcemanager:azure-resourcemanager-security:1.0.0-beta.5'


    // Spock
    testImplementation 'org.codehaus.groovy:groovy-all:3.0.16'
    testImplementation 'org.spockframework:spock-core:2.3-groovy-3.0'

    // add SpringBoot support for ease of configurability
    testImplementation 'org.springframework.boot:spring-boot-starter-test:2.7.15'
    testImplementation 'org.spockframework:spock-spring:2.3-groovy-3.0'
}


test {
    useJUnitPlatform()
    testLogging { 
        showStandardStreams = true
        exceptionFormat "full"
        minGranularity = 3
    }
}
```

and we can validate that is all good with:

```bash
./gradlew clean build
```

#### Step 3. Create a basic test

Firstly let's create our directory structure:

```bash
mkdir -p src/test/groovy/uk/edoatley/azure/it
mkdir -p src/test/java/uk/edoatley/azure/it
mkdir -p src/test/resources
```

Next we create our configuration class in java as personally I found it simpler:

```java
package uk.edoatley.azure.it;

import com.azure.core.management.AzureEnvironment;
import com.azure.core.management.profile.AzureProfile;
import com.azure.identity.DefaultAzureCredential;
import com.azure.identity.DefaultAzureCredentialBuilder;
import com.azure.resourcemanager.AzureResourceManager;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
public class AzureResourceManagerConfiguration {

    @Value("${azure.tenant}")
    private String tenant;

    @Value("${azure.subscription}")
    private String subscription;

    @Bean
    public AzureResourceManager azureResourceManager() {
        AzureProfile toolingAzureProfile = new AzureProfile(tenant, subscription, AzureEnvironment.AZURE);
        DefaultAzureCredential toolingCredential = new DefaultAzureCredentialBuilder()
                .authorityHost(toolingAzureProfile.getEnvironment().getActiveDirectoryEndpoint())
                .build();
        return AzureResourceManager
                .authenticate(toolingCredential, toolingAzureProfile)
                .withDefaultSubscription();
    }
}
```

This class is creating an `AzureResourceManager` bean that we can use to query ARM for details of the deployed infrastructure.
The AzureResourceManager SDK is documented [here](https://learn.microsoft.com/en-us/java/api/overview/azure/resourcemanager-readme?view=azure-java-stable)

Next we can create our groovy test for which we will use spock:

```groovy
import com.azure.resourcemanager.AzureResourceManager
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.test.context.SpringBootTest
import spock.lang.Specification

@SpringBootTest
class AzureIntegrationTest extends Specification {

    @Autowired
    private AzureResourceManager azureResourceManager

    def "Resource group exists"() {
        when:
        def groups = azureResourceManager.resourceGroups()

        then:
        groups.list().size() > 0
        groups.list().any { it.name() == "rg-edo-dev-testapp"}
    }

}
```

finally we need to create an `application.yaml` file like this in `src/test/resources` setting the tenant and subscription values:

```yaml
azure:
  tenant: "11111111-1111-1111-1111-111111111111"
  subscription: "22222222-2222-2222-2222-222222222222"
```

### Step 4. Try the test out to check it fails as expected

We can now try to run our test and check it fails (as we have not deployed anything yet!)

```bash
> ./gradlew clean build
...
AzureIntegrationTest > Resource group exists STANDARD_OUT
    2023-10-19 17:06:23.463  INFO 378627 --- [    Test worker] c.azure.identity.ChainedTokenCredential  : Azure Identity => Attempted credential EnvironmentCredential is unavailable.
    2023-10-19 17:06:24.536  INFO 378627 --- [onPool-worker-1] c.azure.identity.ChainedTokenCredential  : Azure Identity => Attempted credential ManagedIdentityCredential is unavailable.
    2023-10-19 17:06:24.558  INFO 378627 --- [onPool-worker-1] c.azure.identity.ChainedTokenCredential  : Azure Identity => Attempted credential SharedTokenCacheCredential is unavailable.
    2023-10-19 17:06:24.587  INFO 378627 --- [onPool-worker-1] c.azure.identity.ChainedTokenCredential  : Azure Identity => Attempted credential IntelliJCredential is unavailable.
    2023-10-19 17:06:25.039  INFO 378627 --- [onPool-worker-1] com.azure.identity.AzureCliCredential    : Azure Identity => getToken() result for scopes [https://management.core.windows.net//.default]: SUCCESS
    2023-10-19 17:06:25.039  INFO 378627 --- [onPool-worker-1] c.azure.identity.ChainedTokenCredential  : Azure Identity => Attempted credential AzureCliCredential returns a token
    2023-10-19 17:06:25.040  INFO 378627 --- [onPool-worker-1] c.a.c.implementation.AccessTokenCache    : {"az.sdk.message":"Acquired a new access token."}
AzureIntegrationTest

  Test Resource group exists FAILED (2.3s)

  Condition not satisfied:

  groups.list().any { it.name() == "rg-edo-dev-testapp"}
  |      |      |
  |      |      false
  |      <com.azure.resourcemanager.resources.fluentcore.utils.PagedConverter$PagedIterableImpl@52c2d1e8 pagedIterable=com.azure.core.http.rest.PagedIterable@1a67bcaf mapper=com.azure.resourcemanager.resources.fluentcore.arm.collection.implementation.ReadableWrappersImpl$$Lambda$725/0x0000000801758678@4f4b7a2e pageMapper=com.azure.resourcemanager.resources.fluentcore.utils.PagedConverter$PagedIterableImpl$$Lambda$727/0x0000000801758f98@334ac669 pagedFlux=PagedFlux firstPageRetriever=null nextPageRetriever=null pagedFlux=PagedFlux batchSize=1 pageRetrieverSyncProvider=null defaultPageSize=null continuationPredicate=null flux=PagedFlux iterable=null>
  <com.azure.resourcemanager.resources.implementation.ResourceGroupsImpl@2740585b logger=com.azure.core.util.logging.ClientLogger@5aa461 resourceManager=com.azure.resourcemanager.resources.ResourceManager@f3e6876>
      at AzureIntegrationTest.Resource group exists(AzureIntegrationTest.groovy:20)

FAILURE: Executed 1 tests in 5.8s (1 failed)
```

We can see here there is no resource group named "rg-edo-dev-testapp" is the subscription we configured. 

One other thing to note here in the output is the mention of [`DefaultAzureCredential`](https://docs.microsoft.com/en-us/java/api/com.azure.identity.defaultazurecredential?view=azure-java-stable) 
which is a very useful class that will try a number of different ways to authenticate with Azure. In this case it is using the Azure CLI login but in a CI build it could use a manged identity.

### Step 5. Fix the test

Firstly lets destroy the infrastructure we want to test:

```bash
> cd terraform/environments/dev
> terragrunt run-all apply
...
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
  "subnet1" = "/subscriptions/22222222-2222-2222-2222-222222222222/resourceGroups/rg-edo-dev-testapp/providers/Microsoft.Network/virtualNetworks/vnet-edo-dev-testapp/subnets/subnet1"
  "subnet2" = "/subscriptions/22222222-2222-2222-2222-222222222222/resourceGroups/rg-edo-dev-testapp/providers/Microsoft.Network/virtualNetworks/vnet-edo-dev-testapp/subnets/subnet2"
}
vnet_address_space = tolist([
  "10.0.0.0/16",
])
vnet_id = "/subscriptions/22222222-2222-2222-2222-222222222222/resourceGroups/rg-edo-dev-testapp/providers/Microsoft.Network/virtualNetworks/vnet-edo-dev-testapp"
vnet_name = "vnet-edo-dev-testapp"
```

Now we can run our test again and see it pass:

```bash
> ./gradlew clean build
...
AzureIntegrationTest > Resource group exists STANDARD_OUT
    2023-10-19 17:52:32.346  INFO 391518 --- [    Test worker] c.azure.identity.ChainedTokenCredential  : Azure Identity => Attempted credential EnvironmentCredential is unavailable.
    2023-10-19 17:52:33.557  INFO 391518 --- [onPool-worker-1] c.azure.identity.ChainedTokenCredential  : Azure Identity => Attempted credential ManagedIdentityCredential is unavailable.
    2023-10-19 17:52:33.582  INFO 391518 --- [onPool-worker-1] c.azure.identity.ChainedTokenCredential  : Azure Identity => Attempted credential SharedTokenCacheCredential is unavailable.
    2023-10-19 17:52:33.610  INFO 391518 --- [onPool-worker-1] c.azure.identity.ChainedTokenCredential  : Azure Identity => Attempted credential IntelliJCredential is unavailable.
    2023-10-19 17:52:34.121  INFO 391518 --- [onPool-worker-1] com.azure.identity.AzureCliCredential    : Azure Identity => getToken() result for scopes [https://management.core.windows.net//.default]: SUCCESS
    2023-10-19 17:52:34.122  INFO 391518 --- [onPool-worker-1] c.azure.identity.ChainedTokenCredential  : Azure Identity => Attempted credential AzureCliCredential returns a token
    2023-10-19 17:52:34.123  INFO 391518 --- [onPool-worker-1] c.a.c.implementation.AccessTokenCache    : {"az.sdk.message":"Acquired a new access token."}
AzureIntegrationTest

  Test Resource group exists PASSED (2.5s)

SUCCESS: Executed 1 tests in 7s


Deprecated Gradle features were used in this build, making it incompatible with Gradle 9.0.

You can use '--warning-mode all' to show the individual deprecation warnings and determine if they come from your own scripts or plugins.

For more on this, please refer to https://docs.gradle.org/8.4/userguide/command_line_interface.html#sec:command_line_warnings in the Gradle documentation.

BUILD SUCCESSFUL in 10s
8 actionable tasks: 8 executed
```

### Creating more configurable tests

The test above is great and gives us confidence our infrastructure really deployed and was correctly configured. However, it is not very useful
across many environments as it is hard coded to the dev environment. The terragrunt definitions allows us to deploy to many environments and so
it would be good if our tests did the same

### Creating more realistic / useful tests

Now that we have a basic test working we can start to add more tests to validate the infrastructure we have deployed. We can al