terraform {
  backend "s3" {
    bucket     = "terraform-state-storage-586877430255"
    lock_table = "terraform-state-lock-586877430255"
    region     = "us-west-2"

    // THIS MUST BE UNIQUE
    key = "pc-config.tfstate"
  }
}

provider "aws" {
  region = "us-west-2"
}

data "aws_ssm_parameter" "eks_cluster_endpoint" {
  name = "/eks/av-cluster-endpoint"
}

provider "kubernetes" {
  host = data.aws_ssm_parameter.eks_cluster_endpoint.value
}

data "aws_ssm_parameter" "couch_address" {
  name = "/env/couch-new-address"
}

data "aws_ssm_parameter" "couch_username" {
  name = "/env/couch-username"
}

data "aws_ssm_parameter" "couch_password" {
  name = "/env/couch-password"
}

module "dev" {
  source = "github.com/byuoitav/terraform//modules/kubernetes-deployment"

  // required
  name           = "pc-config-dev"
  image          = "docker.pkg.github.com/byuoitav/pc-config/pc-config-dev"
  image_version  = "70fb49d"
  container_port = 8080
  repo_url       = "https://github.com/byuoitav/pc-config"

  // optional
  image_pull_secret = "github-docker-registry"
  public_urls       = ["pc-config.av.byu.edu"]
  container_env     = {}
  container_args = [
    "--port", "8080",
    "--log-level", "0", // set log level to info
    "--db-address", data.aws_ssm_parameter.couch_address.value,
    "--db-username", data.aws_ssm_parameter.couch_username.value,
    "--db-password", data.aws_ssm_parameter.couch_password.value
  ]
}
