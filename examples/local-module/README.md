# Example: Patching Child Modules

This example demonstrates how to patch child modules using kungfu overlays.

## Structure

```
examples/main/
├── main.tf                              # Root module with module declarations
├── modules/
│   └── ec2-bastion-host-module/         # Child module to be patched
│       ├── main.tf
│       └── variables.tf
└── overlays/
    ├── production.kf.hcl                # Production patches
    └── staging.kf.hcl                   # Staging patches
```

## Module Declaration

The root module (`main.tf`) declares a child module:

```hcl
module "ec2_bastion_host" {
  source = "./modules/ec2-bastion-host-module"

  ami_id = "ami-0c55b159cbfafe1f0"
  vpc_id = "vpc-12345678"
}
```

## Overlay Files

Overlay files in the `overlays/` directory define patches for the child module. Each patch must specify the `source` attribute matching the module's source path.

### Production Overlay (`overlays/production.kf.hcl`)

```hcl
patch "aws_instance" "bastion" {
  source = "./modules/ec2-bastion-host-module"

  instance_type           = "t3.large"
  monitoring              = true
  disable_api_termination = true

  tags = merge({
    Owner     = "platform-team"
    ManagedBy = "kungfu"
  })

  vpc_security_group_ids = append(["sg-restricted", "sg-monitoring"])
}
```

### Staging Overlay (`overlays/staging.kf.hcl`)

```hcl
patch "aws_instance" "bastion" {
  source = "./modules/ec2-bastion-host-module"

  instance_type = "t3.medium"
  # ... staging-specific patches
}
```

## Workflow

### 1. Initialize Terraform

First, run `terraform init` to download and initialize modules:

```bash
terraform init
```

### 2. Build with Production Overlay

```bash
kungfu build . --overlay overlays/production.kf.hcl
```

This will:

- Parse the root module and find all module declarations
- Match patches to modules by source attribute
- Generate patched modules to `.terraform/kungfu/modules/ec2_bastion_host/`
- Automatically update `.terraform/modules/modules.json` to point to the patched module

### 3. Use Patched Modules Transparently

Run terraform plan or apply - no configuration changes needed:

```bash
terraform plan
```

Terraform will automatically use the patched module because kungfu updated the module manifest.

### Build with Staging Overlay

```bash
kungfu build . --overlay overlays/staging.kf.hcl
terraform plan
```

### Build with All Overlays in Directory

```bash
kungfu build .
# Uses overlays/ directory by default
terraform plan
```

## How It Works

kungfu operates transparently by modifying Terraform's internal module resolution:

1. **Before kungfu build**: `.terraform/modules/modules.json` points to original module
   ```json
   {
     "Modules": [{
       "Key": "ec2_bastion_host",
       "Source": "./modules/ec2-bastion-host-module",
       "Dir": "modules/ec2-bastion-host-module"
     }]
   }
   ```

2. **After kungfu build**: modules.json is updated to point to patched module
   ```json
   {
     "Modules": [{
       "Key": "ec2_bastion_host",
       "Source": "./modules/ec2-bastion-host-module",
       "Dir": ".terraform/kungfu/modules/ec2_bastion_host"
     }]
   }
   ```

3. **Your configuration stays the same**: No need to change `main.tf` module declarations
   ```hcl
   module "ec2_bastion_host" {
     source = "./modules/ec2-bastion-host-module"  # Unchanged
     # ...
   }
   ```

## Real-World Use Case

This pattern is especially useful for patching third-party modules from registries or git:

```hcl
# Original module from registry
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.0.0"
  # ...
}

# Patch file (overlays/production.kf.hcl)
patch "aws_vpc" "this" {
  source = "terraform-aws-modules/vpc/aws"  # Matches module source

  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = merge({
    Owner = "platform-team"
  })

  # Patch module internals without forking!
}
```

After running `kungfu build . --overlay overlays/production.kf.hcl`, the third-party VPC module will be patched with your customizations, and Terraform will transparently use the patched version.
