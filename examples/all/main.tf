terraform {
    required_version = ">= 1.5.0"
    backend "local" {
        path = "terraform.tfstate"
    }
    required_providers {
        kube = {
            source = "dstower.home.dolbyn.com/davidjspooner/kubernetes"
        }
    }
}

provider "kube" {
  
}

resource "kube_manifest" "test_namespace" {
    api_options = {
        retry = {
            timeout = "3m"
        }
    }
    manifest = {
        apiVersion = "v1"
        kind = "Namespace"
        metadata = {
            name = "test5"
        }
    }
    fetch= {
        status={
            field = "status.phase"
        }
    }
}


resource "kube_manifest" "test_deployment" {
    depends_on = [kube_manifest.test_namespace]
    api_options = {
        retry = {
            timeout = "3m"
        }
    }
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


}

output "generation" {
  value = kube_manifest.test_deployment.output.generation
}