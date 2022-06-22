# openstack-role-helper

`openstack-role-helper` is a tool for performing mass role operations, e.g. migrating all
users/groups from an existing role to a new role.

## Installation

```
go install github.com/sapcc/openstack-role-helper
```

Alternatively, you can build with `make` or install with `make install`. The latter
understands the conventional environment variables for choosing install locations:
`DESTDIR` and `PREFIX`.

## Usage

```
openstack-role-helper --help
```
