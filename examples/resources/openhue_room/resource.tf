
resource "openhue_room" "bedroom" {
    archetype = "bedroom"
    light_ids = [
        "aaaa-bbbb-cccc-ddd"
    ]
}
