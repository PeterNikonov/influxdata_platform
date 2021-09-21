# This repository is archived.

The `platform` repository contains the initial history of InfluxDB 2.0.

The master branch of the [influxdb project](https://github.com/influxdata/influxdb)
now contains all of the 2.0 work and git history so far.

This repository will remain archived in its current state indefinitely,
so that projects dependent on `platform` will continue to work.

Moving forward, we recommend that you update your dependencies to use influxdb instead.
If you were using Go modules to depend on `platform`, you may also consider running
`go mod edit -replace=github.com/influxdata/platform=/dev/null`
so that you don't accidentally reintroduce a dependency on `platform`
as you transition to `influxdb`.
