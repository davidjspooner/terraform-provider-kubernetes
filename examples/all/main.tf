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

resource "kubernetes_manifest" "test_namespace" {
    manifest_content = <<__EOF__
        apiVersion: v1
        kind: Namespace
        metadata:
            name: test5
        __EOF__
}


resource "kubernetes_manifest" "test_deployment" {
    depends_on = [kubernetes_manifest.test_namespace]
#    api_options = {
#        retry = {
#            timeout = "3m"
#        }
#    }
    manifest = {
        metadata = {
            name = "test-deployment6"
            namespace = "test5"
        }
        spec = {
            replicas = 2
            selector = {
                matchLabels = {
                    app = "test4"
                }
            }
            template = {
                metadata = {
                    labels = {
                        app = "test4"
                    }
                }
                spec = {
                    containers = [
                        {
                            name  = "test"
                            image = "nginx"
                            ports = [
                                {
                                    containerPort = 80
                                }
                            ]
                        },
                        {
                            name  = "sidecar"
                            image = "nginx"
                        }
                    ]
                }
            }
        }
    }

    fetch= {
        generation={
            field = "metadata.generation"
        }
    }

    manifest_content = <<__EOF__
        apiVersion: apps/v1
        kind: Deployment
        metadata:
            name: test-deployment
            namespace: test5
        spec:
            replicas: 1
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

output "generation" {
  value = kubernetes_manifest.test_deployment.output.generation
}