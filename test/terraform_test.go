package test

import (
	"encoding/json"
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
  resourceGroupName      = "rg-edo-dev-testapp"
  virtualNetworkName     = "vnet-edo-dev-testapp"
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
			"suffix": []string{"edo", "dev"},
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
	rgName := getOutput(t, terraformOptions, resourceGroupModule, "resource_group_name")
	assert.Equal(t, resourceGroupName, rgName)
}

// helper function to validate the vnet name and CIDR ranges
func validateVirtualNetwork(t *testing.T, terraformOptions *terraform.Options) {
  vnetName := getOutput(t, terraformOptions, "/virtual_network", "vnet_name")
  assert.Equal(t, "vnet-edo-dev-testapp", vnetName)

  vnetAddressSpaces := getOutputList(t, terraformOptions, "/virtual_network", "vnet_address_space")
  assert.Equal(t, "10.0.0.0/16", vnetAddressSpaces[0])

  subnetAddressSpaces := getOutputMap(t, terraformOptions, "/virtual_network", "subnet_address_spaces")
  assert.Equal(t, "10.0.1.0/24", subnetAddressSpaces["subnet1"][0])
  assert.Equal(t, "10.0.2.0/24", subnetAddressSpaces["subnet2"][0])
}

// helper function to fetch a list output
func getOutputList(t *testing.T, terraformOptions *terraform.Options, dir string, output string) []string {
  outputValue := getOutput(t, terraformOptions, dir, output)
  var outputList []string
  err := json.Unmarshal([]byte(outputValue), &outputList)
  if err != nil {
    t.Fatalf("Failed to unmarshal output list: %v", err)
  }
  return outputList
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
  t.Helper()
	terraformOptions.TerraformDir = terraformParentDir + dir
	outputValue := terraform.Output(t, terraformOptions, output)
	terraformOptions.TerraformDir = terraformParentDir
	return outputValue
}
