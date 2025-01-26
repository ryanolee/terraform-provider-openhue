terraform {
  required_providers {
    openhue = {
      source  = "ryanolee/openhue"
      version = "0.0.1"
    }
  }
}

provider "openhue" {
  bridge_ip = "discover"
  cache     = true
}

locals {
  blue   = provider::openhue::hextod65("#0000ff")
  red    = provider::openhue::hextod65("#ff0000")
  green  = provider::openhue::hextod65("#00ff00")
  indgo  = provider::openhue::hextod65("#4b0082")
  puce   = provider::openhue::hextod65("#cc8899")
  yellow = provider::openhue::hextod65("#ffff00")
  your_color = provider::openhue::hextod65("#ffffff")

}

resource "openhue_light" "light_1" {
  name       = "lamp_1"
  on         = true
  brightness = 100
  color      = local.your_color
}

resource "openhue_light" "light_2" {
  name       = "lamp_2"
  on         = true
  brightness = 100
  color      = local.your_color
}

data "openhue_light" "light_2" {
  name = "lamp_2"
}
