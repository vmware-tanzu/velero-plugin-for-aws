
English
Preferences 
Contact Us
Feedback
AWS Documentation

Get started
Service guides
Developer tools
AI resources

Sign In to the Console

Amazon Simple Storage Service
API Reference
Welcome

S3 API Reference

Actions

Amazon S3
AbortMultipartUpload
CompleteMultipartUpload
CopyObject
CreateBucket
CreateBucketMetadataConfiguration
CreateBucketMetadataTableConfiguration
CreateMultipartUpload
CreateSession
DeleteBucket
DeleteBucketAnalyticsConfiguration
DeleteBucketCors
DeleteBucketEncryption
DeleteBucketIntelligentTieringConfiguration
DeleteBucketInventoryConfiguration
DeleteBucketLifecycle
DeleteBucketMetadataConfiguration
DeleteBucketMetadataTableConfiguration
DeleteBucketMetricsConfiguration
DeleteBucketOwnershipControls
DeleteBucketPolicy
DeleteBucketReplication
DeleteBucketTagging
DeleteBucketWebsite
DeleteObject
DeleteObjects
DeleteObjectTagging
DeletePublicAccessBlock
GetBucketAccelerateConfiguration
GetBucketAcl
GetBucketAnalyticsConfiguration
GetBucketCors
GetBucketEncryption
GetBucketIntelligentTieringConfiguration
GetBucketInventoryConfiguration
GetBucketLifecycle
GetBucketLifecycleConfiguration
GetBucketLocation
GetBucketLogging
GetBucketMetadataConfiguration
GetBucketMetadataTableConfiguration
GetBucketMetricsConfiguration
GetBucketNotification
GetBucketNotificationConfiguration
GetBucketOwnershipControls
GetBucketPolicy
GetBucketPolicyStatus
GetBucketReplication
GetBucketRequestPayment
GetBucketTagging
GetBucketVersioning
GetBucketWebsite
GetObject
GetObjectAcl
GetObjectAttributes
GetObjectLegalHold
GetObjectLockConfiguration
GetObjectRetention
GetObjectTagging
GetObjectTorrent
GetPublicAccessBlock
HeadBucket
HeadObject
ListBucketAnalyticsConfigurations
ListBucketIntelligentTieringConfigurations
ListBucketInventoryConfigurations
ListBucketMetricsConfigurations
ListBuckets
ListDirectoryBuckets
ListMultipartUploads
ListObjects
ListObjectsV2
ListObjectVersions
ListParts
PutBucketAccelerateConfiguration
PutBucketAcl
PutBucketAnalyticsConfiguration
PutBucketCors
PutBucketEncryption
PutBucketIntelligentTieringConfiguration
PutBucketInventoryConfiguration
PutBucketLifecycle
PutBucketLifecycleConfiguration
PutBucketLogging
PutBucketMetricsConfiguration
PutBucketNotification
PutBucketNotificationConfiguration
PutBucketOwnershipControls
PutBucketPolicy
PutBucketReplication
PutBucketRequestPayment
PutBucketTagging
PutBucketVersioning
PutBucketWebsite
PutObject
PutObjectAcl
PutObjectLegalHold
PutObjectLockConfiguration
PutObjectRetention
PutObjectTagging
PutPublicAccessBlock
RenameObject
RestoreObject
SelectObjectContent
UpdateBucketMetadataInventoryTableConfiguration
UpdateBucketMetadataJournalTableConfiguration
UploadPart
UploadPartCopy
WriteGetObjectResponse

Amazon S3 Control

Amazon S3 on Outposts

Amazon S3 Tables

Amazon S3 Vectors

Data Types

Developing with Amazon S3

Code examples

Authenticating Requests (AWS Signature Version 4)

Browser-Based Uploads Using POST
Common request headers
Common response headers

Error responses
AWS Glossary
Resources
Document History

Appendix
Documentation
Amazon Simple Storage Service (S3)
API Reference
Documentation
Amazon Simple Storage Service (S3)
API Reference
GetObject
 PDF
Focus mode
Retrieves an object from Amazon S3.

In the GetObject request, specify the full key name for the object.

General purpose buckets - Both the virtual-hosted-style requests and the path-style requests are supported. For a virtual hosted-style request example, if you have the object photos/2006/February/sample.jpg, specify the object key name as /photos/2006/February/sample.jpg. For a path-style request example, if you have the object photos/2006/February/sample.jpg in the bucket named examplebucket, specify the object key name as /examplebucket/photos/2006/February/sample.jpg. For more information about request types, see HTTP Host Header Bucket Specification in the Amazon S3 User Guide.

Directory buckets - Only virtual-hosted-style requests are supported. For a virtual hosted-style request example, if you have the object photos/2006/February/sample.jpg in the bucket named amzn-s3-demo-bucket--usw2-az1--x-s3, specify the object key name as /photos/2006/February/sample.jpg. Also, when you make requests to this API operation, your requests are sent to the Zonal endpoint. These endpoints support virtual-hosted-style requests in the format https://bucket-name.s3express-zone-id.region-code.amazonaws.com/key-name . Path-style requests are not supported. For more information about endpoints in Availability Zones, see Regional and Zonal endpoints for directory buckets in Availability Zones in the Amazon S3 User Guide. For more information about endpoints in Local Zones, see Concepts for directory buckets in Local Zones in the Amazon S3 User Guide.

