# cosi-driver-sample

Sample Driver that provides reference implementation for Container Object Storage Interface (COSI) API

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack](https://kubernetes.slack.com/messages/sig-storage)
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-sig-storage)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).

## Deploy
1. Build
```
make build
make image
make kind
```

2. Deploy provisioner
```
kubectl apply -f resources
```

3. Deploy classes
```
kubectl apply -f sample/classes
```

4. Deploy bucketrequest & bucketaccessrequest
```
kubectl apply -f sample/accessrequest.yaml
kubectl apply -f sample/bucketaccessrequest.yaml
```

5. Deploy testapp
```
kubectl apply -f sample/pod
```
