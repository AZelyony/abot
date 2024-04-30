module "gke_cluster" {
  source         = "github.com/azelyony/tf-google-gke-cluster"
  GOOGLE_REGION  = var.GOOGLE_REGION
  GOOGLE_PROJECT = var.GOOGLE_PROJECT
  GKE_NUM_NODES  = 2
}

terraform {
  backend "gcs" {
    bucket = var.GOOGLE_BUCKET
    prefix = "terraform/state"
  }
}

# # Source: "https://github.com/den-vasyliev/tf-github-repository"
module "github_repository" {
  source                   = "./modules/tf-github-repository"
  github_owner             = var.GITHUB_OWNER
  github_token             = var.GITHUB_TOKEN
  repository_name          = var.FLUX_GITHUB_REPO
  public_key_openssh       = module.tls_private_key.public_key_openssh
  public_key_openssh_title = "flux"
}

# # Source: "https://github.com/den-vasyliev/tf-hashicorp-tls-keys"
module "tls_private_key" {
  source    = "./modules/tf-hashicorp-tls-keys"
  algorithm = "RSA"
}

# # Source: "http://github.com/den-vasyliev/tf-fluxcd-flux-bootstrap"
module "flux_bootstrap" {
  source            = "./modules/flux_bootstrap/"
  github_repository = "${var.GITHUB_OWNER}/${var.FLUX_GITHUB_REPO}"
  private_key       = module.tls_private_key.private_key_pem
  config_host       = module.kind_cluster.endpoint
  config_client_key = module.kind_cluster.client_key
  config_ca         = module.kind_cluster.ca
  config_crt        = module.kind_cluster.crt
  github_token      = var.GITHUB_TOKEN
}