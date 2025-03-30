resource "openhue_light" "light_1" {
  # The light name here is VERY important. 
  # this is how the light is picked up normally
  name       = "lamp_1"
  on         = true
  brightness = 100
  color      = provider::openhue::hextod65("#0000ff") # Blue
}
