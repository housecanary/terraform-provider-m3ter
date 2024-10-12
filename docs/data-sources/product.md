---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "m3ter_product Data Source - m3ter"
subcategory: ""
description: |-
  Product data source
---

# m3ter_product (Data Source)

Product data source



<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `code` (String) A unique short code to identify the Product. It should not contain control characters or spaces.
- `custom_fields` (Dynamic) User defined fields enabling you to attach custom data. The value for a custom field can be either a string or a number.
- `id` (String) Product identifier
- `name` (String) Descriptive name for the Product providing context and information.

### Read-Only

- `version` (Number) Product version