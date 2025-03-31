>[!IMPORTANT]
> This was originally created as an demonstration of how to write a terraform provider for a [talk](https://slides.com/rizza-1/deck-b121bc). Due to this API support is extremely limited and the provider is not intended for production use please feel free to use it as a reference however for writing your own provider.

# Open Hue Terraform Provider
This repository contains a provider for controlling phillips hue lights [Using open hue](https://www.openhue.io/)

## Development
>[!WARNING]
> This will override your `~/.terraform.rc` file. It will take a backup first but make sure to restore the backup after once you are done making changes.
Run `make setup-local-dev` to configure terraform to use this provider under your local dev environment.

## Usage
regenerating docs
```bash
make docs
```