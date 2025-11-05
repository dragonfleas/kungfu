# Example: Patching Remote Modules from Terraform Registry

This example demonstrates how to patch third-party modules from the Terraform Registry using kungfu, without forking or modifying the original module source.

## Why Patch Remote Modules?

When using community modules from the Terraform Registry (like `terraform-aws-modules`), you often need to:
- Add organization-specific tags or metadata
- Enforce security policies (encryption, monitoring, etc.)
- Override default configurations for different environments
- Customize module internals without maintaining a fork

kungfu allows you to apply these patches transparently, while still benefiting from upstream module updates.

## Structure

```
examples/remote-module/
├── main.tf                              # Root module using registry modules
└── overlays/
    ├── production.kf.hcl                # Production environment patches
    └── staging.kf.hcl                   # Staging environment patches
```

## Module Declarations

The root module (`main.tf`) uses popular modules from the Terraform Registry:

```hcl
# VPC module from terraform-aws-modules
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.0.0"

  name = "my-vpc"
  cidr = "10.0.0.0/16"
  # ... basic configuration
}

# EC2 instance module from terraform-aws-modules
module "web_server" {
  source  = "terraform-aws-modules/ec2-instance/aws"
  version = "5.0.0"

  name          = "web-server"
  instance_type = "t3.micro"
  # ... basic configuration
}
```

## Overlay Files

### Production Overlay (`overlays/production.kf.hcl`)

The production overlay enforces production-grade configurations:

```hcl
# Patch the VPC module's internal aws_vpc resource
patch "aws_vpc" "this" {
  source = "terraform-aws-modules/vpc/aws"

  # Enable DNS features
  enable_dns_hostnames = true
  enable_dns_support   = true

  # Add production tags
  tags = merge({
    Owner       = "platform-team"
    ManagedBy   = "kungfu"
    Environment = "production"
    CostCenter  = "engineering"
  })
}

# Patch the EC2 module's internal aws_instance resource
patch "aws_instance" "this" {
  source = "terraform-aws-modules/ec2-instance/aws"

  # Upgrade for production
  instance_type           = "t3.large"
  monitoring              = true
  disable_api_termination = true

  # Add security groups
  vpc_security_group_ids = append(["sg-prod-monitoring", "sg-prod-logging"])

  # Enable encryption
  root_block_device = merge({
    encrypted   = true
    volume_size = 100
    volume_type = "gp3"
  })

  # Production tags
  tags = merge({
    Owner       = "app-team"
    ManagedBy   = "kungfu"
    Environment = "production"
    Backup      = "daily"
  })
}
```

## Workflow

### 1. Initialize Terraform

Download modules from the registry:

```bash
cd examples/remote-module
terraform init
```

This downloads the modules to `.terraform/modules/`:
```
.terraform/modules/
├── modules.json
├── vpc/                          # Downloaded from registry
└── web_server/                   # Downloaded from registry
```

### 2. Build with Production Patches

Apply production patches to the downloaded modules:

```bash
kungfu build . --overlay overlays/production.kf.hcl
```

**What happens:**
1. kungfu reads your module declarations from `main.tf`
2. Matches patches to modules by `source` attribute
3. Parses the downloaded module files in `.terraform/modules/vpc/` and `.terraform/modules/web_server/`
4. Applies patches to the module's internal resources
5. Writes patched versions to `.terraform/kungfu/modules/vpc/` and `.terraform/kungfu/modules/web_server/`
6. Updates `.terraform/modules/modules.json` to point Terraform to the patched modules

### 3. Verify the Patches

Run terraform plan to see the patched configuration:

```bash
terraform plan
```

You'll see:
- VPC with DNS features enabled
- Instance type upgraded to `t3.large`
- Monitoring enabled
- Encryption enabled on root volumes
- All organization tags applied
- Additional security groups attached

**All without modifying your module declarations or forking the upstream modules!**

### 4. Switch to Staging Configuration

```bash
# Rebuild with staging patches
kungfu build . --overlay overlays/staging.kf.hcl

# Plan with staging configuration
terraform plan
```

Now you'll see staging-appropriate configurations (t3.medium instances, basic monitoring, etc.).

