# Velero plugins for AWS

## Overview

This repository contains these plugins to support running Velero on AWS:

- An object store plugin for persisting and retrieving backups on AWS S3. Content of backup is log files, warning/error files, restore logs.

- A volume snapshotter plugin for creating snapshots from volumes (during a backup) and volumes from snapshots (during a restore) on AWS EBS.

## Compatibility

Below is a listing of plugin versions and respective Velero versions that are compatible.

| Plugin Version  | Velero Version |
|-----------------|----------------|
| v1.0.x          | v1.2.0         |


## Setup

To set up Velero on AWS, you:

* [Create an S3 bucket][1]
* [Set permissions for Velero][2]
* [Install and start Velero][3]
* [Migrating PVs across clusters][5]

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

2. Attach policies to give `velero` the necessary permissions:

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

3. Attach policies to give `velero` the necessary permissions:

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
    --plugins velero/velero-plugin-for-aws:v1.0.1 \
    --bucket $BUCKET \
    --backup-location-config region=$REGION \
    --snapshot-location-config region=$REGION \
    --secret-file ./credentials-velero
```

**If using kube2iam**:

```bash
velero install \
    --provider aws \
    --plugins velero/velero-plugin-for-aws:v1.0.1 \
    --bucket $BUCKET \
    --backup-location-config region=$REGION \
    --snapshot-location-config region=$REGION \
    --pod-annotations iam.amazonaws.com/role=arn:aws:iam::<AWS_ACCOUNT_ID>:role/<VELERO_ROLE_NAME> \
    --no-secret
```

Additionally, you can specify `--use-restic` to enable restic support, and `--wait` to wait for the deployment to be ready.

(Optional) Specify [additional configurable parameters][7] for the `--backup-location-config` flag.

(Optional) Specify [additional configurable parameters][8] for the `--snapshot-location-config` flag.

(Optional) Specify [CPU and memory resource requests and limits][9] for the Velero/restic pods.

For more complex installation needs, use either the Helm chart, or add `--dry-run -o yaml` options for generating the YAML representation for the installation.

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
[4]: https://velero.io/docs/v1.2.0/basic-install/
[5]: #Migrating-PVs-across-clusters
[6]: https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-welcome.html
[7]: backupstoragelocation.md
[8]: volumesnapshotlocation.md
[9]: https://velero.io/docs/master/install-requirements
[10]: http://docs.aws.amazon.com/IAM/latest/UserGuide/introduction.html
[11]: https://velero.io/docs/master/faq/
