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

data kube_manifest_files "all" {
    filenames=[
        "${abspath(path.module)}/manifests/*.yaml",
    ]
    variables = {
        namespace = "example"
        replicas = 2
        image = "nginx:latest"
    }
}

data "kube_manifest" "document"{
    for_each = data.kube_manifest_files.all.documents
    text = each.value.text 
}

resource "kube_resource" "cluster" {
    for_each = { for k,v in data.kube_manifest_files.all.documents: k => v if v.metadata.namespace == "" }   
    manifest = data.kube_manifest.document[each.key].manifest
}

resource "kube_resource" "namespaced" {
    for_each = { for k,v in data.kube_manifest_files.all.documents: k => v if v.metadata.namespace != "" }   
    manifest = data.kube_manifest.document[each.key].manifest
    fetch = {
        "conditions": {
            field = "status.conditions"
        }
    }
    depends_on = [kube_resource.cluster]
}

output "documents" {
    value = data.kube_manifest.document[*]
}