Permissions
General purpose bucket permissions - You must have the required permissions in a policy. To use GetObject, you must have the READ access to the object (or version). If you grant READ access to the anonymous user, the GetObject operation returns the object without using an authorization header. For more information, see Specifying permissions in a policy in the Amazon S3 User Guide.

If you include a versionId in your request header, you must have the s3:GetObjectVersion permission to access a specific version of an object. The s3:GetObject permission is not required in this scenario.

If you request the current version of an object without a specific versionId in the request header, only the s3:GetObject permission is required. The s3:GetObjectVersion permission is not required in this scenario.

If the object that you request doesn’t exist, the error that Amazon S3 returns depends on whether you also have the s3:ListBucket permission.

If you have the s3:ListBucket permission on the bucket, Amazon S3 returns an HTTP status code 404 Not Found error.

If you don’t have the s3:ListBucket permission, Amazon S3 returns an HTTP status code 403 Access Denied error.

Directory bucket permissions - To grant access to this API operation on a directory bucket, we recommend that you use the CreateSession API operation for session-based authorization. Specifically, you grant the s3express:CreateSession permission to the directory bucket in a bucket policy or an IAM identity-based policy. Then, you make the CreateSession API call on the bucket to obtain a session token. With the session token in your request header, you can make API requests to this operation. After the session token expires, you make another CreateSession API call to generate a new session token for use. AWS CLI or SDKs create session and refresh the session token automatically to avoid service interruptions when a session expires. For more information about authorization, see CreateSession.

If the object is encrypted using SSE-KMS, you must also have the kms:GenerateDataKey and kms:Decrypt permissions in IAM identity-based policies and AWS KMS key policies for the AWS KMS key.

Storage classes
If the object you are retrieving is stored in the S3 Glacier Flexible Retrieval storage class, the S3 Glacier Deep Archive storage class, the S3 Intelligent-Tiering Archive Access tier, or the S3 Intelligent-Tiering Deep Archive Access tier, before you can retrieve the object you must first restore a copy using RestoreObject. Otherwise, this operation returns an InvalidObjectState error. For information about restoring archived objects, see Restoring Archived Objects in the Amazon S3 User Guide.

Directory buckets - Directory buckets only support EXPRESS_ONEZONE (the S3 Express One Zone storage class) in Availability Zones and ONEZONE_IA (the S3 One Zone-Infrequent Access storage class) in Dedicated Local Zones. Unsupported storage class values won't write a destination object and will respond with the HTTP status code 400 Bad Request.

Encryption
Encryption request headers, like x-amz-server-side-encryption, should not be sent for the GetObject requests, if your object uses server-side encryption with Amazon S3 managed encryption keys (SSE-S3), server-side encryption with AWS Key Management Service (AWS KMS) keys (SSE-KMS), or dual-layer server-side encryption with AWS KMS keys (DSSE-KMS). If you include the header in your GetObject requests for the object that uses these types of keys, you’ll get an HTTP 400 Bad Request error.

Directory buckets - For directory buckets, there are only two supported options for server-side encryption: SSE-S3 and SSE-KMS. SSE-C isn't supported. For more information, see Protecting data with server-side encryption in the Amazon S3 User Guide.

Overriding response header values through the request
There are times when you want to override certain response header values of a GetObject response. For example, you might override the Content-Disposition response header value through your GetObject request.

You can override values for a set of response headers. These modified response header values are included only in a successful response, that is, when the HTTP status code 200 OK is returned. The headers you can override using the following query parameters in the request are a subset of the headers that Amazon S3 accepts when you create an object.

The response headers that you can override for the GetObject response are Cache-Control, Content-Disposition, Content-Encoding, Content-Language, Content-Type, and Expires.

To override values for a set of response headers in the GetObject response, you can use the following query parameters in the request.

response-cache-control

response-content-disposition

response-content-encoding

response-content-language

response-content-type

response-expires

Note
When you use these parameters, you must sign the request by using either an Authorization header or a presigned URL. These parameters cannot be used with an unsigned (anonymous) request.

HTTP Host header syntax
Directory buckets - The HTTP Host header syntax is Bucket-name.s3express-zone-id.region-code.amazonaws.com.

The following operations are related to GetObject:

ListBuckets

GetObjectAcl

Important
You must URL encode any signed header values that contain spaces. For example, if your header value is my file.txt, containing two spaces after my, you must URL encode this value to my%20%20file.txt.

Request Syntax

GET /Key+?partNumber=PartNumber&response-cache-control=ResponseCacheControl&response-content-disposition=ResponseContentDisposition&response-content-encoding=ResponseContentEncoding&response-content-language=ResponseContentLanguage&response-content-type=ResponseContentType&response-expires=ResponseExpires&versionId=VersionId HTTP/1.1
Host: Bucket.s3.amazonaws.com
If-Match: IfMatch
If-Modified-Since: IfModifiedSince
If-None-Match: IfNoneMatch
If-Unmodified-Since: IfUnmodifiedSince
Range: Range
x-amz-server-side-encryption-customer-algorithm: SSECustomerAlgorithm
x-amz-server-side-encryption-customer-key: SSECustomerKey
x-amz-server-side-encryption-customer-key-MD5: SSECustomerKeyMD5
x-amz-request-payer: RequestPayer
x-amz-expected-bucket-owner: ExpectedBucketOwner
x-amz-checksum-mode: ChecksumMode
URI Request Parameters

