provider_installation {
  dev_overrides {
      "registry.terraform.io/ryanolee/openhue" = "$PWD/dist/"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
