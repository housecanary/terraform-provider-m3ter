resource "m3ter_notification" "test" {
  name              = "terraform test"
  description       = "terraform test description"
  code              = "terraform_test"
  always_fire_event = true
  active            = false
  event_name        = "billing.billjob.updated"
}
