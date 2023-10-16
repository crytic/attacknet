terraform {
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.11.1"
    }

    helm = {
      source  = "hashicorp/helm"
      version = "2.4.1"
    }

    kubectl = {
      source  = "gavinbunney/kubectl"
      version = "1.13.1"
    }

    sops = {
      source  = "carlpett/sops"
      version = "0.7.1"
    }

    bcrypt = {
      source  = "viktorradnai/bcrypt"
      version = "0.1.2"
    }
    rancher2 = {
      source  = "rancher/rancher2"
      version = "1.24.2"
    }
  }

  backend "s3" {
    skip_credentials_validation = true
    skip_metadata_api_check     = true
    endpoint                    = "https://ams3.digitaloceanspaces.com"
    region                      = "us-east-1"
    bucket                      = "yourbucketname"
    key                         = "attacknet/terraform.tfstate"
  }
}


variable "do_token" {
  sensitive = true
}
