terraform {
  required_providers {
    m3ter = {
      source = "registry.terraform.io/housecanary/m3ter"
    }
  }
}

provider "m3ter" {}
