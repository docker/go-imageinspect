# Defines the output folder
variable "DESTDIR" {
  default = ""
}
function "bindir" {
  params = [defaultdir]
  result = DESTDIR != "" ? DESTDIR : "./bin/${defaultdir}"
}

group "default" {
  targets = ["test"]
}

group "validate" {
  targets = ["lint", "vendor-validate", "license-validate"]
}

target "lint" {
  target = "lint"
  output = ["type=cacheonly"]
}

target "vendor-validate" {
  target = "vendor-validate"
  output = ["type=cacheonly"]
}

target "vendor-update" {
  target = "vendor-update"
  output = ["."]
}

target "test" {
  target = "test-coverage"
  output = [bindir("coverage")]
}

target "license-validate" {
  target = "license-validate"
  output = ["type=cacheonly"]
}

target "license-update" {
  target = "license-update"
  output = ["."]
}
