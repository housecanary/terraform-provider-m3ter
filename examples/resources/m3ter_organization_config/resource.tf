resource "m3ter_organization_config" "org_config" {
  timezone                     = "UTC"
  year_epoch                   = "2022-01-01"
  month_epoch                  = "2022-01-01"
  week_epoch                   = "2022-01-01"
  day_epoch                    = "2022-01-01"
  currency                     = "USD"
  days_before_bill_due         = 10
  auto_generate_statement_mode = "JSON_AND_CSV"
}