The request uses the following URI parameters.

Bucket
The bucket name containing the object.

Directory buckets - When you use this operation with a directory bucket, you must use virtual-hosted-style requests in the format Bucket-name.s3express-zone-id.region-code.amazonaws.com. Path-style requests are not supported. Directory bucket names must be unique in the chosen Zone (Availability Zone or Local Zone). Bucket names must follow the format bucket-base-name--zone-id--x-s3 (for example, amzn-s3-demo-bucket--usw2-az1--x-s3). For information about bucket naming restrictions, see Directory bucket naming rules in the Amazon S3 User Guide.

Access points - When you use this action with an access point for general purpose buckets, you must provide the alias of the access point in place of the bucket name or specify the access point ARN. When you use this action with an access point for directory buckets, you must provide the access point name in place of the bucket name. When using the access point ARN, you must direct requests to the access point hostname. The access point hostname takes the form AccessPointName-AccountId.s3-accesspoint.Region.amazonaws.com. When using this action with an access point through the AWS SDKs, you provide the access point ARN in place of the bucket name. For more information about access point ARNs, see Using access points in the Amazon S3 User Guide.

Object Lambda access points - When you use this action with an Object Lambda access point, you must direct requests to the Object Lambda access point hostname. The Object Lambda access point hostname takes the form AccessPointName-AccountId.s3-object-lambda.Region.amazonaws.com.

Note
Object Lambda access points are not supported by directory buckets.

S3 on Outposts - When you use this action with S3 on Outposts, you must direct requests to the S3 on Outposts hostname. The S3 on Outposts hostname takes the form AccessPointName-AccountId.outpostID.s3-outposts.Region.amazonaws.com. When you use this action with S3 on Outposts, the destination bucket must be the Outposts access point ARN or the access point alias. For more information about S3 on Outposts, see What is S3 on Outposts? in the Amazon S3 User Guide.

Required: Yes

If-Match
Return the object only if its entity tag (ETag) is the same as the one specified in this header; otherwise, return a 412 Precondition Failed error.

If both of the If-Match and If-Unmodified-Since headers are present in the request as follows: If-Match condition evaluates to true, and; If-Unmodified-Since condition evaluates to false; then, S3 returns 200 OK and the data requested.

For more information about conditional requests, see RFC 7232.

If-Modified-Since
Return the object only if it has been modified since the specified time; otherwise, return a 304 Not Modified error.

If both of the If-None-Match and If-Modified-Since headers are present in the request as follows: If-None-Match condition evaluates to false, and; If-Modified-Since condition evaluates to true; then, S3 returns 304 Not Modified status code.

For more information about conditional requests, see RFC 7232.

If-None-Match
Return the object only if its entity tag (ETag) is different from the one specified in this header; otherwise, return a 304 Not Modified error.

If both of the If-None-Match and If-Modified-Since headers are present in the request as follows: If-None-Match condition evaluates to false, and; If-Modified-Since condition evaluates to true; then, S3 returns 304 Not Modified HTTP status code.

For more information about conditional requests, see RFC 7232.

If-Unmodified-Since
Return the object only if it has not been modified since the specified time; otherwise, return a 412 Precondition Failed error.

If both of the If-Match and If-Unmodified-Since headers are present in the request as follows: If-Match condition evaluates to true, and; If-Unmodified-Since condition evaluates to false; then, S3 returns 200 OK and the data requested.

For more information about conditional requests, see RFC 7232.

Key
Key of the object to get.

Length Constraints: Minimum length of 1.

Required: Yes

partNumber
Part number of the object being read. This is a positive integer between 1 and 10,000. Effectively performs a 'ranged' GET request for the part specified. Useful for downloading just a part of an object.

Range
Downloads the specified byte range of an object. For more information about the HTTP Range header, see https://www.rfc-editor.org/rfc/rfc9110.html#name-range.

Note
Amazon S3 doesn't support retrieving multiple ranges of data per GET request.

response-cache-control
Sets the Cache-Control header of the response.

response-content-disposition
Sets the Content-Disposition header of the response.

response-content-encoding
Sets the Content-Encoding header of the response.

response-content-language
Sets the Content-Language header of the response.

response-content-type
Sets the Content-Type header of the response.

response-expires
Sets the Expires header of the response.

versionId
Version ID used to reference a specific version of the object.

By default, the GetObject operation returns the current version of an object. To return a different version, use the versionId subresource.

Note
If you include a versionId in your request header, you must have the s3:GetObjectVersion permission to access a specific version of an object. The s3:GetObject permission is not required in this scenario.

If you request the current version of an object without a specific versionId in the request header, only the s3:GetObject permission is required. The s3:GetObjectVersion permission is not required in this scenario.

Directory buckets - S3 Versioning isn't enabled and supported for directory buckets. For this API operation, only the null value of the version ID is supported by directory buckets. You can only specify null to the versionId query parameter in the request.

For more information about versioning, see PutBucketVersioning.

x-amz-checksum-mode
To retrieve the checksum, this mode must be enabled.

Valid Values: ENABLED

