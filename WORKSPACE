workspace(name = "kubecontrollerdemo")

# Pull in rules_go
git_repository(
    name = "io_bazel_rules_go",
    remote = "https://github.com/bazelbuild/rules_go.git",
    tag = "0.5.3",
)
load("@io_bazel_rules_go//go:def.bzl", "go_repositories")

go_repositories()

# Pull in rules_docker
git_repository(
    name = "io_bazel_rules_docker",
    commit = "27b494ceefedd35b0ae72100860997f7ab1bf714",
    remote = "https://github.com/bazelbuild/rules_docker.git",
)

load(
    "@io_bazel_rules_docker//docker:docker.bzl",
    "docker_repositories",
)

docker_repositories()

# Pull in the go_image contrib stuff.
load(
    "@io_bazel_rules_docker//docker/contrib/go:image.bzl",
    _go_image_repos = "repositories",
)

_go_image_repos()

# Pull in rules_k8s
git_repository(
    name = "io_bazel_rules_k8s",
    remote = "https://github.com/mattmoor/rules_k8s",
    commit = "57d84456f80af89cc76fe43375d184b50f48ca19",
)

load("@io_bazel_rules_k8s//k8s:k8s.bzl", "k8s_repositories", "k8s_defaults")

k8s_repositories()

k8s_defaults(
    # This becomes the name of the @repository and the rule
    # you will import in your BUILD files.
    name = "k8s_daemonset",
    kind = "daemonset",
    cluster = "gke_convoy-adapter_us-central1-f_bazel-grpc",
)

k8s_defaults(
    # This becomes the name of the @repository and the rule
    # you will import in your BUILD files.
    name = "k8s_crd",
    kind = "customresourcedefinition",
    cluster = "gke_convoy-adapter_us-central1-f_bazel-grpc",
)

k8s_defaults(
    # This becomes the name of the @repository and the rule
    # you will import in your BUILD files.
    name = "k8s_deploy",
    kind = "deployment",
    cluster = "gke_convoy-adapter_us-central1-f_bazel-grpc",
)

k8s_defaults(
    # This becomes the name of the @repository and the rule
    # you will import in your BUILD files.
    name = "k8s_package",
    kind = "package",
    cluster = "gke_convoy-adapter_us-central1-f_bazel-grpc",
)
