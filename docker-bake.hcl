variable "PICFLOW_IMAGE" {
  default = "picflow:latest"
}

variable "GOPROXY" {
  default = "https://proxy.golang.org,direct"
}

group "default" {
  targets = ["picflow"]
}

target "picflow" {
  context    = "."
  dockerfile = "Dockerfile"
  tags       = [PICFLOW_IMAGE]
  platforms  = ["linux/amd64", "linux/arm64"]

  args = {
    GOPROXY = GOPROXY
  }
}
