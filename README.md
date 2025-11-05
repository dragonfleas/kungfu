# kungfu

kungfu is a tool to patch and extend the internals of OpenTofu/Terraform modules without forking repositories, inspired by Kustomize's approach to declarative configuration patching.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [CLI Reference](#cli-reference)
- [How It Works](#how-it-works)
- [Patch Strategies](#patch-strategies)
- [Use Cases](#use-cases)
- [Examples](#examples)
- [Limitations](#limitations)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)
- [Roadmap](#roadmap)

## Installation

```bash
go build -o kungfu
```

## Quick Start

1. **Run terraform init** to download your modules:

```bash
terraform init
```

2. **Create an overlay file** in the `overlays/` directory:

```hcl
# overlays/production.kf.hcl
patch "aws_instance" "bastion" {
  source = "./modules/ec2-bastion-host-module"  # Must match module source

  instance_type = "t3.large"
  monitoring    = true

  tags = merge({
    Owner     = "platform-team"
    ManagedBy = "kungfu"
  })

  vpc_security_group_ids = append(["sg-restricted"])
}
```

3. **Build the patched modules**:

```bash
kungfu build . --overlay overlays/production.kf.hcl
```

4. **Run terraform plan** - your patched modules are now active:

```bash
terraform plan
```

No need to modify your `main.tf` or module declarations - kungfu automatically updates Terraform's internal module manifest.

## CLI Reference

### `kungfu build [root-module-path] [flags]`

Builds patched modules from overlay files and transparently activates them.

**Flags:**

- `--overlay <path>` - Specific `.kf.hcl` file or directory (default: `overlays/`)
- `-o, --output <path>` - Output directory (default: `.terraform/kungfu/modules`)

**Examples:**

```bash
# Build with all overlays in overlays/ directory
kungfu build .

# Build with a specific overlay file
kungfu build . --overlay overlays/production.kf.hcl

# Build from a different root module path
kungfu build ./infrastructure --overlay overlays/production.kf.hcl
```

**Workflow:**

1. Parses root module to find all module declarations
2. Finds and parses all `.kf.hcl` files in overlay directory
3. Matches patches to modules by source attribute
4. Generates patched modules to `.terraform/kungfu/modules/`
5. Updates `.terraform/modules/modules.json` to point to patched modules
6. Next `terraform plan` or `terraform apply` transparently uses patched modules

## How It Works

kungfu operates on **child modules** referenced in your root module. Each patch block has a `source` attribute that must match a module's source path:

```hcl
# In main.tf
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.0.0"
}

# In overlays/production.kf.hcl
patch "aws_vpc" "this" {
  source = "terraform-aws-modules/vpc/aws"  # Must match module source

  enable_dns_hostnames = true

  tags = merge({
    Owner = "platform-team"
  })
}
```

After building, kungfu modifies `.terraform/modules/modules.json` to redirect Terraform to the patched modules:

```json
{
  "Modules": [{
    "Key": "vpc",
    "Source": "terraform-aws-modules/vpc/aws",
    "Dir": ".terraform/kungfu/modules/vpc"  // Redirected to patched version
  }]
}
```

## Patch Strategies

kungfu supports three strategies for applying patches:

### 1. Replace (default)

Completely replaces the original value:

```hcl
patch "aws_instance" "example" {
  source = "./modules/ec2-instance"

  instance_type = "t3.large"  # Replaces original value
}
```

### 2. Merge

Deep merges maps/objects, preserving original keys:

```hcl
patch "aws_instance" "example" {
  source = "./modules/ec2-instance"

  tags = merge({
    Owner     = "platform-team"  # Merged with existing tags
    ManagedBy = "kungfu"
  })
}
```

Original tags are preserved, new tags are added, and conflicting keys use the patch value.

### 3. Append

Appends items to lists/arrays:

```hcl
patch "aws_instance" "example" {
  source = "./modules/ec2-instance"

  vpc_security_group_ids = append(["sg-restricted", "sg-monitoring"])
}
```

Original list items are preserved, patch items are appended.

### Combining Strategies

```hcl
patch "aws_instance" "app" {
  source = "./modules/app-server"

  instance_type          = "t3.large"           # replace (default)
  tags                   = merge({...})          # merge maps
  vpc_security_group_ids = append([...])        # append to lists
}
```

## Use Cases

> [!NOTE]
> The following examples are simplified for demonstration purposes. Real-world usage will depend on your specific module internals and requirements.

### Patching Third-Party Registry Modules

```hcl
# main.tf
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.0.0"

  name = "my-vpc"
  cidr = "10.0.0.0/16"
  # ... other standard config
}

# overlays/add-compliance-tags.kf.hcl
# Add organization-wide compliance tags to the VPC resource itself
# (not exposed as module variables)
patch "aws_vpc" "this" {
  source = "terraform-aws-modules/vpc/aws"

  tags = merge({
    CostCenter     = "infrastructure"
    DataClass      = "internal"
    Compliance     = "sox"
    BackupPolicy   = "daily"
    PatchingWindow = "sun-03:00"
  })
}

# overlays/enable-flow-logs-encryption.kf.hcl
# Enable encryption on VPC flow logs (internal resource not configurable via module)
patch "aws_flow_log" "this" {
  source = "terraform-aws-modules/vpc/aws"

  tags = merge({
    Owner      = "security-team"
    Encrypted  = "true"
  })
}
```

### Environment-Specific Configurations

```hcl
# overlays/staging.kf.hcl
patch "aws_instance" "app" {
  source        = "./modules/app-server"
  instance_type = "t3.medium"
}

# overlays/production.kf.hcl
patch "aws_instance" "app" {
  source        = "./modules/app-server"
  instance_type = "t3.xlarge"
  monitoring    = true
}
```

Build for different environments:

```bash
kungfu build . --overlay overlays/staging.kf.hcl
kungfu build . --overlay overlays/production.kf.hcl
```

### Security Hardening

```hcl
patch "aws_instance" "web" {
  source = "terraform-aws-modules/ec2-instance/aws"

  monitoring              = true
  disable_api_termination = true

  metadata_options = merge({
    http_tokens = "required"
  })
}
```

### Compliance and Governance

```hcl
patch "aws_s3_bucket" "data" {
  source = "terraform-aws-modules/s3-bucket/aws"

  tags = merge({
    Owner          = "data-team"
    Compliance     = "GDPR"
    Classification = "sensitive"
  })
}
```

## Examples

Complete working examples are available in the `examples/` directory:

### [Remote Module Patching](examples/remote-module/)

Demonstrates patching third-party modules from the Terraform Registry without forking:

- Patches `terraform-aws-modules/vpc/aws` and `terraform-aws-modules/ec2-instance/aws`
- Shows production and staging environment overlays
- Explains how to enforce organizational policies on community modules
- Includes real-world use cases for security, compliance, and tagging

**Quick start:**

```bash
cd examples/remote-module
terraform init
kungfu build . --overlay overlays/production.kf.hcl
terraform plan
```

### [Local Module Patching](examples/local-module/)

Demonstrates patching local child modules:

- Shows the transparent workflow with modules.json
- Multiple overlays for different environments
- Complete explanation of how kungfu integrates with Terraform

**Quick start:**

```bash
cd examples/local-module
terraform init
kungfu build . --overlay overlays/production.kf.hcl
terraform plan
```

## Limitations

- Only `resource` blocks can be patched currently (variables, outputs, data sources, locals coming soon)
- Only HCL **attributes** can be patched (e.g., `tags = {...}`), not HCL **blocks** (e.g., `root_block_device { ... }`)
- Nested blocks (like `ingress` blocks, `root_block_device` blocks, `ebs_block_device` blocks) cannot be patched yet
- Must run `terraform init` before `kungfu build` (modules must be downloaded first)
- The `source` attribute in patches must exactly match the module source in your root module
- Complex expressions may not preserve formatting exactly

## Best Practices

### 1. Version Control

Always commit your `overlays/` directory alongside your root module.

### 2. Documentation

Comment your patches to explain why modifications are needed:

```hcl
# Security requirement: All production instances must have termination protection
# Reference: SEC-001
patch "aws_instance" "app" {
  source                  = "./modules/app-server"
  disable_api_termination = true
}
```

### 3. Default to Merge Strategy

When patching maps/objects, prefer `merge()` over direct replacement. This preserves the module's original behavior and only adds or overrides specific values you care about. Direct replacement can accidentally remove important attributes the module author set.

```hcl
# Good: preserves module's original values, only adds/overrides what you specify
tags = merge({
  Owner = "platform-team"
})

metadata_options = merge({
  http_tokens = "required"  # Override this one field
})

# Bad: completely replaces all values, losing module defaults
tags = {
  Owner = "platform-team"  # Loses any tags the module set!
}

metadata_options = {
  http_tokens = "required"  # Loses other important metadata_options!
}
```

**Why this matters:** If the module author later adds new default tags or metadata options in an update, using `merge()` means you'll automatically inherit those improvements. With direct replacement, you'd be stuck with only what you explicitly defined.

### 4. Module Source Consistency

Ensure `source` attributes match exactly between `main.tf` and overlay files:

```hcl
# main.tf
module "vpc" {
  source = "terraform-aws-modules/vpc/aws"
}

# overlays/production.kf.hcl - must match exactly
patch "aws_vpc" "this" {
  source = "terraform-aws-modules/vpc/aws"
}
```

## Troubleshooting

### Error: "No module found for source X"

The `source` in your patch doesn't match any module. Verify:

1. Module source in `main.tf` matches exactly
2. You've run `terraform init` to download modules

```bash
terraform init
kungfu build . --overlay overlays/production.kf.hcl
```

### Warning: "modules.json not found. Run 'terraform init' first."

Run `terraform init` before `kungfu build`:

```bash
terraform init
kungfu build . --overlay overlays/production.kf.hcl
```

### Error: "resource X not found"

The resource type and name must exactly match a resource in the module. Check:

- Module's source code for resource names
- `terraform state list` if module is already applied
- Module documentation

### Patches Not Applied

Debug steps:

```bash
# 1. Verify modules downloaded
ls .terraform/modules/

# 2. Check kungfu output
kungfu build . --overlay overlays/production.kf.hcl

# 3. Verify modules.json updated
cat .terraform/modules/modules.json
```

### Changes Not Reflected

Rebuild after modifying overlay files:

```bash
kungfu build . --overlay overlays/production.kf.hcl
terraform plan
```

## Roadmap

_In no specific order:_

- [x] Module code generation
- [x] Patch specific resources
- [x] Merge strategies (replace, merge, append)
- [x] Root module context and child module patching
- [x] Multiple overlay file support
- [x] Remote module patching (registry, git)
- [ ] Patch variables, outputs, data sources, locals
- [ ] HCL block-level patching (for constructs like `root_block_device { ... }`, `ingress { ... }`, etc.)
- [ ] Dynamic block patching (for `dynamic` blocks)
- [ ] Conditional patches
- [ ] Patch validation and linting
- [ ] Diff output for patches
- [ ] Dry-run mode
- [ ] Package distribution (homebrew, apt, yum)

## Status

kungfu is experimental and under active development. Core functionality works but breaking changes should be expected.

## Contributing

Contributions are welcome! Submit issues or pull requests.

## License

GNU Affero General Public License v3.0 - see LICENSE file for details.