x-amz-expected-bucket-owner
The account ID of the expected bucket owner. If the account ID that you provide does not match the actual owner of the bucket, the request fails with the HTTP status code 403 Forbidden (access denied).

x-amz-request-payer
Confirms that the requester knows that they will be charged for the request. Bucket owners need not specify this parameter in their requests. If either the source or destination S3 bucket has Requester Pays enabled, the requester will pay for corresponding charges to copy the object. For information about downloading objects from Requester Pays buckets, see Downloading Objects in Requester Pays Buckets in the Amazon S3 User Guide.

Note
This functionality is not supported for directory buckets.

Valid Values: requester

x-amz-server-side-encryption-customer-algorithm
Specifies the algorithm to use when decrypting the object (for example, AES256).

If you encrypt an object by using server-side encryption with customer-provided encryption keys (SSE-C) when you store the object in Amazon S3, then when you GET the object, you must use the following headers:

x-amz-server-side-encryption-customer-algorithm

x-amz-server-side-encryption-customer-key

x-amz-server-side-encryption-customer-key-MD5

For more information about SSE-C, see Server-Side Encryption (Using Customer-Provided Encryption Keys) in the Amazon S3 User Guide.

Note
This functionality is not supported for directory buckets.

x-amz-server-side-encryption-customer-key
Specifies the customer-provided encryption key that you originally provided for Amazon S3 to encrypt the data before storing it. This value is used to decrypt the object when recovering it and must match the one used when storing the data. The key must be appropriate for use with the algorithm specified in the x-amz-server-side-encryption-customer-algorithm header.

If you encrypt an object by using server-side encryption with customer-provided encryption keys (SSE-C) when you store the object in Amazon S3, then when you GET the object, you must use the following headers:

x-amz-server-side-encryption-customer-algorithm

x-amz-server-side-encryption-customer-key

x-amz-server-side-encryption-customer-key-MD5

For more information about SSE-C, see Server-Side Encryption (Using Customer-Provided Encryption Keys) in the Amazon S3 User Guide.

Note
This functionality is not supported for directory buckets.

x-amz-server-side-encryption-customer-key-MD5
Specifies the 128-bit MD5 digest of the customer-provided encryption key according to RFC 1321. Amazon S3 uses this header for a message integrity check to ensure that the encryption key was transmitted without error.

If you encrypt an object by using server-side encryption with customer-provided encryption keys (SSE-C) when you store the object in Amazon S3, then when you GET the object, you must use the following headers:

x-amz-server-side-encryption-customer-algorithm

x-amz-server-side-encryption-customer-key

x-amz-server-side-encryption-customer-key-MD5

For more information about SSE-C, see Server-Side Encryption (Using Customer-Provided Encryption Keys) in the Amazon S3 User Guide.

Note
This functionality is not supported for directory buckets.

Request Body

The request does not have a request body.

Response Syntax

HTTP/1.1 200
x-amz-delete-marker: DeleteMarker
accept-ranges: AcceptRanges
x-amz-expiration: Expiration
x-amz-restore: Restore
Last-Modified: LastModified
Content-Length: ContentLength
ETag: ETag
x-amz-checksum-crc32: ChecksumCRC32
x-amz-checksum-crc32c: ChecksumCRC32C
x-amz-checksum-crc64nvme: ChecksumCRC64NVME
x-amz-checksum-sha1: ChecksumSHA1
x-amz-checksum-sha256: ChecksumSHA256
x-amz-checksum-type: ChecksumType
x-amz-missing-meta: MissingMeta
x-amz-version-id: VersionId
Cache-Control: CacheControl
Content-Disposition: ContentDisposition
Content-Encoding: ContentEncoding
Content-Language: ContentLanguage
Content-Range: ContentRange
Content-Type: ContentType
Expires: Expires
x-amz-website-redirect-location: WebsiteRedirectLocation
x-amz-server-side-encryption: ServerSideEncryption
x-amz-server-side-encryption-customer-algorithm: SSECustomerAlgorithm
x-amz-server-side-encryption-customer-key-MD5: SSECustomerKeyMD5
x-amz-server-side-encryption-aws-kms-key-id: SSEKMSKeyId
x-amz-server-side-encryption-bucket-key-enabled: BucketKeyEnabled
x-amz-storage-class: StorageClass
x-amz-request-charged: RequestCharged
x-amz-replication-status: ReplicationStatus
x-amz-mp-parts-count: PartsCount
x-amz-tagging-count: TagCount
x-amz-object-lock-mode: ObjectLockMode
x-amz-object-lock-retain-until-date: ObjectLockRetainUntilDate
x-amz-object-lock-legal-hold: ObjectLockLegalHoldStatus

Body
Response Elements

If the action is successful, the service sends back an HTTP 200 response.

The response returns the following HTTP headers.

accept-ranges
Indicates that a range of bytes was specified in the request.

Cache-Control
Specifies caching behavior along the request/reply chain.

Content-Disposition
Specifies presentational information for the object.

Content-Encoding
Indicates what content encodings have been applied to the object and thus what decoding mechanisms must be applied to obtain the media-type referenced by the Content-Type header field.

Content-Language
The language the content is in.

Content-Length
Size of the body in bytes.

Content-Range
The portion of the object returned in the response.

Content-Type
A standard MIME type describing the format of the object data.

ETag
An entity tag (ETag) is an opaque identifier assigned by a web server to a specific version of a resource found at a URL.

