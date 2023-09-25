package test

import (
	"encoding/json"
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

// helper function to fetch a map output
func getOutputMap(t *testing.T, terraformOptions *terraform.Options, dir string, output string) map[string]string {
  outputValue := getOutput(t, terraformOptions, dir, output)
  var outputMap map[string]string
  err := json.Unmarshal([]byte(outputValue), &outputMap)
  if err != nil {
    t.Fatalf("Failed to unmarshal output map: %v", err)
  }
  return outputMap
}


// helper function to simplify fetching the outputs when using terragrunt run-all
func getOutput(t *testing.T, terraformOptions *terraform.Options, dir string, output string) string {
  terraformOptions.TerraformDir = terraformParentDir + dir
  outputValue := terraform.Output(t, terraformOptions, output)
  terraformOptions.TerraformDir = terraformParentDir
  return outputValue
}