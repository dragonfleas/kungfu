# Staging patches for terraform-aws-modules/vpc/aws

patch "aws_vpc" "this" {
  source = "terraform-aws-modules/vpc/aws"

  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = merge({
    Owner       = "platform-team"
    ManagedBy   = "kungfu"
    Environment = "staging"
  })
}

# Staging patches for terraform-aws-modules/ec2-instance/aws
patch "aws_instance" "this" {
  source = "terraform-aws-modules/ec2-instance/aws"

  # Medium instance for staging
  instance_type = "t3.medium"

  # Basic monitoring for staging
  monitoring = false

  # Staging-specific tags
  tags = merge({
    Owner       = "app-team"
    ManagedBy   = "kungfu"
    Environment = "staging"
  })
}