Expires
The date and time at which the object is no longer cacheable.

Last-Modified
Date and time when the object was last modified.

General purpose buckets - When you specify a versionId of the object in your request, if the specified version in the request is a delete marker, the response returns a 405 Method Not Allowed error and the Last-Modified: timestamp response header.

x-amz-checksum-crc32
The Base64 encoded, 32-bit CRC32 checksum of the object. This checksum is only present if the object was uploaded with the object. For more information, see Checking object integrity in the Amazon S3 User Guide.

x-amz-checksum-crc32c
The Base64 encoded, 32-bit CRC32C checksum of the object. This checksum is only present if the checksum was uploaded with the object. For more information, see Checking object integrity in the Amazon S3 User Guide.

x-amz-checksum-crc64nvme
The Base64 encoded, 64-bit CRC64NVME checksum of the object. For more information, see Checking object integrity in the Amazon S3 User Guide.

x-amz-checksum-sha1
The Base64 encoded, 160-bit SHA1 digest of the object. This checksum is only present if the checksum was uploaded with the object. For more information, see Checking object integrity in the Amazon S3 User Guide.

x-amz-checksum-sha256
The Base64 encoded, 256-bit SHA256 digest of the object. This checksum is only present if the checksum was uploaded with the object. For more information, see Checking object integrity in the Amazon S3 User Guide.

x-amz-checksum-type
The checksum type, which determines how part-level checksums are combined to create an object-level checksum for multipart objects. You can use this header response to verify that the checksum type that is received is the same checksum type that was specified in the CreateMultipartUpload request. For more information, see Checking object integrity in the Amazon S3 User Guide.

Valid Values: COMPOSITE | FULL_OBJECT

x-amz-delete-marker
Indicates whether the object retrieved was (true) or was not (false) a Delete Marker. If false, this response header does not appear in the response.

Note
If the current version of the object is a delete marker, Amazon S3 behaves as if the object was deleted and includes x-amz-delete-marker: true in the response.

If the specified version in the request is a delete marker, the response returns a 405 Method Not Allowed error and the Last-Modified: timestamp response header.

x-amz-expiration
If the object expiration is configured (see PutBucketLifecycleConfiguration), the response includes this header. It includes the expiry-date and rule-id key-value pairs providing object expiration information. The value of the rule-id is URL-encoded.

Note
Object expiration information is not returned in directory buckets and this header returns the value "NotImplemented" in all responses for directory buckets.

x-amz-missing-meta
This is set to the number of metadata entries not returned in the headers that are prefixed with x-amz-meta-. This can happen if you create metadata using an API like SOAP that supports more flexible metadata than the REST API. For example, using SOAP, you can create metadata whose values are not legal HTTP headers.

Note
This functionality is not supported for directory buckets.

x-amz-mp-parts-count
The count of parts this object has. This value is only returned if you specify partNumber in your request and the object was uploaded as a multipart upload.

x-amz-object-lock-legal-hold
Indicates whether this object has an active legal hold. This field is only returned if you have permission to view an object's legal hold status.

Note
This functionality is not supported for directory buckets.

Valid Values: ON | OFF

x-amz-object-lock-mode
The Object Lock mode that's currently in place for this object.

Note
This functionality is not supported for directory buckets.

Valid Values: GOVERNANCE | COMPLIANCE

x-amz-object-lock-retain-until-date
The date and time when this object's Object Lock will expire.

Note
This functionality is not supported for directory buckets.

x-amz-replication-status
Amazon S3 can return this if your request involves a bucket that is either a source or destination in a replication rule.

Note
This functionality is not supported for directory buckets.

Valid Values: COMPLETE | PENDING | FAILED | REPLICA | COMPLETED

x-amz-request-charged
If present, indicates that the requester was successfully charged for the request. For more information, see Using Requester Pays buckets for storage transfers and usage in the Amazon Simple Storage Service user guide.

Note
This functionality is not supported for directory buckets.

Valid Values: requester

x-amz-restore
Provides information about object restoration action and expiration time of the restored object copy.

Note
This functionality is not supported for directory buckets. Directory buckets only support EXPRESS_ONEZONE (the S3 Express One Zone storage class) in Availability Zones and ONEZONE_IA (the S3 One Zone-Infrequent Access storage class) in Dedicated Local Zones.

x-amz-server-side-encryption
The server-side encryption algorithm used when you store this object in Amazon S3 or Amazon FSx.

Note
When accessing data stored in Amazon FSx file systems using S3 access points, the only valid server side encryption option is aws:fsx.

Valid Values: AES256 | aws:fsx | aws:kms | aws:kms:dsse

x-amz-server-side-encryption-aws-kms-key-id
If present, indicates the ID of the KMS key that was used for object encryption.

x-amz-server-side-encryption-bucket-key-enabled
Indicates whether the object uses an S3 Bucket Key for server-side encryption with AWS Key Management Service (AWS KMS) keys (SSE-KMS).

x-amz-server-side-encryption-customer-algorithm
If server-side encryption with a customer-provided encryption key was requested, the response will include this header to confirm the encryption algorithm that's used.

Note
This functionality is not supported for directory buckets.

x-amz-server-side-encryption-customer-key-MD5
If server-side encryption with a customer-provided encryption key was requested, the response will include this header to provide the round-trip message integrity verification of the customer-provided encryption key.

