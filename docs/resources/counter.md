---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "m3ter_counter Resource - m3ter"
subcategory: ""
description: |-
  Counter resource
---

# m3ter_counter (Resource)

Counter resource



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Descriptive name for the Counter.
- `unit` (String) User defined label for units shown on Bill line items, and indicating to your customers what they are being charged for.

### Optional

- `code` (String) Code of the Counter - unique short code used to identify the Counter.
- `product_id` (String) UUID of the product the Counter belongs to. (Optional) - if left blank, the Counter is global.

### Read-Only

- `id` (String) Counter identifier
- `version` (Number) Counter version
