# Volume Snapshot Location Configurable Parameters

The AWS plugin supports several configurable parameters when defining a `VolumeSnapshotLocation`. These should be provided as key-value pairs to the `velero install` command's `--snapshot-location-config` flag, or under the `VolumeSnapshotLocation's` `spec.config` field.

| Key | Type | Default | Meaning |
| --- | --- | --- | --- |
| `region` | string | Empty | *Example*: "us-east-1"<br><br>See [AWS documentation][0] for the full list.<br><br>Required. |
| `profile` | string | "default" | AWS profile within the credential file to use for given store |

[0]: http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-regions-availability-zones.html#concepts-available-regions
