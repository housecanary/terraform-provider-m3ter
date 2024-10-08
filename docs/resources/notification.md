---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "m3ter_notification Resource - m3ter"
subcategory: ""
description: |-
  Notification resource
---

# m3ter_notification (Resource)

Notification resource

## Example Usage

```terraform
resource "m3ter_notification" "test" {
  name              = "terraform test"
  description       = "terraform test description"
  code              = "terraform_test"
  always_fire_event = true
  active            = false
  event_name        = "billing.billjob.updated"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `code` (String) The short code for the Notification.
- `description` (String) Description of the notification
- `event_name` (String) The name of the Event that triggers the Notification.
- `name` (String) Name of the notification

### Optional

- `active` (Boolean) Boolean flag that sets the Notification as active or inactive. Only active Notifications are sent when triggered by the Event they are based on.
- `always_fire_event` (Boolean) A Boolean flag indicating whether the Notification is always triggered, regardless of other conditions and omitting reference to any calculation. This means the Notification will be triggered simply by the Event it is based on occurring and with no further conditions having to be met.
- `calculation` (String) A logical expression that that is evaluated to a Boolean. If it evaluates as True, a Notification for the Event is created and sent to the configured destination. Calculations can reference numeric, string, and boolean Event fields.

### Read-Only

- `id` (String) Notification identifier
- `version` (Number) Notification version
