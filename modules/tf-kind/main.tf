resource "kind_cluster" "this" {
  name           = "kind-cluster"
  wait_for_ready = true
  node_image = "kindest/node:v1.28.0"
}