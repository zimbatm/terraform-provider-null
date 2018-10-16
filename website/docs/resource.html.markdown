---
layout: "null"
page_title: "Null Resource"
sidebar_current: "docs-null-resource"
description: |-
  A resource that does nothing.
---

# Null Resource

The `null_resource` resource implements the standard resource lifecycle but
takes no further action.

The `triggers` argument allows specifying an arbitrary set of values that,
when changed, will cause the resource to be replaced.

## Example Usage

The primary use-case for the null resource is as a do-nothing container for
arbitrary actions taken by a provisioner, as follows:

```hcl
resource "aws_instance" "cluster" {
  count = 3

  # ...
}

resource "null_resource" "cluster" {
  # Changes to any instance of the cluster requires re-provisioning
  triggers = {
    cluster_instance_ids = "${join(",", aws_instance.cluster.*.id)}"
  }

  # Bootstrap script can run on any instance of the cluster
  # So we just choose the first in this case
  connection {
    host = "${element(aws_instance.cluster.*.public_ip, 0)}"
  }

  provisioner "local-exec" {
    # Bootstrap script called with private_ip of each node in the clutser
    command = "bootstrap-cluster.sh ${join(" ", aws_instance.cluster.*.private_ip)}"
  }
}
```

In this example, three EC2 instances are created and then a
`null_resource` instance is used to gather data about all three and execute
a single action that affects them all. Due to the `triggers` map, the
`null_resource` will be replaced each time the instance ids change, and thus
the `remote-exec` provisioner will be re-run.

## Argument Reference

The following arguments are supported:

* `triggers` - (Optional) A map of arbitrary strings that, when changed, will
  force the null resource to be replaced, re-running any associated
provisioners.
* `external_trigger` - (Optional) Run an external command to determine if the
  resource should be re-created. See [External Trigger](#external-triggers)
  below for details.

### External triggers

~> **Warning** Terraform Enterprise does not guarantee availability of any
particular language runtimes or external programs beyond standard shell
utilities, so it is not recommended to use this provider within configurations
that are applied within Terraform Enterprise.

Each `external_command` is executed on resource creation and read. If the
output changes it triggers the resource re-creation. This allows for advanced
deployment scenarios.

The following arguments are supported:

* `command` - (Required) This is the command to execute. It can be provided as a relative path to the current working directory or as an absolute path. It is evaluated in a shell, and can use environment variables or Terraform variables.

* `working_dir` - (Optional) If provided, specifies the working directory where command will be executed. It can be provided as as a relative path to the current working directory or as an absolute path. The directory must exist.

* `interpreter` - (Optional) If provided, this is a list of interpreter arguments used to execute the command. The first argument is the interpreter itself. It can be provided as a relative path to the current working directory or as an absolute path. The remaining arguments are appended prior to the command. This allows building command lines of the form "/bin/bash", "-c", "echo foo". If interpreter is unspecified, sensible defaults will be chosen based on the system OS.

* `environment` - (Optional) block of key value pairs representing the environment of the executed command. inherits the current process environment.

### Example

In the case where the trigger should only happen on arbitrary commands, it's
also possible to use the `external_trigger` method.

```hcl
resource "google_compute_instance" "myhost" {
  # ...
}

locals {
  host = "${google_compute_instance.myhost.network_interface.0.network_ip}"
}

resource "null_resource" "nixos_deploy" {
  triggers {
    # check if it's a new instance
    instance_id = "${google_compute_instance.myhost.instance_id}"
  }

  # check if the nix evaluation output has changed
  external_trigger {
    command = "nix-instantiate ${path.module}/files"
  }

  connection {
    type  = "ssh"
    host  = "${local.host}"
    user  = "root"
    agent = true
  }

  # check that SSH is working
  provisioner "remote-exec" {
    inline = ["uname -a"]
  }

  # the deploy command
  provisioner "local-exec" {
    command = "nixos-rebuild --target-host root@${local.host} --build-host root@${local.host} -I nixos-config=${path.module}/files/configuration.nix switch"
  }
}
```

## Attributes Reference

The following attributes are exported:

* `id` - An arbitrary value that changes each time the resource is replaced.
  Can be used to cause other resources to be updated or replaced in response
  to `null_resource` changes.

For any `external_trigger` the command `output` is exported.
