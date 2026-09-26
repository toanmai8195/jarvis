"""Macro com_tm_go_image: go_binary + OCI image distroless.

Gazelle sinh macro này thay cho mọi `go_binary` nhờ directive trong BUILD.bazel gốc:

    # gazelle:map_kind go_binary com_tm_go_image //tools/rules:com_tm_container.bzl

Một lời gọi `com_tm_go_image(name = "x", ...)` sinh:

    x          go_binary (host khi build thường; Linux khi --config=linux-arm64|amd64)
    x_layer    tar chứa binary tại /app/x
    x_image    oci_image trên base distroless, entrypoint /app/x
    x_docker   oci_load, tag com.tm.go.x:<tag> (hoặc com.tm.go.<image>:<tag> khi
               truyền `image`, vd image = "core-server") — `bazel run` để nạp vào Docker
    x_push     oci_push — chỉ khi truyền `repository`

Image chỉ tương thích với platform Linux: `bazel build //...` trên macOS bỏ qua
(SKIPPED) các target image, còn binary và test vẫn build/chạy bình thường.
Build image: `bazel run --config=linux-arm64 //path:x_docker`.
"""

load("@rules_go//go:def.bzl", "go_binary")
load("@rules_oci//oci:defs.bzl", "oci_image", "oci_load", "oci_push")
load("@tar.bzl", "mutate", "tar")

_LINUX_ONLY = ["@platforms//os:linux"]

def image_names(name, repository = None):
    """Tên các target mà com_tm_go_image sinh ra.

    Args:
      name: tên go_binary.
      repository: registry đích; có thì sinh thêm `<name>_push`.

    Returns:
      dict: vai trò → tên target.
    """
    names = {
        "binary": name,
        "layer": name + "_layer",
        "image": name + "_image",
        "docker": name + "_docker",
    }
    if repository:
        names["push"] = name + "_push"
    return names

def image_tag(name, tag = "v1.0.0", image = None):
    """Tag Docker local của image: `com.tm.go.<image hoặc name>:<tag>`.

    Args:
      name: tên go_binary; dùng làm tên image khi không truyền `image`.
      tag: tag image, mặc định `v1.0.0`.
      image: tên image đặt tường minh, ví dụ `core-server` (quy tắc
        `<service>-<binary>` cho binary của service, tránh trùng giữa
        `services/core/cmd/worker` và `services/stats-worker/cmd/worker`).

    Returns:
      string: ví dụ `com.tm.go.core-server:v1.0.0`.
    """
    return "com.tm.go.%s:%s" % (image or name, tag)

def com_tm_go_image(
        name,
        image = None,
        repository = None,
        tag = "v1.0.0",
        base = "@distroless_static",
        visibility = None,
        tags = None,
        **kwargs):
    """go_binary + layer + oci_image + oci_load (+ oci_push khi có repository).

    Args:
      name: tên go_binary (Gazelle đặt theo thư mục); tên target và
        entrypoint `/app/<name>` luôn theo `name`.
      image: tên image cho tag Docker `com.tm.go.<image>:<tag>`; bỏ trống thì
        dùng `name`. Gazelle giữ nguyên attr này khi chạy lại (nó chỉ merge
        attr của go_binary như embed/srcs/deps). Không truyền xuống go_binary.
      repository: registry đích cho `<name>_push`, ví dụ `ghcr.io/org/core`.
      tag: tag image, mặc định `v1.0.0`.
      base: base image (oci.pull trong MODULE.bazel).
      visibility: visibility của go_binary.
      tags: tags của go_binary.
      **kwargs: truyền nguyên cho go_binary (embed, deps, srcs, x_defs...).
    """
    names = image_names(name, repository)

    go_binary(
        name = names["binary"],
        pure = "on",
        visibility = visibility,
        tags = tags,
        **kwargs
    )

    # go_binary xuất ra <pkg>/<name>_/<name>; đưa vào tar tại app/<name>.
    # include_runfiles = False: mặc định (mtree "auto") tar.bzl đóng gói cả
    # runfiles của go_binary — app/<name>.runfiles/_main/.../<name>, bản sao
    # thứ hai của binary, làm layer to gấp đôi. Binary Go tĩnh (pure) không
    # cần runfiles lúc chạy, nên layer chỉ gồm app/ và app/<name>.
    tar(
        name = names["layer"],
        srcs = [":" + names["binary"]],
        include_runfiles = False,
        mutate = mutate(
            strip_prefix = "%s/%s_" % (native.package_name(), name),
            package_dir = "app",
        ),
        target_compatible_with = _LINUX_ONLY,
    )

    oci_image(
        name = names["image"],
        base = base,
        entrypoint = ["/app/" + name],
        tars = [":" + names["layer"]],
        target_compatible_with = _LINUX_ONLY,
    )

    oci_load(
        name = names["docker"],
        image = ":" + names["image"],
        repo_tags = [image_tag(name, tag, image)],
        target_compatible_with = _LINUX_ONLY,
    )

    if repository:
        oci_push(
            name = names["push"],
            image = ":" + names["image"],
            repository = repository,
            remote_tags = [tag],
            target_compatible_with = _LINUX_ONLY,
        )
