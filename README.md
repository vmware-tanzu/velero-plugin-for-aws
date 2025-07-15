[![Build Status][101]][102]

# Velero plugins for AWS

## Overview

This repository contains these plugins to support running Velero on AWS:

- An object store plugin for persisting and retrieving backups on AWS S3. Content of backup is kubernetes resources and metadata files for CSI objects, progress of async operations.  
  It is also used to store the result data of backups and restores include log files, warning/error files, etc.

- A volume snapshotter plugin for creating snapshots from volumes (during a backup) and volumes from snapshots (during a restore) on AWS EBS.
  - Since v1.4.0 the snapshotter plugin can handle the volumes provisioned by CSI driver `ebs.csi.aws.com`


## Compatibility

Below is a listing of plugin versions and respective Velero versions that are compatible.

| Plugin Version | Velero Version |
|----------------|----------------|
| v1.12.x        | v1.16.x        |
| v1.11.x        | v1.15.x        |
| v1.10.x        | v1.14.x        |
| v1.9.x         | v1.13.x        |
| v1.8.x         | v1.12.x        |

## Non-AWS S3 compatible provider known issues with plugin v1.10.x (aws-sdk-go-v2):
| Cloud Provider      |Notes|Velero Issue|Cloud Provider Issue|
|-|-|-|-|
|Google Cloud Storage|[Should use GCP plugin instead](https://github.com/vmware-tanzu/velero-plugin-for-gcp)||https://issuetracker.google.com/issues/256641357|
|Net App|`operation error S3: PutObject, https response error StatusCode: 501, RequestID: , HostID: , api error NotImplemented: The s3 command you requested is not implemented.`|https://github.com/vmware-tanzu/velero/issues/7828 https://github.com/vmware-tanzu/velero/issues/8152|[Fixed in ONTAP Release 9.15.1P2.](https://github.com/vmware-tanzu/velero/issues/8152#issuecomment-2464471703:~:text=issue%20is%20fixed%20using%20latest%20version%20NetApp%20Release%209.15.1P2.) <br> [Fixed in Net App StorageGRID® Version 11.8.0.7](https://github.com/vmware-tanzu/velero/issues/7828#issuecomment-2507228050)|
|Oracle||https://github.com/vmware-tanzu/velero/issues/8013||
|IBM COS|checksumAlgorithm="" should work if [retention is not enabled](https://github.com/vmware-tanzu/velero/issues/7543#issuecomment-2225803682)|https://github.com/vmware-tanzu/velero/issues/7543||
|Hitachi Content Platform (HCP)||||
|Cloudian||https://github.com/vmware-tanzu/velero/issues/8264||
|Qumulo|not compatible with `x-id`, etc.|https://github.com/vmware-tanzu/velero/issues/8312||||
|Ceph S3|checksumAlgorithm="" to avoid `api error XAmzContentSHA256Mismatch`||||
|Backblaze B2|checksumAlgorithm="" to avoid `api error XAmzContentSHA256Mismatch`||||
## Filing issues

If you would like to file a GitHub issue for the plugin, please open the issue on the [core Velero repo][103]


## Setup

To set up Velero on AWS, you:

* [Create an S3 bucket][1]
* [Set permissions for Velero][2]
* [Install and start Velero][3]

You can also use this plugin to [migrate PVs across clusters][5] or create an additional [Backup Storage Location][12].

If you do not have the `aws` CLI locally installed, follow the [user guide][6] to set it up.

## Create S3 bucket

Velero requires an object storage bucket to store backups in, preferably unique to a single Kubernetes cluster (see the [FAQ][11] for more details). Create an S3 bucket, replacing placeholders appropriately:

```bash
BUCKET=<YOUR_BUCKET>
REGION=<YOUR_REGION>
aws s3api create-bucket \
    --bucket $BUCKET \
    --region $REGION \
    --create-bucket-configuration LocationConstraint=$REGION
```
NOTE: us-east-1 does not support a `LocationConstraint`.  If your region is `us-east-1`, omit the bucket configuration:

```bash
aws s3api create-bucket \
    --bucket $BUCKET \
    --region us-east-1
```

## Set permissions for Velero

### Option 1: Set permissions with an IAM user

For more information, see [the AWS documentation on IAM users][10].

1. Create the IAM user:

    ```bash
    aws iam create-user --user-name velero
    ```

    If you'll be using Velero to backup multiple clusters with multiple S3 buckets, it may be desirable to create a unique username per cluster rather than the default `velero`.

2. Attach policies to give `velero` the necessary permissions (note that `s3:PutObjectTagging` is only needed 
   if you make use of the `config.tagging` field in the `BackupStorageLocation` spec):

    ```
    cat > velero-policy.json <<EOF
    {
        "Version": "2012-10-17",
        "Statement": [
            {
                "Effect": "Allow",
                "Action": [
                    "ec2:DescribeVolumes",
                    "ec2:DescribeSnapshots",
                    "ec2:CreateTags",
                    "ec2:CreateVolume",
                    "ec2:CreateSnapshot",
                    "ec2:DeleteSnapshot"
                ],
                "Resource": "*"
            },
            {
                "Effect": "Allow",
                "Action": [
                    "s3:GetObject",
                    "s3:DeleteObject",
                    "s3:PutObject",
                    "s3:PutObjectTagging",
                    "s3:AbortMultipartUpload",
                    "s3:ListMultipartUploadParts"
                ],
                "Resource": [
                    "arn:aws:s3:::${BUCKET}/*"
                ]
            },
            {
                "Effect": "Allow",
                "Action": [
                    "s3:ListBucket"
                ],
                "Resource": [
                    "arn:aws:s3:::${BUCKET}"
                ]
            }
        ]
    }
    EOF
    ```
    ```bash
    aws iam put-user-policy \
      --user-name velero \
      --policy-name velero \
      --policy-document file://velero-policy.json
    ```

3. Create an access key for the user:

    ```bash
    aws iam create-access-key --user-name velero
    ```

    The result should look like:

    ```json
    {
      "AccessKey": {
            "UserName": "velero",
            "Status": "Active",
            "CreateDate": "2017-07-31T22:24:41.576Z",
            "SecretAccessKey": <AWS_SECRET_ACCESS_KEY>,
            "AccessKeyId": <AWS_ACCESS_KEY_ID>
      }
    }
    ```

4. Create a Velero-specific credentials file (`credentials-velero`) in your local directory:

    ```bash
    [default]
    aws_access_key_id=<AWS_ACCESS_KEY_ID>
    aws_secret_access_key=<AWS_SECRET_ACCESS_KEY>
    ```

    where the access key id and secret are the values returned from the `create-access-key` request.


### Option 2: Set permissions using kube2iam

[Kube2iam](https://github.com/jtblin/kube2iam) is a Kubernetes application that allows managing AWS IAM permissions for pod via annotations rather than operating on API keys.

> This path assumes you have `kube2iam` already running in your Kubernetes cluster. If that is not the case, please install it first, following the docs here: [https://github.com/jtblin/kube2iam](https://github.com/jtblin/kube2iam)

It can be set up for Velero by creating a role that will have required permissions, and later by adding the permissions annotation on the velero deployment to define which role it should use internally.

1. Create a Trust Policy document to allow the role being used for EC2 management & assume kube2iam role:

    ```
    cat > velero-trust-policy.json <<EOF
    {
        "Version": "2012-10-17",
        "Statement": [
            {
                "Effect": "Allow",
                "Principal": {
                    "Service": "ec2.amazonaws.com"
                },
                "Action": "sts:AssumeRole"
            },
            {
                "Effect": "Allow",
                "Principal": {
                    "AWS": "arn:aws:iam::<AWS_ACCOUNT_ID>:role/<ROLE_CREATED_WHEN_INITIALIZING_KUBE2IAM>"
                },
                "Action": "sts:AssumeRole"
            }
        ]
    }
    EOF
    ```

2. Create the IAM role:

    ```bash
    aws iam create-role --role-name velero --assume-role-policy-document file://./velero-trust-policy.json
    ```

3. Attach policies to give `velero` the necessary permissions (note that `s3:PutObjectTagging` is only needed 
   if you make use of the `config.tagging` field in the `BackupStorageLocation` spec):

    ```
    BUCKET=<YOUR_BUCKET>
    cat > velero-policy.json <<EOF
    {
        "Version": "2012-10-17",
        "Statement": [
            {
                "Effect": "Allow",
                "Action": [
                    "ec2:DescribeVolumes",
                    "ec2:DescribeSnapshots",
                    "ec2:CreateTags",
                    "ec2:CreateVolume",
                    "ec2:CreateSnapshot",
                    "ec2:DeleteSnapshot"
                ],
                "Resource": "*"
            },
            {
                "Effect": "Allow",
                "Action": [
                    "s3:GetObject",
                    "s3:DeleteObject",
                    "s3:PutObject",
                    "s3:PutObjectTagging",
                    "s3:AbortMultipartUpload",
                    "s3:ListMultipartUploadParts"
                ],
                "Resource": [
                    "arn:aws:s3:::${BUCKET}/*"
                ]
            },
            {
                "Effect": "Allow",
                "Action": [
                    "s3:ListBucket"
                ],
                "Resource": [
                    "arn:aws:s3:::${BUCKET}"
                ]
            }
        ]
    }
    EOF
    ```
    ```bash
    aws iam put-role-policy \
      --role-name velero \
      --policy-name velero-policy \
      --policy-document file://./velero-policy.json
    ```

## Install and start Velero

[Download][4] Velero

Install Velero, including all prerequisites, into the cluster and start the deployment. This will create a namespace called `velero`, and place a deployment named `velero` in it.

**If using IAM user and access key**:

```bash
velero install \
    --provider aws \
    --plugins velero/velero-plugin-for-aws:v1.10.0 \
    --bucket $BUCKET \
    --backup-location-config region=$REGION \
    --snapshot-location-config region=$REGION \
    --secret-file ./credentials-velero
```

**If using kube2iam**:

```bash
velero install \
    --provider aws \
    --plugins velero/velero-plugin-for-aws:v1.10.0 \
    --bucket $BUCKET \
    --backup-location-config region=$REGION \
    --snapshot-location-config region=$REGION \
    --pod-annotations iam.amazonaws.com/role=arn:aws:iam::<AWS_ACCOUNT_ID>:role/<VELERO_ROLE_NAME> \
    --no-secret
```

Additionally, you can specify `--use-node-agent` to enable node agent support, and `--wait` to wait for the deployment to be ready.

```
securityContext:
        fsGroup: 65534
```

(Optional) Specify [additional configurable parameters][7] for the `--backup-location-config` flag.

(Optional) Specify [additional configurable parameters][8] for the `--snapshot-location-config` flag.

(Optional) [Customize the Velero installation][9] further to meet your needs.

## Server-Side Encryption with Customer-Provided Encryption Keys (SSE-C)

Velero supports using SSE-C encryption for S3 backups. This allows you to provide your own 32-byte encryption key that S3 will use to encrypt your backup data. There are two ways to provide the customer key:

### Option 1: Using a mounted file (customerKeyEncryptionFile)

1. Create a Kubernetes secret containing your 32-byte encryption key:

```bash
# Generate a 32-byte key (example)
openssl rand -out customer-key.txt 32

# Create the secret
kubectl create secret generic velero-sse-c-key \
  -n velero \
  --from-file=customer-key=customer-key.txt
```

2. Mount the secret in the Velero deployment:

```bash
kubectl patch deployment/velero -n velero --type='json' -p='[
  {
    "op": "add",
    "path": "/spec/template/spec/volumes/-",
    "value": {
      "name": "sse-c-key",
      "secret": {
        "secretName": "velero-sse-c-key"
      }
    }
  },
  {
    "op": "add",
    "path": "/spec/template/spec/containers/0/volumeMounts/-",
    "value": {
      "name": "sse-c-key",
      "mountPath": "/credentials/sse-c",
      "readOnly": true
    }
  }
]'
```

3. Configure the backup storage location to use the mounted key file:

```bash
velero backup-location create default \
  --provider aws \
  --bucket $BUCKET \
  --config region=$REGION,customerKeyEncryptionFile=/credentials/sse-c/customer-key
```

### Option 2: Using a Kubernetes secret directly (customerKeyEncryptionSecret)

This option allows Velero to read the encryption key directly from a Kubernetes secret without mounting it as a file.

1. Create a Kubernetes secret containing your 32-byte encryption key:

```bash
# Generate a 32-byte key (example)
openssl rand 32 | kubectl create secret generic velero-sse-c-key \
  -n velero \
  --from-file=customer-key=/dev/stdin
```

2. Configure the backup storage location to reference the secret:

```bash
velero backup-location create default \
  --provider aws \
  --bucket $BUCKET \
  --config region=$REGION,customerKeyEncryptionSecret=velero-sse-c-key/customer-key
```

The format for `customerKeyEncryptionSecret` is `secretName/key`, where:

- `secretName` is the name of the Kubernetes secret
- `key` is the key within the secret that contains the 32-byte encryption key

The secret must exist in the same namespace as Velero (determined by the `VELERO_NAMESPACE` environment variable).

### Important Notes about SSE-C

- The customer key must be exactly 32 bytes
- You cannot use SSE-C in combination with `kmsKeyId`
- You must specify either `customerKeyEncryptionFile` or `customerKeyEncryptionSecret`, not both
- Keep your encryption key secure - losing it means losing access to your backups
- The same key must be available during restore operations

For more complex installation needs, use either the Helm chart, or add `--dry-run -o yaml` options for generating the YAML representation for the installation.

## Create an additional Backup Storage Location

If you are using Velero v1.6.0 or later, you can create additional AWS [Backup Storage Locations][13] that use their own credentials.
These can also be created alongside Backup Storage Locations that use other providers.

### Limitations
It is not possible to use different credentials for additional Backup Storage Locations if you are pod based authentication such as [kube2iam][14].

### Prerequisites

* Velero 1.6.0 or later
* AWS plugin must be installed, either at install time, or by running `velero plugin add velero/velero-plugin-for-aws:plugin-version`, replace the `plugin-version` with the corresponding value

### Configure S3 bucket and credentials

To configure a new Backup Storage Location with its own credentials, it is necessary to follow the steps above to [create the bucket to use][15] and to [generate the credentials file][16] to interact with that bucket.
Once you have created the credentials file, create a [Kubernetes Secret][17] in the Velero namespace that contains these credentials:

```bash
kubectl create secret generic -n velero bsl-credentials --from-file=aws=</path/to/credentialsfile>
```

This will create a secret named `bsl-credentials` with a single key (`aws`) which contains the contents of your credentials file.
The name and key of this secret will be given to Velero when creating the Backup Storage Location, so it knows which secret data to use.

### Create Backup Storage Location

Once the bucket and credentials have been configured, these can be used to create the new Backup Storage Location:

```bash
velero backup-location create <bsl-name> \
  --provider aws \
  --bucket $BUCKET \
  --config region=$REGION \
  --credential=bsl-credentials=aws
```

The Backup Storage Location is ready to use when it has the phase `Available`.
You can check this with the following command:

```bash
velero backup-location get
```

To use this new Backup Storage Location when performing a backup, use the flag `--storage-location <bsl-name>` when running `velero backup create`.

## Migrating PVs across clusters

### Setting AWS_CLUSTER_NAME (Optional)

If you have multiple clusters and you want to support migration of resources between them, you can use `kubectl edit deploy/velero -n velero` to edit your deployment:

Add the environment variable `AWS_CLUSTER_NAME` under `spec.template.spec.env`, with the current cluster's name. When restoring backup, it will make Velero (and cluster it's running on) claim ownership of AWS volumes created from snapshots taken on different cluster.
The best way to get the current cluster's name is to either check it with used deployment tool or to read it directly from the EC2 instances tags.

The following listing shows how to get the cluster's nodes EC2 Tags. First, get the nodes external IDs (EC2 IDs):

```bash
kubectl get nodes -o jsonpath='{.items[*].spec.externalID}'
```

Copy one of the returned IDs `<ID>` and use it with the `aws` CLI tool to search for one of the following:

  * The `kubernetes.io/cluster/<AWS_CLUSTER_NAME>` tag of the value `owned`. The `<AWS_CLUSTER_NAME>` is then your cluster's name:

    ```bash
    aws ec2 describe-tags --filters "Name=resource-id,Values=<ID>" "Name=value,Values=owned"
    ```

  * If the first output returns nothing, then check for the legacy Tag `KubernetesCluster` of the value `<AWS_CLUSTER_NAME>`:

    ```bash
    aws ec2 describe-tags --filters "Name=resource-id,Values=<ID>" "Name=key,Values=KubernetesCluster"
    ```


[1]: #Create-S3-bucket
[2]: #Set-permissions-for-Velero
[3]: #Install-and-start-Velero
[4]: https://velero.io/docs/install-overview/
[5]: #Migrating-PVs-across-clusters
[6]: https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-welcome.html
[7]: backupstoragelocation.md
[8]: volumesnapshotlocation.md
[9]: https://velero.io/docs/customize-installation/
[10]: http://docs.aws.amazon.com/IAM/latest/UserGuide/introduction.html
[11]: https://velero.io/docs/faq/
[12]: #Create-an-additional-Backup-Storage-Location
[13]: https://velero.io/docs/latest/api-types/backupstoragelocation/
[14]: #option-2-set-permissions-using-kube2iam
[15]: #create-s3-bucket
[16]: #option-1-set-permissions-with-an-iam-user
[17]: https://kubernetes.io/docs/concepts/configuration/secret/
[101]: https://github.com/vmware-tanzu/velero-plugin-for-aws/workflows/Main%20CI/badge.svg
[102]: https://github.com/vmware-tanzu/velero-plugin-for-aws/actions?query=workflow%3A"Main+CI"
[103]: https://github.com/vmware-tanzu/velero/issues/new/choose
