package test

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

// constant to define terraform directory we want to test
const (
  terraformBinary        = "terragrunt"
  terraformParentDir     = "../terraform/environments/dev"
  resourceGroupModule    = "/resource_group"
  virtualNetworkModule   = "/virtual_network"
  virtualNetworkAddress  = "10.0.0.0/16"
  subnet1Address         = "10.0.1.0/24"
  subnet2Address         = "10.0.2.0/24"
)

func TestTerraformRunAll(t *testing.T) {
	t.Parallel()

  

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir:    terraformParentDir,
		TerraformBinary: terraformBinary,
		Vars: map[string]interface{}{
			"suffix": []string{"terratest", "edo"},
		},
	})

	// At the end of the test, run `terragrunt run-all destroy` to clean up any resources that were created.
	defer terraform.TgDestroyAll(t, terraformOptions)

	// Run `terragrunt run-all apply`. Fail the test if there are any errors.
	terraform.TgApplyAll(t, terraformOptions)

  t.Run("Resource Group", func(t *testing.T) {
    t.Helper()
    validateResourceGroup(t, terraformOptions)
  })

  t.Run("Virtual Network", func(t *testing.T) {
      t.Helper()
      validateVirtualNetwork(t, terraformOptions)
  })
}

// helper function to validate the resource group
func validateResourceGroup(t *testing.T, terraformOptions *terraform.Options) {
	rgName := getSimpleOutput(t, terraformOptions, resourceGroupModule, "resource_group_name")
	assert.Equal(t, "rg-terratest-edo-testapp", rgName)
}

// helper function to validate the vnet name and CIDR ranges
func validateVirtualNetwork(t *testing.T, terraformOptions *terraform.Options) {
  vnetName := getSimpleOutput(t, terraformOptions, "/virtual_network", "vnet_name")
  assert.Equal(t, "vnet-terratest-edo-testapp", vnetName)

  var vnetAddressSpaces []string
  getOutput(t, terraformOptions, "/virtual_network", "vnet_address_space", &vnetAddressSpaces)
  assert.Equal(t, "10.0.0.0/16", vnetAddressSpaces[0])

  var subnetAddressSpaces map[string][]string
  getOutput(t, terraformOptions, "/virtual_network", "subnet_address_spaces", &subnetAddressSpaces)

  assert.Equal(t, "10.0.1.0/24", subnetAddressSpaces["subnet1"][0])
  assert.Equal(t, "10.0.2.0/24", subnetAddressSpaces["subnet2"][0])
}

// helper function to fetch simple outputs when using terragrunt run-all
func getSimpleOutput(t *testing.T, terraformOptions *terraform.Options, dir string, outputRequested string) string {
	terraformOptions.TerraformDir = terraformParentDir + dir
	outputValue, err := terraform.OutputE(t, terraformOptions, outputRequested)
  if err != nil {
    t.Fatalf("Failed to fetch output %s: %v", outputRequested, err)
  }
	terraformOptions.TerraformDir = terraformParentDir
	return outputValue
}

// helper function to fetch more complicated outputs when using terragrunt run-all
func getOutput(t *testing.T, terraformOptions *terraform.Options, dir string, outputRequested string, output interface{}) {
  terraformOptions.TerraformDir = terraformParentDir + dir
	err := terraform.OutputStructE(t, terraformOptions, outputRequested, output)
  if err != nil {
    t.Fatalf("Failed to fetch output %s: %v", output, err)
  }
}