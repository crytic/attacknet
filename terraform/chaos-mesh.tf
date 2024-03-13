
resource "kubernetes_namespace" "chaos-mesh" {
  metadata {
    name = "chaos-mesh"
  }
}

resource "helm_release" "chaos-mesh" {
  name       = "chaos-mesh"
  namespace  = kubernetes_namespace.chaos-mesh.metadata[0].name
  repository = "https://charts.chaos-mesh.org"
  chart      = "chaos-mesh"
  version    = "2.6.1"
  depends_on = [kubernetes_namespace.chaos-mesh]
  set {
    name  = "chaosDaemon.runtime"
    value = "containerd"
  }
  set {
    name  = "chaosDaemon.socketPath"
    value = "/run/containerd/containerd.sock"
  }
}

resource "kubernetes_service_account" "dashboard" {
  metadata {
    name      = "account-cluster-manager-dashboard"
    namespace = "default"
  }

}

resource "kubernetes_cluster_role" "dashboard-role" {
  metadata {
    name = "role-account-cluster-manager-dashboard"
  }

  rule {
    api_groups = [""]
    resources  = ["namespaces", "pods"]
    verbs      = ["get", "list", "watch"]
  }
  rule {
    api_groups = ["chaos-mesh.org"]
    resources  = ["*"]
    verbs      = ["get", "list", "watch", "create", "delete", "patch", "update"]
  }
}

resource "kubernetes_cluster_role_binding" "dashboard-role-binding" {
  metadata {
    name = "role-binding-account-cluster-manager-dashboard"
  }
  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "role-account-cluster-manager-dashboard"
  }
  subject {
    kind      = "ServiceAccount"
    name      = "account-cluster-manager-dashboard"
    namespace = "default"
  }
}