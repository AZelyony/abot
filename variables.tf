#variable "GOOGLE_PROJECT" {
#  type        = string
#  description = "GCP project name"
#}

#variable "GOOGLE_REGION" {
#  type        = string
#  #default     = "us-central1-c"
#  default     = "europe-west4"
#  description = "GCP region to use"
#}

#variable "GOOGLE_BUCKET" {
#  type        = string
#  description = "GCS bucket name"
#}

#variable "GKE_NUM_NODES" {
#  type        = number
#  default     = 1
#  description = "GKE nodes number"
#}

variable "GITHUB_OWNER" {
  type        = string
  description = "GitHub owner repository to use"
}

variable "GITHUB_TOKEN" {
  type        = string
  description = "GitHub personal access token"
}

variable "FLUX_GITHUB_REPO" {
  type        = string
  default     = "flux-gitops"
  description = "Flux GitOps repository"
}

variable "FLUX_GITHUB_TARGET_PATH" {
  type        = string
  default     = "kind/dev"
  description = "Flux manifests subdirectory"
}