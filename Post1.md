# Building and testing Terraform Infrastructure as Code

## Introduction

Infrastructure as Code (IaC) is the process of defining your infrastructure resources in source code that can be versioned 
and managed like any other software. This allows you to automate the provisioning of your infrastructure in a repeatable and 
consistent manner and allows changes to be tracked and audited. When adopting the cloud like Azure and AWS this becomes even more
important as the number of resources you are managing can grow exponentially and the ability to manage them manually becomes
impossible.

In the Azure world the primary way to manage resource is Azure Resource Manager (ARM) templates but these are not human readable 
(at least to us mortals!) nor as flexible as Terraform which is widely used and declarative. Therefore this document will focus on 
Terraform where I have most experience but explore some new tooling that will hopefully help improve my workflows. There are two key 
challenges I am interested in addressing:

- How to structure the Terraform code to make it easier to manage multiple configurations
- How to test terraform code so that we are confident it works and are very happy to destroy our infrastructure safe in the knowledge we can recreate it

I am going to tackle these questions in 3 parts:

- **Part 1** - setting up and utilising terragrunt to manage the terraform code and configuration
- **Part 2** - using terratest to test the terraform code
- **Part 3** - using spock to test the deployed

In a nutshell the key challenge I am looking at here is how to configure test Terraform code to ensure it is working as expected across
all the environments it executes. This is not a trivial task as a handful of terraform resources will lead to the execution of dozens of 
API requests to ARM to both check the current state. I hope you enjoy my exploration of this topic and find it useful.

## Part 1 - Terragrunt

First we will set up a terragrunt project to manage/wrap the terraform code as it provides a number of useful features
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