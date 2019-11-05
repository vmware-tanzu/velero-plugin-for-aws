# Backup Storage Location Configurable Parameters

The AWS plugin supports several configurable parameters when defining a `BackupStorageLocation`. These should be provided as key-value pairs to the `velero install` command's `--backup-location-config` flag, or under the `BackupStorageLocation's` `spec.config` field.

| Key | Type | Default | Meaning |
| --- | --- | --- | --- |
| `region` | string | Empty | *Example*: "us-east-1"<br><br>See [AWS documentation][0] for the full list.<br><br>Queried from the AWS S3 API if not provided. |
| `s3ForcePathStyle` | bool | `false` | Set this to `true` if you are using a local storage service like Minio. |
| `s3Url` | string | Required field for non-AWS-hosted storage| *Example*: http://minio:9000<br><br>You can specify the AWS S3 URL here for explicitness, but Velero can already generate it from `region`, and `bucket`. This field is primarily for local storage services like Minio.|
| `publicUrl` | string | Empty | *Example*: https://minio.mycluster.com<br><br>If specified, use this instead of `s3Url` when generating download URLs (e.g., for logs). This field is primarily for local storage services like Minio.|
| `serverSideEncryption` | string | Empty | The name of the server-side encryption algorithm to use for uploading objects, e.g. `AES256`. If using SSE-KMS and `kmsKeyId` is specified, this field will automatically be set to `aws:kms` so does not need to be specified by the user. | 
| `kmsKeyId` | string | Empty | *Example*: "502b409c-4da1-419f-a16e-eif453b3i49f" or "alias/`<KMS-Key-Alias-Name>`"<br><br>Specify an [AWS KMS key][1] id or alias to enable encryption of the backups stored in S3. Only works with AWS S3 and may require explicitly granting key usage rights.|
| `signatureVersion` | string | `"4"` | Version of the signature algorithm used to create signed URLs that are used by velero cli to download backups or fetch logs. Possible versions are "1" and "4". Usually the default version 4 is correct, but some S3-compatible providers like Quobyte only support version 1.|
| `profile` | string | "default" | AWS profile within the credential file to use for given store |
| `insecureSkipTLSVerify` | bool | `false` | Set this to `true` if you do not want to verify the TLS certificate when connecting to the object store--like self-signed certs in Minio. This is susceptible to man-in-the-middle attacks and is not recommended for production. |

[0]: http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-regions-availability-zones.html#concepts-available-regions
[1]: http://docs.aws.amazon.com/kms/latest/developerguide/overview.html
