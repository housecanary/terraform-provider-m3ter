resource "m3ter_integration_configuration" "test" {
  name           = "terraform test"
  entity_type    = "Notification"
  entity_id      = m3ter_notification.test.id
  destination    = "Webhook"
  destination_id = m3ter_webhook_destination.test.id
  config_data    = jsonencode({})
}
