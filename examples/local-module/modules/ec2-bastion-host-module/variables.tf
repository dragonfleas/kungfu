variable "ami_id" {
  description = "AMI ID for the bastion host"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID where the bastion host will be deployed"
  type        = string
}
