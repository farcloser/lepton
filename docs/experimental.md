# Experimental features of nerdctl

The following features are experimental and subject to change.
See [`./config.md`](config.md) about how to enable these features.

- [Windows containers](https://github.com/containerd/nerdctl/issues/28)
- [Image Sign and Verify (cosign)](./cosign.md)
- [Image Sign and Verify (notation)](./notation.md)
- [Rootless container networking acceleration with bypass4netns](./rootless.md#bypass4netns)
- [Interactive debugging of Dockerfile](./builder-debug.md)
- Kubernetes (`cri`) log viewer: `nerdctl --namespace=k8s.io logs`
