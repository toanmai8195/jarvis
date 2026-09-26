"""Unit test (bazel_skylib unittest) cho hàm thuần của com_tm_container.bzl."""

load("@bazel_skylib//lib:unittest.bzl", "asserts", "unittest")
load(":com_tm_container.bzl", "image_names", "image_tag")

def _image_names_without_repository_test(ctx):
    env = unittest.begin(ctx)
    asserts.equals(
        env,
        {
            "binary": "core",
            "docker": "core_docker",
            "image": "core_image",
            "layer": "core_layer",
        },
        image_names("core"),
    )
    asserts.false(env, "push" in image_names("core", repository = ""), "repository rỗng không sinh _push")
    return unittest.end(env)

image_names_without_repository_test = unittest.make(_image_names_without_repository_test)

def _image_names_with_repository_test(ctx):
    env = unittest.begin(ctx)
    names = image_names("stats-worker", repository = "ghcr.io/org/stats-worker")
    asserts.equals(env, "stats-worker_push", names["push"])
    asserts.equals(env, "stats-worker_image", names["image"])
    asserts.equals(env, 5, len(names))
    return unittest.end(env)

image_names_with_repository_test = unittest.make(_image_names_with_repository_test)

def _image_tag_test(ctx):
    env = unittest.begin(ctx)
    asserts.equals(env, "com.tm.go.smoke:v1.0.0", image_tag("smoke"))
    asserts.equals(env, "com.tm.go.core:v2.3.4", image_tag("core", tag = "v2.3.4"))
    return unittest.end(env)

image_tag_test = unittest.make(_image_tag_test)

def _image_tag_with_image_name_test(ctx):
    env = unittest.begin(ctx)
    asserts.equals(env, "com.tm.go.core-server:v1.0.0", image_tag("server", image = "core-server"))
    asserts.equals(env, "com.tm.go.core-worker:v1.0.0", image_tag("worker", image = "core-worker"))
    asserts.equals(env, "com.tm.go.core-worker:v2.0.0", image_tag("worker", tag = "v2.0.0", image = "core-worker"))
    asserts.equals(env, "com.tm.go.worker:v1.0.0", image_tag("worker", image = None), "image None → theo name")
    asserts.equals(env, "com.tm.go.worker:v1.0.0", image_tag("worker", image = ""), "image rỗng → theo name")
    return unittest.end(env)

image_tag_with_image_name_test = unittest.make(_image_tag_with_image_name_test)

def com_tm_container_test_suite(name):
    unittest.suite(
        name,
        image_names_without_repository_test,
        image_names_with_repository_test,
        image_tag_test,
        image_tag_with_image_name_test,
    )
