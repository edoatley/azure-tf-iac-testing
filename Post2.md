# Testing Terraform code using Terratest

## Introduction

In my [previous article](https://www.linkedin.com/pulse/using-terragrunt-deploy-infrastructure-code-ed-oatley-poeqe) I explained how I 
had used Terragrunt to build up a Terraform project that is configurable and can easily target multiple different environments. My next
concern is how we are going to test that the code is going to work as expected to define the correct infrastructure.

Testing is always an interesting challenge and ensuring you write the correct tests for your infrastructure is no
different. I like to work in a test-driven way where I can, though I am not a zealot and like to get all the basics up and 
ready before getting too carried away with a [red, green, refactor cycle](https://www.codecademy.com/article/tdd-red-green-refactor)!

With our basic project set up, we can use terratest to help validate it. Terratest is a library (written in [Go](https://go.dev/))
that makes it easier to write automated tests for your infrastructure code by providing a number of helper functions.

I will extend this article in a future one to look at how we can use integration testing to validate our infrastructure further.

### Step 1 - Pre-requisites

I will begin with my project in the same state it finished in the previous article. We have a `terraform` directory that looks like this:

```bash
❯ cd terraform/
❯ tree .
.
├── environments
│   ├── dev
│   │   ├── dev-common.yaml
│   │   ├── resource_group
│   │   │   └── terragrunt.hcl
│   │   ├── terragrunt.hcl
│   │   ├── virtual_machine
│   │   │   └── terragrunt.hcl
│   │   └── virtual_network
│   │       └── terragrunt.hcl
│   ├── prod
│   │   ├── prod-common.yaml
│   │   ├── resource_group
│   │   │   └── terragrunt.hcl
│   │   ├── terragrunt.hcl
│   │   ├── virtual_machine
│   │   │   └── terragrunt.hcl
│   │   └── virtual_network
│   │       └── terragrunt.hcl
│   └── terragrunt.hcl
└── modules
    ├── resource_group
    │   ├── main.tf
    │   ├── outputs.tf
    │   └── variables.tf
    ├── vm
    │   ├── main.tf
    │   ├── outputs.tf
    │   └── variables.tf
    └── vnet
        ├── main.tf
        ├── outputs.tf
        └── variables.tf
```

So there is a `dev` and a `prod` environment and the code will deploy a resource group, a virtual network and a virtual machine.

### Step 2. Create a basic test

To use terratest we must install go following [these instructions](https://golang.org/doc/install). We can then set up our project 
with the help of the [official guidance](https://terratest.gruntwork.io/docs/getting-started/quick-start/#setting-up-your-project).

Now we create a new directory `test` and add a file `terraform_test.go` with the following contents:

```go
package test

// import the testing modules that we need
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

I have commented the code to explain what it is doing. The key points are:

- target the `dev` environment directory and use the `terragrunt` binary
- [`defer`](https://go.dev/tour/flowcontrol/12) destruction of the resources until the function returns
- use terragrunt to apply the resources (a `terragrunt run-all apply` under the covers)
- fetch the resource group name
- assert that the resource group name is as expected

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

Picking out the key parts of the output, we see the test:


1. Running the `terragrunt run-all apply` command:

```output
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

```output
TestTerraformBasicExample 2023-09-22T13:01:35+01:00 logger.go:66: Apply complete! Resources: 3 added, 0 changed, 0 destroyed.
TestTerraformBasicExample 2023-09-22T13:01:35+01:00 logger.go:66: 
TestTerraformBasicExample 2023-09-22T13:01:35+01:00 logger.go:66: Outputs:
TestTerraformBasicExample 2023-09-22T13:01:35+01:00 logger.go:66: 
TestTerraformBasicExample 2023-09-22T13:01:35+01:00 logger.go:66: resource_group_name = "rg-edo-dev-testapp"
```

3. Getting the output of the resource group module

```output
TestTerraformBasicExample 2023-09-22T13:02:14+01:00 retry.go:91: terragrunt [output -no-color -json resource_group_name --terragrunt-non-interactive]
TestTerraformBasicExample 2023-09-22T13:02:14+01:00 logger.go:66: Running command terragrunt with args [output -no-color -json resource_group_name --terragrunt-non-interactive]
TestTerraformBasicExample 2023-09-22T13:02:14+01:00 logger.go:66: "rg-edo-dev-testapp"
```

4. Successfully destroying the resources

```output
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

```output
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

This said if what you want to test deploying all the modules then you are likely better off using the `run-all` command 
as this is more realistic.

### Step 5. Making a more useful, clean test

The long running nature of an apply-all test does make for an interesting challenge where you either:

- run a long running test many times each with a single assertion
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

and this is now an isolated test that does not affect the real dev environment.

## Conclusions

In this article I have shown how we can use terratest to test our terraform code. Terratest does have the benefit of running the 
actual terraform code and testing the results but there are a number of limitations:

1. It is slow. This is because it is running the actual terraform code and so the resource providers really execute in Azure and
   this takes time.

2. We are a bit limited in the assertions we can make as we are seeing everything through the lens of terraform outputs. This is
   not a bad thing but it does mean we are not testing the actual resources in Azure.

In my next article I attempt to address this latter issue by using integration testing to test the actual resources in Azure.

Thanks for reading I hope this was useful.