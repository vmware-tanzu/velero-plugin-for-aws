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

  # The credentials intended to be used with this location.
  # optional (if not set, default credentials secret is used)
  credential:
    # Key within the secret data which contains the cloud credentials
    key: cloud
    # Name of the secret containing the credentials
    name: cloud-credentials

  config:
    # The AWS region where the volumes/snapshots are located.
    #
    # Required.
    region: us-east-1

    # AWS profile within the credentials file to use for the volume snapshot location.
    #
    # Optional (defaults to "default").
    profile: "default"

    # Set this to "true" if you want to load the credentials file as a [shared config file](https://docs.aws.amazon.com/sdkref/latest/guide/file-format.html).
    # This will have no effect if credentials are not specific for a VSL.
    #
    # Optional (defaults to "false").
    enableSharedConfig: "true"

    # The KMS key ID to use for encrypting EBS volumes restored from snapshots.
    # If not specified, volumes will inherit encryption settings from the snapshot.
    # Supports multiple formats: Key ID, Key alias (e.g., "alias/my-key"),
    # Key ARN, or Alias ARN.
    #
    # Optional.
    ebsKmsKeyId: "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012"
```
