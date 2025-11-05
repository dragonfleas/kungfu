resource "aws_instance" "bastion" {
  ami           = var.ami_id
  instance_type = "t3.micro"

  tags = {
    Name        = "bastion-host"
    Environment = "production"
  }

  monitoring              = false
  disable_api_termination = false
  vpc_security_group_ids  = ["sg-default"]
}

resource "aws_security_group" "bastion_sg" {
  name        = "bastion-sg"
  description = "Security group for bastion host"

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "bastion-security-group"
  }
}
