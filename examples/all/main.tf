terraform {
  required_version = ">= 0.12"
  required_providers {
    kubernetes = {
      source = "dstower.home.dolbyn.com/davidjspooner/kubernetes"
    }
  }
}

provider "kubernetes" {
  
}

resource "kubernetes_resource" "test_namespace" {
    manifest = <<__EOF__
        apiVersion: v1
        kind: Namespace
        metadata:
            name: test5
        __EOF__
}


resource "kubernetes_resource" "test_deployment" {
    depends_on = [kubernetes_resource.test_namespace]
#    api_options = {
#        retry = {
#            timeout = "3m"
#        }
#    }
    manifest = <<__EOF__
        apiVersion: apps/v1
        kind: Deployment
        metadata:
            name: test-deployment
            namespace: test5
        spec:
            replicas: 2
            selector:
                matchLabels:
                    app: test3
            template:
                metadata:
                    labels:
                        app: test3
                spec:
                    containers:
                    - name: test
                      image: nginx
                      ports:
                      - containerPort: 80
        __EOF__
}