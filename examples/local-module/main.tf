terraform {
  backend "local" {}

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0.0"
    }
  }
}

module "ec2_bastion_host" {
  source = "./modules/ec2-bastion-host-module"

  ami_id = "ami-0c55b159cbfafe1f0" # Example AMI ID, replace with a valid one for your region
  vpc_id = "vpc-12345678"          # Example VPC ID, replace with a valid one for your setup
}