## How the Source Matching Works

kungfu matches patches to modules using the `source` attribute:

**Module Declaration:**
```hcl
module "vpc" {
  source = "terraform-aws-modules/vpc/aws"
  # ...
}
```

**Patch File:**
```hcl
patch "aws_vpc" "this" {
  source = "terraform-aws-modules/vpc/aws"  # Must match exactly
  # ...
}
```

The `source` in the patch file tells kungfu:
- **Which module** to patch (matches the module's source)
- **Which resource** inside that module to patch (`aws_vpc.this`)

## Understanding Module Resolution

### Before kungfu build

`.terraform/modules/modules.json`:
```json
{
  "Modules": [
    {
      "Key": "vpc",
      "Source": "terraform-aws-modules/vpc/aws",
      "Dir": ".terraform/modules/vpc"
    },
    {
      "Key": "web_server",
      "Source": "terraform-aws-modules/ec2-instance/aws",
      "Dir": ".terraform/modules/web_server"
    }
  ]
}
```

### After kungfu build

`.terraform/modules/modules.json`:
```json
{
  "Modules": [
    {
      "Key": "vpc",
      "Source": "terraform-aws-modules/vpc/aws",
      "Dir": ".terraform/kungfu/modules/vpc"
    },
    {
      "Key": "web_server",
      "Source": "terraform-aws-modules/ec2-instance/aws",
      "Dir": ".terraform/kungfu/modules/web_server"
    }
  ]
}
```

**Your `main.tf` stays unchanged** - Terraform automatically uses the patched modules because kungfu updated the module manifest.

## Real-World Use Cases

### 1. Organization-Wide Tagging Policy

```hcl
# Every resource in every module gets compliance tags
patch "aws_instance" "this" {
  source = "terraform-aws-modules/ec2-instance/aws"

  tags = merge({
    Owner      = "platform-team"
    CostCenter = "engineering"
    Compliance = "sox"
  })
}
```

### 2. Security Hardening

```hcl
# Enforce encryption on all EBS volumes
patch "aws_instance" "this" {
  source = "terraform-aws-modules/ec2-instance/aws"

  root_block_device = merge({
    encrypted = true
  })

  ebs_block_device = merge({
    encrypted = true
  })
}
```

### 3. Environment-Specific Scaling

```hcl
# Production: larger instances + monitoring
# Staging: smaller instances, no monitoring
# Dev: minimal resources
```

### 4. Network Security

```hcl
# Add mandatory security groups to all instances
patch "aws_instance" "this" {
  source = "terraform-aws-modules/ec2-instance/aws"

  vpc_security_group_ids = append([
    "sg-monitoring",
    "sg-logging",
    "sg-compliance"
  ])
}
```

## Benefits

✅ **No Forking Required**: Use upstream modules as-is, apply patches on top
✅ **Stay Up-to-Date**: Pull module updates without merge conflicts
✅ **Environment-Specific**: Different patches for prod/staging/dev
✅ **Transparent**: No changes to your module declarations
✅ **Reversible**: Remove kungfu patches anytime by running `terraform init -upgrade`
✅ **Policy Enforcement**: Centralize organization policies in overlay files

## Tips

1. **Run `terraform init` first**: kungfu requires modules to be downloaded
2. **Match sources exactly**: The `source` in patches must match module declarations exactly
3. **Know the module internals**: You need to know resource names inside the module (check the module source code)
4. **Use version constraints**: Pin module versions in `main.tf` for consistency
5. **Test patches**: Run `terraform plan` to verify patches before applying

## Advanced: Git-Based Modules

kungfu also works with git-based modules:

```hcl
# Module from git
module "app" {
  source = "git::https://github.com/example/terraform-app.git?ref=v1.0.0"
}

# Patch file
patch "aws_instance" "app" {
  source = "git::https://github.com/example/terraform-app.git?ref=v1.0.0"
  # patches...
}
```

## Next Steps

1. Try the example: `cd examples/remote-module && terraform init && kungfu build .`
2. Inspect the patched modules: `ls -la .terraform/kungfu/modules/`
3. View the diff: `terraform plan`
4. Experiment with your own patches
5. Use in production to enforce organizational policies!
