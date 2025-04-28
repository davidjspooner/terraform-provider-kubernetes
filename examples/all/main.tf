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
    api_options = {
        retry = {
            attempts = 6
            interval = "1s,5s,10s"
            timeout = "5m"
        }
    }
}


data kube_manifest_documents "all" {
    file_sets = [
        {
            paths = [
                "${abspath(path.module)}/manifests/*.yaml"
            ]
            variables = {
                namespace = "example"
                replicas = 2
                image = "nginx:latest"
            }
            template_type = "go/text"
        }
    ]
}

data "kube_parsed_manifest" "document"{
    for_each = data.kube_manifest_documents.all.documents
    text = each.value.text 
}

resource "kube_applied_manifest" "cluster" {
    for_each = { for k,v in data.kube_manifest_documents.all.documents: k => v if v.metadata.namespace == "" }   
    manifest = data.kube_parsed_manifest.document[each.key].manifest
}

resource "kube_applied_manifest" "namespaced" {
    for_each = { for k,v in data.kube_manifest_documents.all.documents: k => v if v.metadata.namespace != "" }   
    manifest = data.kube_parsed_manifest.document[each.key].manifest
    fetch = {
        "conditions": {
            field = "status.conditions"
        }
    }
    depends_on = [kube_applied_manifest.cluster]
}

data "kube_files" "test" {
    file_sets = [
        {
            paths = [
                "${abspath(path.module)}/files/*.txt"
            ]
        }
    ]
}

resource "kube_applied_manifest" "secret" {
    manifest = {
        apiVersion = "v1"
        kind = "Secret"
        metadata = {
            name = "test"
            namespace = "example"
        }
        type = "Opaque"
        stringData = merge(data.kube_files.test.contents,{
            "inline_test1" = "content_inline_test_1"
        })
        data = {
            "inline_test2" = base64encode("content_inline_test_2")
        }
    }
}
resource "kube_applied_manifest" "configmap" {
    manifest = {
        apiVersion = "v1"
        kind = "ConfigMap"
        metadata = {
            name = "test"
            namespace = "example"
        }
        data = merge(data.kube_files.test.contents,{
            "inline_test1" = "content_inline_test_1"
        })
    }
}

data "kube_query" "test" {
    api_version = "v1"
    kind = "ConfigMap"
    metadata = {
        namespace = "example"
        name = "test"
    }
    fetch = {
        "data": {
            field = "data.inline_test1"
        }
    }
    depends_on = [kube_applied_manifest.configmap]
}


output "namespace" {
    value = data.kube_query.test.output.data
}

data "kube_current_context" "current" {
    # This will be the current context
}
output "current_context" {
    value = data.kube_current_context.current.authinfo
}
output "current_context_cluster" {
    value = data.kube_current_context.current.cluster
}
output "current_context_user" {
    value = data.kube_current_context.current.namespace
}