Note
This functionality is not supported for directory buckets.

x-amz-storage-class
Provides storage class information of the object. Amazon S3 returns this header for all objects except for S3 Standard storage class objects.

Note
Directory buckets - Directory buckets only support EXPRESS_ONEZONE (the S3 Express One Zone storage class) in Availability Zones and ONEZONE_IA (the S3 One Zone-Infrequent Access storage class) in Dedicated Local Zones.

Valid Values: STANDARD | REDUCED_REDUNDANCY | STANDARD_IA | ONEZONE_IA | INTELLIGENT_TIERING | GLACIER | DEEP_ARCHIVE | OUTPOSTS | GLACIER_IR | SNOW | EXPRESS_ONEZONE | FSX_OPENZFS

x-amz-tagging-count
The number of tags, if any, on the object, when you have the relevant permission to read object tags.

You can use GetObjectTagging to retrieve the tag set associated with an object.

Note
This functionality is not supported for directory buckets.

x-amz-version-id
Version ID of the object.

Note
This functionality is not supported for directory buckets.

x-amz-website-redirect-location
If the bucket is configured as a website, redirects requests for this object to another object in the same bucket or to an external URL. Amazon S3 stores the value of this header in the object metadata.

Note
This functionality is not supported for directory buckets.

The following data is returned in binary format by the service.

Body
Errors

InvalidObjectState
Object is archived and inaccessible until restored.

If the object you are retrieving is stored in the S3 Glacier Flexible Retrieval storage class, the S3 Glacier Deep Archive storage class, the S3 Intelligent-Tiering Archive Access tier, or the S3 Intelligent-Tiering Deep Archive Access tier, before you can retrieve the object you must first restore a copy using RestoreObject. Otherwise, this operation returns an InvalidObjectState error. For information about restoring archived objects, see Restoring Archived Objects in the Amazon S3 User Guide.

HTTP Status Code: 403

NoSuchKey
The specified key does not exist.

HTTP Status Code: 404

Examples

Sample Request for general purpose buckets
The following request returns the object my-image.jpg.



            GET /my-image.jpg HTTP/1.1
            Host: amzn-s3-demo-bucket.s3.<Region>.amazonaws.com
            Date: Mon, 3 Oct 2016 22:32:00 GMT
            Authorization: authorization string
         
Sample Response for general purpose buckets
This example illustrates one usage of GetObject.



            HTTP/1.1 200 OK
            x-amz-id-2: eftixk72aD6Ap51TnqcoF8eFidJG9Z/2mkiDFu8yU9AS1ed4OpIszj7UDNEHGran
            x-amz-request-id: 318BC8BC148832E5
            Date: Mon, 3 Oct 2016 22:32:00 GMT
            Last-Modified: Wed, 12 Oct 2009 17:50:00 GMT
            ETag: "fba9dede5f27731c9771645a39863328"
            Content-Length: 434234

           [434234 bytes of object data]
         
Sample Response for general purpose buckets: Object with associated tags
If the object had tags associated with it, Amazon S3 returns the x-amz-tagging-count header with tag count.



            HTTP/1.1 200 OK
            x-amz-id-2: eftixk72aD6Ap51TnqcoF8eFidJG9Z/2mkiDFu8yU9AS1ed4OpIszj7UDNEHGran
            x-amz-request-id: 318BC8BC148832E5
            Date: Mon, 3 Oct 2016 22:32:00 GMT
            Last-Modified: Wed, 12 Oct 2009 17:50:00 GMT
            ETag: "fba9dede5f27731c9771645a39863328"
            Content-Length: 434234
            x-amz-tagging-count: 2

           [434234 bytes of object data]
         
Sample Response for general purpose buckets: Object with an expiration
If the object had expiration set using lifecycle configuration, you get the following response with the x-amz-expiration header.



            HTTP/1.1 200 OK
            x-amz-id-2: eftixk72aD6Ap51TnqcoF8eFidJG9Z/2mkiDFu8yU9AS1ed4OpIszj7UDNEHGran
            x-amz-request-id: 318BC8BC148832E5
            Date: Wed, 28 Oct 2009 22:32:00 GMT
            Last-Modified: Wed, 12 Oct 2009 17:50:00 GMT
            x-amz-expiration: expiry-date="Fri, 23 Dec 2012 00:00:00 GMT", rule-id="picture-deletion-rule"
            ETag: "fba9dede5f27731c9771645a39863328"
            Content-Length: 434234
            Content-Type: text/plain

            [434234 bytes of object data]
         
Sample Response for general purpose buckets: If an object is archived in the S3 Glacier Flexible Retrieval or S3 Glacier Deep Archive storage classes
If the object you are retrieving is stored in the S3 Glacier Flexible Retrieval or S3 Glacier Deep Archive storage classes, you must first restore a copy using RestoreObject. Otherwise, this action returns an InvalidObjectState error.



            HTTP/1.1 403 Forbidden
            x-amz-request-id: CD4BD8A1310A11B3
            x-amz-id-2: m9RDbQU0+RRBTjOUN1ChQ1eqMUnr9dv8b+KP6I2gHfRJZSTSrMCoRP8RtPRzX9mb
            Content-Type: application/xml
            Date: Mon, 12 Nov 2012 23:53:21 GMT
            Server: Amazon S3
            Content-Length: 231

            <Error>
              <Code>InvalidObjectState</Code>
              <Message>The action is not valid for the object's storage class</Message>
              <RequestId>9FEFFF118E15B86F</RequestId>
              <HostId>WVQ5kzhiT+oiUfDCOiOYv8W4Tk9eNcxWi/MK+hTS/av34Xy4rBU3zsavf0aaaaa</HostId>
            </Error>
         
