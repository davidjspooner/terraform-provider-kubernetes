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

data kube_manifest_files "nginx" {
    filenames=[
        "${abspath(path.module)}/manifests/*.yaml",
    ]
    variables = {
        namespace = "example"
        replicas = 3
        image = "nginx:latest"
    }
}

output "documents" {
    value = data.kube_manifest_files.nginx.manifests
}