# CRD-Controller
目前，现在很多crd自定义的项目，都是基于 kubebuilder 或 operator-SDK 作为脚手架，该项目完全是全部手动完成，旨在帮助理解 operator 的核心。

## crd 设计
```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: unitedDeployments.extension.k8s.io
  # for more information on the below annotation, please see
  # https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/2337-k8s.io-group-protection/README.md
  annotations:
    "api-approved.kubernetes.io": "unapproved, experimental-only; please get an approval from Kubernetes API reviewers if you're trying to develop a CRD in the *.k8s.io or *.kubernetes.io groups"
spec:
  group: extension.k8s.io
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        # schema used for validation
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                deploymentName:
                  type: string
                replicas:
                  type: integer
                  minimum: 1
                  maximum: 10
            status:
              type: object
              properties:
                availableReplicas:
                  type: integer
  names:
    kind: UnitedDeployment
    plural: unitedDeployments
  scope: Namespaced
```
## cr 示例
```yaml
apiVersion: extension.k8s.io/v1
kind: UnitedDeployment
metadata:
  name: example-united-deployment
spec:
  deploymentName: example-deployment
  replicas: 2
```
## ref
```text
sample-controller：https://github.com/kubernetes/sample-controller/tree/v0.28.0

code-generator：https://github.com/kubernetes/code-generator/tree/v0.28.0

custom-resource-definitions: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/
```