Sample Response for general purpose buckets: If an object is archived with the S3 Intelligent-Tiering Archive or S3 Intelligent-Tiering Deep Archive tiers
If the object you are retrieving is stored in the S3 Intelligent-Tiering Archive or S3 Intelligent-Tiering Deep Archive tiers, you must first restore a copy using RestoreObject. Otherwise, this action returns an InvalidObjectState error. When restoring from Archive Access or Deep Archive Access tiers, the response will include StorageClass and AccessTier elements. Access tier valid values are ARCHIVE_ACCESS and DEEP_ARCHIVE_ACCESS. There is no syntax change if there is an ongoing restore.



            HTTP/1.1 403 Forbidden
            x-amz-request-id: CB6AW8C4332B23B7
            x-amz-id-2: n3RRfT90+PJDUhut3nhGW2ehfhfNU5f55c+a2ceCC36ab7c7fe3a71Q273b9Q45b1R5
            Content-Type: application/xml
            Date: Mon, 12 Nov 2012 23:53:21 GMT
            Server: Amazon S3
            Content-Length: 231

            <Error>
              <Code>InvalidObjectState</Code>
              <Message>The action is not valid for the object's access tier</Message>
              <StorageClass>INTELLIGENT_TIERING</StorageClass>
              <AccessTier>ARCHIVE_ACCESS</AccessTier>
              <RequestId>9FEFFF118E15B86F</RequestId>
              <HostId>WVQ5kzhiT+oiUfDCOiOYv8W4Tk9eNcxWi/MK+hTS/av34Xy4rBU3zsavf0aaaaa</HostId>
            </Error>
            
Sample Response for general purpose buckets: If the Latest Object Is a Delete Marker
Notice that the delete marker returns a 404 Not Found error.



            HTTP/1.1 404 Not Found
            x-amz-request-id: 318BC8BC148832E5
            x-amz-id-2: eftixk72aD6Ap51Tnqzj7UDNEHGran
            x-amz-version-id: 3GL4kqtJlcpXroDTDm3vjVBH40Nr8X8g
            x-amz-delete-marker:  true
            Date: Wed, 28 Oct 2009 22:32:00 GMT
            Content-Type: text/plain
            Connection: close
            Server: AmazonS3
         
Sample Request for general purpose buckets: Getting a specified version of an object
The following request returns the specified version of an object.



            GET /myObject?versionId=3/L4kqtJlcpXroDTDmpUMLUo HTTP/1.1
            Host: bucket.s3.<Region>.amazonaws.com
            Date: Wed, 28 Oct 2009 22:32:00 GMT
            Authorization: authorization string
         
Sample Response for general purpose buckets: GET a versioned object
This example illustrates one usage of GetObject.



            HTTP/1.1 200 OK
            x-amz-id-2: eftixk72aD6Ap54OpIszj7UDNEHGran
            x-amz-request-id: 318BC8BC148832E5
            Date: Wed, 28 Oct 2009 22:32:00 GMT
            Last-Modified: Sun, 1 Jan 2006 12:00:00 GMT
            x-amz-version-id: 3/L4kqtJlcpXroDTDmJ+rmSpXd3QBpUMLUo
            ETag: "fba9dede5f27731c9771645a39863328"
            Content-Length: 434234
            Content-Type: text/plain
            Connection: close
            Server: AmazonS3
            [434234 bytes of object data]
         
