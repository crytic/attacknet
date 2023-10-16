resource "digitalocean_vpc" "attacknet" {
  name     = local.cluster_name
  region   = var.region
  ip_range = "10.221.0.0/16"
}

resource "digitalocean_project" "attacknet" {
  name        = "Attacknet"
  description = "Attacknet testing infrastructure"
  purpose     = "Other"
  environment = "Development"
}

resource "digitalocean_project_resources" "attacknet" {
  project = digitalocean_project.attacknet.id

  resources = [
    digitalocean_kubernetes_cluster.attacknet.urn,
  ]
}


resource "digitalocean_kubernetes_cluster" "attacknet" {
  name     = local.cluster_name
  region   = var.region
  version  = "1.26.7-do.0"
  vpc_uuid = digitalocean_vpc.attacknet.id
  tags     = local.common_tags

  lifecycle {
    ignore_changes = [
      node_pool[0].node_count,
      node_pool[0].nodes,
    ]
  }

  node_pool {
    name       = "${local.cluster_name}-default"
    size       = "s-8vcpu-16gb-amd" # $320/month,  list available options with `doctl compute size list`
    labels     = {}
    node_count = 1
    auto_scale = true
    max_nodes  = 50
    min_nodes  = 10
    tags       = concat(local.common_tags, ["default"])
  }
}
