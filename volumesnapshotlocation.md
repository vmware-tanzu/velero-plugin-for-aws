# Volume Snapshot Location

The following sample AWS `VolumeSnapshotLocation` YAML shows all of the configurable parameters. The items under `spec.config` can be provided as key-value pairs to the `velero install` command's `--snapshot-location-config` flag -- for example, `region=us-east-1,profile=secondary,...`.

```yaml
apiVersion: velero.io/v1
kind: VolumeSnapshotLocation
metadata:
  name: aws-default
  namespace: velero
spec:
  # Name of the volume snapshotter plugin to use to connect to this location.
  #
  # Required.
  provider: velero.io/aws
  
  config:
    # The AWS region where the volumes/snapshots are located.
    #
    # Required.
    region: us-east-1

    # AWS profile within the credentials file to use for the volume snapshot location.
    # 
    # Optional (defaults to "default").
    profile: "default"
```
