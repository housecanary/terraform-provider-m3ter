resource "m3ter_webhook_destination" "test" {
  name        = "terraform test"
  description = "terraform test description"
  code        = "terraform_test"
  url         = "https://webhooks.example.com"
  credentials = {
    api_key = "test-api-key"
    secret  = "test-secret"
  }
}
