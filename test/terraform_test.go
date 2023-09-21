package test

import (
	"testing"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

// define terraform directory we want to test
var terraformParentDir string = "../terraform/environments/dev"

// An example of how to test the simple Terraform module in examples/terraform-basic-example using Terratest.
func TestTerraformBasicExample(t *testing.T) {
	t.Parallel()

	// Construct the terraform options defining the path to the Terraform code and
	// specifying the terragrunt binary.
	
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		// The path to where our Terraform code is located
		TerraformDir: terraformParentDir,
		TerraformBinary: "terragrunt",
	})
	
	// At the end of the test, run `terraform destroy` to clean up any resources that were created.
	defer terraform.TgDestroyAll(t, terraformOptions)

	// Run `terraform init` and `terraform apply`. Fail the test if there are any errors.
	terraform.TgApplyAll(t, terraformOptions)

	// Need to Fetch all outputs to get the resource group name
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
