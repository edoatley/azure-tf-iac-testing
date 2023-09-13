# Tell this child configuration to include the root terragrunt.hcl file
include "root" {
  path = find_in_parent_folders()
}

