resource "m3ter_scheduled_event_configuration" "test" {
  name   = "scheduled.bill.terraformtest"
  entity = "Bill"
  field  = "endDate"
  offset = 1
}
