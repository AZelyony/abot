provider "flux" {
  kubernetes = {
    host                   = var.config_host
    token                  = var.config_token
    #cluster_ca_certificate = module.tf-google-gke-cluster.output.config_ca
    cluster_ca_certificate = var.config_ca
  }
  git = {
    url = "https://github.com/${var.github_repository}.git"
    http = {
      username = "git"
      password = var.github_token
    }
  }
}

resource "flux_bootstrap_git" "this" {
  path = var.target_path
}