# CRD-Controller
目前，现在很多crd自定义的项目，都是基于 kubebuilder 或 operator-SDK 作为脚手架，该项目完全是全部手动完成，旨在帮助理解 operator 的核心。

## crd 设计
```yaml
apiVersion: extension.k8s.io/v1
kind: UnitedDeployment
metadata:
  name: example-united-deployment
spec:
  deploymentName: example-deployment
  replicas: 2
```
