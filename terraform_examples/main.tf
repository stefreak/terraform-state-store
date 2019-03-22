variable "sleep_seconds" {
  type    = "string"
  default = "10"
}

resource "null_resource" "test1" {
  provisioner "local-exec" {
    command = "sleep ${var.sleep_seconds}"
  }
}

terraform {
  backend "http" {
    address        = "http://localhost:8080/v1/state/foo"
    lock_address   = "http://localhost:8080/v1/state/foo"
    unlock_address = "http://localhost:8080/v1/state/foo"
    username       = "steffen"
    password       = "verysecret"
  }
}
