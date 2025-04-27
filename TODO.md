#TODO 


consider

* kube_crd
* kube_resource_selector


```
data "kube_resource_selector" "example_pods" {
  kind      = "Pod"
  namespace = "default"
  labels = {
    app = "nginx"
  }
}

resource "kube_crd" "example" {
  metadata = {
    name = "widgets.example.com"
    labels = {
      app = "widgets"
    }
  }

  spec = {
    group = "example.com"
    versions = [
      {
        name    = "v1"
        served  = true
        storage = true
        schema = {
          openAPIV3Schema = {
            type = "object"
            properties = {
              spec = {
                type = "object"
                properties = {
                  size = { type = "integer" }
                  color = { type = "string" }
                }
              }
            }
          }
        }
      }
    ]
    scope = "Namespaced"
    names = {
      plural     = "widgets"
      singular   = "widget"
      kind       = "Widget"
      shortNames = ["wdg"]
    }
  }

  wait_until_available = true   # default true
  wait_timeout_seconds = 60     # optional
}
```