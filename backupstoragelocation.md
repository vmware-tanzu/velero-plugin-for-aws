# Backup Storage Location

The following sample AWS `BackupStorageLocation` YAML shows all of the configurable parameters. The items under `spec.config` can be provided as key-value pairs to the `velero install` command's `--backup-location-config` flag -- for example, `region=us-east-1,serverSideEncryption=AES256,...`.

```yaml
apiVersion: velero.io/v1
kind: BackupStorageLocation
metadata:
  name: default
  namespace: velero
spec:
  # Name of the object store plugin to use to connect to this location.
  #
  # Required.
  provider: velero.io/aws
  
  objectStorage:
    # The bucket in which to store backups.
    #
    # Required.
    bucket: my-bucket
    
    # The prefix within the bucket under which to store backups.
    #
    # Optional.
    prefix: my-prefix
  
  # The credentials intended to be used with this location.
  # optional (if not set, default credentials secret is used)
  credential:
    # Key within the secret data which contains the cloud credentials
    key: cloud
    # Name of the secret containing the credentials
    name: cloud-credentials

  config:
    # The AWS region where the bucket is located. Queried from the AWS S3 API if not provided.
    #
    # Optional if s3ForcePathStyle is false.
    region: us-east-1

    # Whether to use path-style addressing instead of virtual hosted bucket addressing. Set to "true"
    # if using a local storage service like MinIO.
    #
    # Optional (defaults to "false").
    s3ForcePathStyle: "true"

    # You can specify the AWS S3 URL here for explicitness, but Velero can already generate it from 
    # "region" and "bucket". This field is primarily for local storage services like MinIO.
    #
    # Optional.
    s3Url: "http://minio:9000"
    
    # If specified, use this instead of "s3Url" when generating download URLs (e.g., for logs). This 
    # field is primarily for local storage services like MinIO.
    #
    # Optional.
    publicUrl: "https://minio.mycluster.com"

    # The name of the server-side encryption algorithm to use for uploading objects, e.g. "AES256".
    # If using SSE-KMS and "kmsKeyId" is specified, this field will automatically be set to "aws:kms"
    # so does not need to be specified by the user.
    #
    # Optional.
    serverSideEncryption: AES256

    # Specify an AWS KMS key ID (formatted per the example) or alias (formatted as "alias/<KMS-key-alias-name>"), or its full ARN
    # to enable encryption of the backups stored in S3. Only works with AWS S3 and may require explicitly 
    # granting key usage rights. 
    #
    # Cannot be used in conjunction with customerKeyEncryptionFile.
    #
    # Optional.
    kmsKeyId: "502b409c-4da1-419f-a16e-eif453b3i49f"
    
    # Specify the file that contains the SSE-C customer key to enable customer key encryption of the backups
    # stored in S3. The referenced file should contain a 32-byte string.
    #  
    # The customerKeyEncryptionFile points to a mounted secret within the velero container.
    # Add the below values to the velero cloud-credentials secret:
    # customer-key: <your_b64_encoded_32byte_string>
    # The default value below points to the already mounted secret.
    # 
    # Cannot be used in conjunction with kmsKeyId.
    #
    # Optional (defaults to "", which means SSE-C is disabled).
    customerKeyEncryptionFile: "/credentials/customer-key"

    # Version of the signature algorithm used to create signed URLs that are used by velero CLI to 
    # download backups or fetch logs. Possible versions are "1" and "4". Usually the default version 
    # 4 is correct, but some S3-compatible providers like Quobyte only support version 1.
    #
    # Optional (defaults to "4").
    signatureVersion: "1"

    # AWS profile within the credentials file to use for the backup storage location.
    # 
    # Optional (defaults to "default").
    profile: "default"

    # Set this to "true" if you do not want to verify the TLS certificate when connecting to the 
    # object store -- like for self-signed certs with MinIO. This is susceptible to man-in-the-middle 
    # attacks and is not recommended for production.
    #
    # Optional (defaults to "false").
    insecureSkipTLSVerify: "true"

    # Set this to "true" if you want to load the credentials file as a [shared config file](https://docs.aws.amazon.com/sdkref/latest/guide/file-format.html).
    # This will have no effect if credentials are not specific for a BSL.
    #
    # Optional (defaults to "false").
    enableSharedConfig: "true"

    # Tags that need to be placed on AWS S3 objects. 
    # For example "Key1=Value1&Key2=Value2"
    #
    # Optional (defaults to empty "")
    tagging: ""

    # The checksum algorithm to use for uploading objects to S3.
    # The Supported values are  "CRC32",  "CRC32C", "SHA1", "SHA256".
    # If the value is set as empty string "", no checksum will be calculated and attached to 
    # the request headers.
    #
    # Optional (defaults to "CRC32")
    checksumAlgorithm: "CRC32"
```
