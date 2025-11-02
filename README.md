# kungfu

kungfu is a tool to patch and extend the internals of OpenTofu modules without needing to modify the module source, inspired by Kustomize's approach to bringing extensibility to declarative configuration. This project has no affiliation with Kustomize.

## Overview

While Kustomize allows you to patch and create variants of Kubernetes manifests without modifying the original files, kungfu brings the same philosophy to OpenTofu modules. It takes a source reusable module written by you or an external developer, applies patches over the top of its internals in the form of overlays and variants, and generates a new module that can be utilized by your main OpenTofu code.

This approach allows teams to maintain reusable base modules while creating environment-specific or use-case-specific variants through patches and overlays. Rather than forking modules or using complex variable configurations, kungfu generates customized modules from your base modules with targeted modifications applied.

kungfu natively supports HCL and does not require any bespoke configuration language like JSON or YAML. Configuration is written in the same language used for OpenTofu, making it familiar and straightforward for practitioners already working with infrastructure as code.

## Using the application

TBD

## Roadmap

- [ ] Module code generation
- [ ] Ability to patch specific resources at plan time
- [ ] Creating variants
- [ ] Creating overlays
- [ ] Variant & overlay versioning
- [ ] Packaging
- [ ] Distribution

## Status

kungfu is currently experimental and under active development. Do not use this tool for production use cases yet. The project structure and core functionality are being established, and breaking changes should be expected.
