patch "aws_instance" "bastion" {
  source = "./modules/ec2-bastion-host-module"

  instance_type           = "t3.large"
  monitoring              = true
  disable_api_termination = true

  tags = merge({
    Owner       = "platform-team"
    ManagedBy   = "kungfu"
  })

  vpc_security_group_ids = append(["sg-restricted", "sg-monitoring"])
}

patch "aws_security_group" "bastion_sg" {
  description = replace("Security group for bastion host - restricted access")

  tags = merge({
    Owner     = "security-team"
    ManagedBy = "kungfu"
  })
}