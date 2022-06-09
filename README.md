# openstack-role-helper

`openstack-role-helper` is a wrapper around the OpenStack CLI and is used for performing
mass role operations, e.g. migrating all user/groups from an existing role to a new role.

## Installation

```
go install github.com/sapcc/openstack-role-helper
```

Alternatively, you can build with `make` or install with `make install`. The latter
understands the conventional environment variables for choosing install locations:
`DESTDIR` and `PREFIX`.

## Usage

> **Note**: OpenStack CLI needs to be installed because `openstack-role-helper` uses it to perform all its operations.

For usage instructions:

```
openstack-role-helper --help
```