Sample Request for general purpose buckets: Parameters altering response header values
The following request specifies all the query string parameters in a GET request overriding the response header values.



            GET /Junk3.txt?response-cache-control=No-cache&response-content-disposition=attachment%3B%20filename%3Dtesting.txt&response-content-encoding=x-gzip&response-content-language=mi%2C%20en&response-expires=Thu%2C%2001%20Dec%201994%2016:00:00%20GMT HTTP/1.1
            x-amz-date: Sun, 19 Dec 2010 01:53:44 GMT
            Accept: */*
            Authorization: AWS AKIAIOSFODNN7EXAMPLE:aaStE6nKnw8ihhiIdReoXYlMamW=
         
Sample Response for general purpose buckets: With overridden response header values
The following request specifies all the query string parameters in a GET request overriding the response header values.



            HTTP/1.1 200 OK
            x-amz-id-2: SIidWAK3hK+Il3/Qqiu1ZKEuegzLAAspwsgwnwygb9GgFseeFHL5CII8NXSrfWW2
            x-amz-request-id: 881B1CBD9DF17WA1
            Date: Sun, 19 Dec 2010 01:54:01 GMT
            x-amz-meta-param1: value 1
            x-amz-meta-param2: value 2
            Cache-Control: No-cache
            Content-Language: mi, en
            Expires: Thu, 01 Dec 1994 16:00:00 GMT
            Content-Disposition: attachment; filename=testing.txt
            Content-Encoding: x-gzip
            Last-Modified: Fri, 17 Dec 2010 18:10:41 GMT
            ETag: "0332bee1a7bf845f176c5c0d1ae7cf07"
            Accept-Ranges: bytes
            Content-Type: text/plain
            Content-Length: 22
            Server: AmazonS3

            [object data not shown]
         
Sample Request for general purpose buckets: Range header
The following request specifies the HTTP Range header to retrieve the first 10 bytes of an object. For more information about the HTTP Range header, see https://www.rfc-editor.org/rfc/rfc9110.html#name-range.

Note
Amazon S3 doesn't support retrieving multiple ranges of data per GET request.



            GET /example-object HTTP/1.1
            Host: amzn-s3-demo-bucket.s3.<Region>.amazonaws.com
            x-amz-date: Fri, 28 Jan 2011 21:32:02 GMT
            Range: bytes=0-9
            Authorization: AWS AKIAIOSFODNN7EXAMPLE:Yxg83MZaEgh3OZ3l0rLo5RTX11o=
            Sample Response with Specified Range of the Object Bytes
         
Sample Response for general purpose buckets
In the following sample response, note that the header values are set to the values specified in the true request.



            HTTP/1.1 206 Partial Content
            x-amz-id-2: MzRISOwyjmnupCzjI1WC06l5TTAzm7/JypPGXLh0OVFGcJaaO3KW/hRAqKOpIEEp
            x-amz-request-id: 47622117804B3E11
            Date: Fri, 28 Jan 2011 21:32:09 GMT
            x-amz-meta-title: the title
            Last-Modified: Fri, 28 Jan 2011 20:10:32 GMT
            ETag: "b2419b1e3fd45d596ee22bdf62aaaa2f"
            Accept-Ranges: bytes
            Content-Range: bytes 0-9/443
            Content-Type: text/plain
            Content-Length: 10
            Server: AmazonS3

           [10 bytes of object data]
         
Sample Request for general purpose buckets: Get an object stored using server-side encryption with customer-provided encryption keys
If an object is stored in Amazon S3 using server-side encryption with customer-provided encryption keys, Amazon S3 needs encryption information so that it can decrypt the object before sending it to you in response to a GET request. You provide the encryption information in your GET request using the relevant headers, as shown in the following example request.



            GET /example-object HTTP/1.1
            Host: amzn-s3-demo-bucket.s3.<Region>.amazonaws.com	

            Accept: */*
            Authorization:authorization string   
            Date: Wed, 28 May 2014 19:24:44 +0000   
            x-amz-server-side-encryption-customer-key:g0lCfA3Dv40jZz5SQJ1ZukLRFqtI5WorC/8SEKEXAMPLE   
            x-amz-server-side-encryption-customer-key-MD5:ZjQrne1X/iTcskbY2m3example  
            x-amz-server-side-encryption-customer-algorithm:AES256
         
Sample Response for general purpose buckets
The following sample response shows some of the response headers Amazon S3 returns. Note that it includes the encryption information in the response.



            HTTP/1.1 200 OK
            x-amz-id-2: ka5jRm8X3N12ZiY29Z989zg2tNSJPMcK+to7jNjxImXBbyChqc6tLAv+sau7Vjzh
            x-amz-request-id: 195157E3E073D3F9   
            Date: Wed, 28 May 2014 19:24:45 GMT   
            Last-Modified: Wed, 28 May 2014 19:21:01 GMT   
            ETag: "c12022c9a3c6d3a28d29d90933a2b096"   
            x-amz-server-side-encryption-customer-algorithm: AES256   
            x-amz-server-side-encryption-customer-key-MD5: ZjQrne1X/iTcskbY2m3example    
         
See Also

For more information about using this API in one of the language-specific AWS SDKs, see the following:

AWS Command Line Interface

AWS SDK for .NET

AWS SDK for C++

AWS SDK for Go v2

AWS SDK for Java V2

AWS SDK for JavaScript V3

AWS SDK for Kotlin

AWS SDK for PHP V3

AWS SDK for Python

AWS SDK for Ruby V3

Discover highly rated pages Abstracts generated by AI

1
2
3
4

AmazonS3 › userguide
What is Amazon S3?
Amazon S3 offers object storage service with scalability, availability, security, and performance. Manage storage classes, lifecycle policies, access permissions, data transformations, usage metrics, and query tabular data.
October 15, 2025
AmazonS3 › userguide
General purpose bucket naming rules
Bucket naming rules include length, valid characters, formatting, uniqueness. Avoid periods, choose relevant names, include GUIDs. Create buckets with GUIDs using AWS CLI, SDK.
October 15, 2025
AmazonS3 › userguide
Hosting a static website using Amazon S3
Enabling website hosting on Amazon S3 allows hosting static websites with static content and client-side scripts. Configure index document, custom error document, permissions, logging, redirects, and cross-origin resource sharing.
October 15, 2025

On this page
Request Syntax
URI Request Parameters
Request Body
Response Syntax
Response Elements
Errors
Examples
See Also
Recommended tasks
Learn about

Understand how to retrieve object access control list
Did this page help you?
Yes
No
Provide feedback

Next topic:GetObjectAcl
Previous topic:GetBucketWebsite
Need help?
Try AWS re:Post 
PrivacySite termsCookie preferences© 2025, Amazon Web Services, Inc. or its affiliates. All rights reserved.
