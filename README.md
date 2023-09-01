# CRD-Controller
目前，现在很多crd自定义的项目，都是基于 kubebuilder 或 operator-SDK 作为脚手架，该项目完全是全部手动完成，旨在帮助理解 operator 的核心。

## cr 示例
```yaml
apiVersion: extensions.k8s.io/v1
kind: UnitedDeployment
metadata:
  name: examples-united-deployment
spec:
  deploymentName: examples-deployment
  replicas: 2
```
## ref
```text
sample-controller：https://github.com/kubernetes/sample-controller/tree/v0.28.0

code-generator：https://github.com/kubernetes/code-generator/tree/v0.28.0

custom-resource-definitions: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/
```