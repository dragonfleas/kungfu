# Production patches for terraform-aws-modules/vpc/aws
# This demonstrates patching a popular third-party module from the Terraform Registry

patch "aws_vpc" "this" {
  source = "terraform-aws-modules/vpc/aws"

  # Enable DNS features for production
  enable_dns_hostnames = true
  enable_dns_support   = true

  # Add production-specific tags
  tags = merge({
    Owner       = "platform-team"
    ManagedBy   = "kungfu"
    Environment = "production"
    CostCenter  = "engineering"
  })
}

# Patch VPC flow logs configuration
patch "aws_flow_log" "this" {
  source = "terraform-aws-modules/vpc/aws"

  tags = merge({
    Owner     = "security-team"
    ManagedBy = "kungfu"
  })
}

# Production patches for terraform-aws-modules/ec2-instance/aws
patch "aws_instance" "this" {
  source = "terraform-aws-modules/ec2-instance/aws"

  # Upgrade instance type for production workloads
  instance_type = "t3.large"

  # Enable detailed monitoring for production
  monitoring = true

  # Enable termination protection
  disable_api_termination = true

  # Add additional security groups
  vpc_security_group_ids = append(["sg-prod-monitoring", "sg-prod-logging"])

  # Production-specific root volume configuration
  root_block_device = merge({
    encrypted   = true
    volume_size = 100
    volume_type = "gp3"
    iops        = 3000
    throughput  = 125
  })

  # Add production tags
  tags = merge({
    Owner       = "app-team"
    ManagedBy   = "kungfu"
    Environment = "production"
    Backup      = "daily"
  })

  # Merge volume tags
  volume_tags = merge({
    Owner       = "app-team"
    ManagedBy   = "kungfu"
    Encrypted   = "true"
  })
}
