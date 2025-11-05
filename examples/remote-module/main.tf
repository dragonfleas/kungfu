terraform {
  backend "local" {}

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0.0"
    }
  }
}

provider "aws" {
  region = "us-west-2"
}

# Example using popular terraform-aws-modules/vpc module from registry
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "6.5.0"

  name = "my-vpc"
  cidr = "10.0.0.0/16"

  azs             = ["us-west-2a", "us-west-2b", "us-west-2c"]
  private_subnets = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
  public_subnets  = ["10.0.101.0/24", "10.0.102.0/24", "10.0.103.0/24"]

  enable_nat_gateway = true
  enable_vpn_gateway = false

  tags = {
    Terraform   = "true"
    Environment = "dev"
  }
}

# Example using terraform-aws-modules/ec2-instance module from registry
module "web_server" {
  source  = "terraform-aws-modules/ec2-instance/aws"
  version = "6.1.4"

  name = "web-server"

  instance_type          = "t3.micro"
  monitoring             = false
  vpc_security_group_ids = []
  subnet_id              = ""

  tags = {
    Terraform   = "true"
    Environment = "dev"
  }
}
