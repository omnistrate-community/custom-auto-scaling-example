# Docker Compose Service Specification

Omnistrate extends the Docker Compose specification, which allows custom extensions using the syntax `x-`, and supports a subset of the standard specifications. Omnistrate leverages the standard spec and those extensions together to complete the spec for your SaaS. We will go over the [extensions](#custom-tags) below in more detail.

## Automatic Docker Compose Generation

When using build from repo, Omnistrate will automatically generate a docker compose file for you if one doesn't already exist in your project. This generated compose file will include the necessary configuration based on your container image and any environment variables you specify during the build process.

## Docker Compose File Format

The Compose file is a YAML file defining:

- **Version** (Optional)
- **Services** (Required)
- **Networks**
- **Volumes**
- **Configs**
- **Secrets**

The default path for a Compose file is `compose.yaml`.

Omnistrate support Docker Compose 3.9 specification

```
  version: '3.9'
```

### Basic Structure

```
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
    volumes:
      - ./html:/usr/share/nginx/html
    environment:
      - ENV_VAR=value

volumes:
  data:

networks:
  frontend:
```

## Supported Native Tags

Here are the native tags Omnistrate supports either natively or with an optimized implementation:

### image

Specifies the image to start the container from. The image must follow the Open Container Specification addressable image format, as `[<registry>/][<project>/]<image>[:<tag>|@<digest>]`.

```
services:
  web:
    image: redis
    # or
    image: redis:5
    # or
    image: redis@sha256:0ed5d5928d4737458944eb604cc8509e245c3e19d02ad83935398bc4b991aac7
    # or
    image: my_private.registry:5000/redis
```

Native support for all public registries and private registries with credentials.

### expose

Defines the (incoming) port or a range of ports that Compose exposes from the container. These ports must be accessible to linked services and should not be published to the host machine.

```
services:
  web:
    expose:
      - "3000"
      - "8000"
      - "8080-8085/tcp"
```

### ports

Exposes container ports. Port mapping must not be used with `network_mode: host`.

**Short syntax:**

```
services:
  web:
    ports:
      - "3000"
      - "3000-3005"
      - "8000:8000"
      - "9090-9091:8080-8081"
      - "127.0.0.1:8001:8001"
      - "6060:6060/udp"
```

**Long syntax:**

```
services:
  web:
    ports:
      - name: web
        target: 80
        protocol: tcp
```

### volumes

Define mount host paths or named volumes that are accessible by service containers.

**Short syntax:**

```
services:
  web:
    volumes:
      - /var/lib/mysql
      - /opt/data:/var/lib/mysql
      - ./cache:/tmp/cache
      - ~/configs:/etc/configs/:ro
```

**Long syntax:**

```
services:
  web:
    volumes:
      - type: volume
        source: mydata
        target: /data
        volume:
          nocopy: true
      - type: bind
        source: ./static
        target: /opt/app/static
```

### environment

Defines environment variables set in the container. Environment variables can use either an array or a map.

**Map syntax:**

```
services:
  web:
    environment:
      RACK_ENV: development
      SHOW: "true"
      USER_INPUT:
```

**Array syntax:**

```
services:
  web:
    environment:
      - RACK_ENV=development
      - SHOW=true
      - USER_INPUT
```

#### env_file

Adds environment variables to the container based on the file content.

```
services:
  web:
    env_file: .env
    # or
    env_file:
      - ./a.env
      - ./b.env
```

**Env file format:** Each line in an `.env` file must be in `VAR[=[VAL]]` format. The following syntax rules apply:

- Lines beginning with `#` are processed as comments and ignored
- Blank lines are ignored
- Values can optionally be quoted
- Variable interpolation is supported using `${VARIABLE_NAME}` syntax

Example `.env` file:

```
# Set Rails/Rack environment
RACK_ENV=development
VAR="quoted"
DB_HOST=${DB_HOST}
```

### depends_on

Expresses startup and shutdown dependencies between services.

**Short syntax:**

```
services:
  web:
    depends_on:
      - db
      - redis
  redis:
    image: redis
  db:
    image: postgres
```

**Long syntax:**

```
services:
  web:
    depends_on:
      db:
        condition: service_healthy
        restart: true
      redis:
        condition: service_started
```

### entrypoint

Declares the default entrypoint for the service container. This overrides the `ENTRYPOINT` instruction from the service's Dockerfile.

```
services:
  web:
    entrypoint: /code/entrypoint.sh
    # or
    entrypoint:
      - php
      - -d
      - zend_extension=/usr/local/lib/php/extensions/no-debug-non-zts-20100525/xdebug.so
      - -d
      - memory_limit=-1
      - vendor/bin/phpunit
```

### command

Overrides the default command declared by the container image.

```
services:
  web:
    command: bundle exec thin -p 3000
    # or
    command: ["bundle", "exec", "thin", "-p", "3000"]
```

### init

Runs an init process (PID 1) inside the container that forwards signals and reaps processes. Tag is ignored and supported through cluster_init action hooks, which runs as a job.

### container_name and hostname

These tags are ignored. Omnistrate has a fully managed DNS service that doesn't require any explicit configuration to work.

### networks

Networks tags are ignored as Omnistrate auto-configures network and ports as part of the tenancy model.

### tmpfs

These tags are ignored. Omnistrate automatically mounts a temp file on /tmp folder.

### logging

These tags are ignored. Omnistrate offers its own managed logging system as integration that can be enabled by adding the `x-customer-integrations` custom spec followed by the `logs` item. This tag is ignored and supported through `x-customer-integrations`.

### healthcheck

These tags are ignored. Healthchecks are configured via `x-omnistrate-action-hooks`. Declares a check that's run to determine whether or not the service containers are "healthy". This tag is ignored and supported through `x-omnistrate-action-hooks`.

### user

User definition is achieved via `SECURITY_CONTEXT` environment variables according to the UID and GID.

```
services:
  web:
    user: "1001"
    # or
    user: "1001:1001"
```

### ulimits

Overrides the default ulimits for a container.

```
services:
  web:
    ulimits:
      nproc: 65535
      nofile:
        soft: 20000
        hard: 40000
```

### cap_add / cap_drop

Specifies additional container capabilities or capabilities to drop.

```
services:
  web:
    cap_add:
      - ALL
    cap_drop:
      - NET_ADMIN
      - SYS_ADMIN
```

### sysctls

These tags are ignored. Omnistrate sets up default configuration.

### labels

Allows you to specify custom metadata for resources, such as custom resource names or descriptions.

```
services:
  web:
    labels:
      - "name=Your Resource Name"
      - "description=Your Resource Description"
    # or
    labels:
      com.example.description: "Accounting webapp"
      com.example.department: "Finance"
```

### platform

Defines the target platform the containers for the service run on. It uses the `os[/arch[/variant]]` syntax.

```
services:
  web:
    platform: linux/amd64
    # or
    platform: linux/arm64
```

The supported values by Omnistrate are `linux/amd64` and `linux/arm64`. It is only used when an instance type is not explicitly defined.

### deploy

Specifies the configuration for the deployment and lifecycle of services. This includes resource constraints and requirements.

```
services:
  web:
    deploy:
      resources:
        limits:
          cpus: '1000m'
          memory: 256M
        reservations:
          cpus: '100m'
          memory: 100M
```

**Resource configuration:**

- **limits**: Define the maximum resources the container can use
- **cpus**: Maximum CPU allocation (can be specified as decimal or with 'm' suffix for millicores)
- **memory**: Maximum memory allocation (supports units like M, G)
- **reservations**: Define the minimum resources guaranteed to the container
- **cpus**: Minimum CPU allocation
- **memory**: Minimum memory allocation

### privileged

Configures the service container to run with elevated privileges. Support and actual impacts are platform specific.

```
services:
  web:
    privileged: true
```

## Custom Tags

In addition, Omnistrate also supports several custom tags as follows.

### x-omnistrate-service-plan

`x-omnistrate-service-plan` allows you to configure the Plan for your SaaS in the compose specification file.

```
x-omnistrate-service-plan:
  name: 'Mysql Free Tier'
  tenancyType: 'OMNISTRATE_DEDICATED_TENANCY'
  features:
    CUSTOM_NETWORKS:
    CUSTOM_DEPLOYMENT_CELL_PLACEMENT:
      maximumDeploymentsPerCell: 1
  deployment:
    hostedDeployment:
      awsAccountId: 'xxxxxxxxxxx'
      awsBootstrapRoleAccountArn: 'arn:aws:iam::xxxxxxxxxxx:role/omnistrate-bootstrap-role'
      gcpProjectId: 'test-account'
      gcpProjectNumber: 'xxxxxxxxxxx3'
      gcpServiceAccountEmail: 'bootstrap.service@gcp.test.iam'
      azureSubscriptionId: 'xxxxxxxx-xxxx-xxx-xxxx-xxxxxxxxxx'
      azureTenantId: 'xxxxxxxx-xxxx-xxx-xxxx-xxxxxxxxxx'
```

Configuration fields:

- **name**: The name of the Plan
- **tenancyType**: The tenancy type of the Plan. Options:
- `OMNISTRATE_DEDICATED_TENANCY`: Infrastructure dedicated to a single customer deployment
- `OMNISTRATE_MULTI_TENANCY`: Infrastructure shared among multiple customers and deployments
- `CUSTOM_TENANCY`: Infrastructure provisioned by Omnistrate with custom affinity configuration
- **deployment**: The deployment type of the Plan. Options:
- `hostedDeployment`: The Plan is deployed on your account
- `byoaDeployment`: The Plan is deployed on your customer's account
- Omitted: Omnistrate will host the Plan for you

#### Custom Deployment Cell Placement

The `CUSTOM_DEPLOYMENT_CELL_PLACEMENT` feature allows you to control how many deployments can be co-located on the same host cluster (deployment cell / Kubernetes cluster). This provides fine-grained control over deployment isolation and resource allocation.

```
x-omnistrate-service-plan:
  name: 'PostgreSQL Service'
  features:
    CUSTOM_DEPLOYMENT_CELL_PLACEMENT:
      maximumDeploymentsPerCell: 1
```

#### Custom Networks

The `CUSTOM_NETWORKS` feature allows your customers to define network partitioning on a dedicated stack while keeping the service deployed in the Service Provider Account. This provides complete isolation by provisioning a dedicated stack that is not shared with other customers, enabling private network connectivity while maintaining self-service capabilities.

```
x-omnistrate-service-plan:
  name: 'PostgreSQL Service'
  features:
    CUSTOM_NETWORKS:
```

> **Note:** For `deployment`, replace the account numbers, project id and other information with your own account information.

### x-omnistrate-my-account (deprecated)

`x-omnistrate-my-account` is deprecated allows you to configure your cloud provider account details for hosted deployments. Use `x-omnistrate-service-plan.deployment.hostedDeployment` tag to configure your cloud account.

## x-omnistrate-byoa (deprecated)

`x-omnistrate-byoa` is deprecated allows you to configure your cloud provider account details for hosted deployments. Use `x-omnistrate-service-plan.deployment.byoaDeployment` tag to configure your cloud account.

### x-omnistrate-mode-internal

`x-omnistrate-mode-internal` tag allows you to tag any Resource in the compose specification as internal.

```
services:
  internal-service:
    image: my-internal-service
    x-omnistrate-mode-internal: true
```

- Possible values are `true` or `false`
- Default value is `false`
- If set to `true`, this Resource will be internal and won't be exposed to your customers
- If set to `false`, the Resource will be exposed to your customers to configure and provision them

### x-omnistrate-api-params

`x-omnistrate-api-params` allows you to define API params in addition to environment variables.

```
services:
  database:
    image: postgres
    x-omnistrate-api-params:
      - key: writerInstanceType
        description: Writer Instance Type
        name: Writer Instance Type
        type: String
        modifiable: true
        required: true
        export: true
```

Each API parameter can be configured to specify:

- **key**: Unique identifier for the parameter
- **name**: Display name of the parameter
- **description**: Description for the parameter
- **type**: Type of the parameter (String, Float64, Boolean, etc.)
- **required**: Specify if this field is a required parameter
- **export**: Configures if this field will be returned as part of the describe call
- **modifiable**: Configure if this field is modifiable once configured
- **defaultValue**: Default value for this field
- **options**: List of options for the user to choose from
- **labeledOptions**: List of options with user-friendly labels
- **limits**: Minimum and maximum values for this field
- **regex**: Regex pattern to validate the input

**Example with options:**

```
x-omnistrate-api-params:
  - key: writerInstanceType
    description: Writer Instance Type
    name: Writer Instance Type
    type: String
    modifiable: true
    required: true
    export: true
    options:
      - t4g.small
      - t4g.medium
      - t4g.large
```

**Example with labeled options:**

```
x-omnistrate-api-params:
  - key: writerInstanceType
    description: Writer Instance Type
    name: Writer Instance Type
    type: String
    modifiable: true
    required: true
    export: true
    labeledOptions:
      Small: t4g.small
      Medium: t4g.medium
      Large: t4g.large
```

**Example with limits:**

```
x-omnistrate-api-params:
  - key: postgresqlRootPassword
    description: Postgresql Root Password
    name: Postgresql Root Password
    type: String
    modifiable: true
    required: true
    export: true
    limits:
      minLength: 8
      maxLength: 64
  - key: postgresqlPort
    description: Postgresql Port
    name: Postgresql Port
    type: Float64
    modifiable: true
    required: true
    export: true
    limits:
      min: 1024
      max: 65535
```

**Example with default value:**

```
x-omnistrate-api-params:
  - key: postgresqlUsername
    description: Postgresql Username
    name: Postgresql Username
    type: String
    modifiable: true
    required: true
    export: true
    defaultValue: postgres
```

**Example with regex validation:**

```
x-omnistrate-api-params:
  - key: postgresqlPassword
    description: Default DB Password
    name: Password
    type: String
    modifiable: false
    required: true
    export: false
    regex: ^[a-zA-Z0-9!@#$%^&*()_+]{8,32}$
```

#### API Parameter Types

The valid parameter types are:

- **boolean**: A true or false value
- **int**: A signed integer (platform-dependent size, at least 32 bits)
- **string**: A sequence of characters
- **password**: A sensitive string for passwords, stored securely and masked in outputs
- **int32**: A 32-bit signed integer
- **int64**: A 64-bit signed integer
- **uint**: An unsigned integer (platform-dependent size, at least 32 bits)
- **uint32**: A 32-bit unsigned integer
- **uint64**: A 64-bit unsigned integer
- **float32**: A 32-bit floating-point number
- **float64**: A 64-bit floating-point number
- **bytes**: A sequence of bytes
- **json**: A JSON formatted string
- **any**: Any of the supported types
- **resource**: Used for resource linking to enforce creating a linked resource before creating a parent resource

#### Resource Linking with API Parameters

An API parameter of resource type can be used to link any resource with another resource to enforce creating a linked resource before creating a parent resource.

```
services:
  Proxy:
    image: omnistrate/pgadmin4:7.5
    x-omnistrate-api-params:
      - key: database
        description: backend database
        name: database
        type: Resource
        export: false
        required: true
        modifiable: false
  Database:
    image: 'bitnami/postgresql:latest'
```

In this example, customers must create a Database instance first, then create a Proxy instance by specifying which Database instance to link to.

#### Parameter Dependency Mapping

You can map API parameters between dependent resources:

```
services:
  Cluster:
    image: omnistrate/noop
    depends_on:
      - Writer
      - Reader
    x-omnistrate-api-params:
      - key: instanceType
        description: Instance Type
        name: Instance Type
        type: String
        modifiable: true
        required: true
        export: true
        defaultValue: t4g.small
        parameterDependencyMap:
          Writer: writerInstanceType
          Reader: readerInstanceType
```

This maps the `instanceType` parameter of the Cluster resource to `writerInstanceType` of the Writer resource and `readerInstanceType` of the Reader resource.

> **Note:** All environment variables are **automatically** added as API parameters if `x-omnistrate-api-params` is not specified.

### x-omnistrate-actionhooks

`x-omnistrate-actionhooks` allows you to configure action hooks for a given Resource.

```
services:
  database:
    image: postgres
    x-omnistrate-actionhooks:
      - scope: CLUSTER
        type: INIT
        commandTemplate: >
          PGPASSWORD={{ $var.postgresqlRootPassword }} psql -U postgres
          -h writer {{ $var.postgresqlDatabase }} -c "create extension vector"
```

Action hooks allow you to inject custom code at different phases of the lifecycle of your control plane operations. In the above example, we are enabling vector extension for Postgres on cluster initialization.

#### Action Hook Scopes and Types

Action hooks are categorized into two scopes:

**Node Scope** - runs for every node of the Resource:

- **`HEALTH_CHECK`**: Periodic health checks mapped to Kubernetes liveness probes
- **`READINESS_CHECK`**: Determines service readiness for traffic (defaults to `HEALTH_CHECK`)
- **`STARTUP_CHECK`**: Ensures service startup completion (defaults to `HEALTH_CHECK`)
- **`POST_START`**: Executes after node starts
- **`ADD`**: Triggers when a new node is added
- **`REMOVE`**: Triggers when a node is removed
- **`INIT`**: Runs initialization before node starts
- **`PROMOTE`**: Triggers when a node is promoted to primary role
- **`DEMOTE`**: Triggers when a node is demoted before replacement

**Cluster Scope** - runs for the entire Resource:

- **`POST_START`**: Executes after all nodes have started
- **`PRE_START`**: Runs before all nodes are started
- **`POST_STOP`**: Executes after all nodes are stopped
- **`PRE_STOP`**: Runs before all nodes are stopped
- **`POST_UPGRADE`**: Executes after all nodes are upgraded
- **`PRE_UPGRADE`**: Runs before all nodes are upgraded
- **`INIT`**: Runs initialization before all nodes are started

#### Advanced Action Hook Configuration

Action hooks support additional configuration options:

```
services:
  high-performance-db:
    image: postgres:15
    x-omnistrate-actionhooks:
      - image: busybox:1.37
        scope: NODE
        type: INIT
        command:
          - /bin/sh
          - -c
        commandTemplate: |
          # Set the AIO max number of events
          sysctl -w fs.aio-max-nr='102052'
```

**Configuration fields:**

- **image**: Custom container image for the action hook
- **command**: Command array to execute
- **commandTemplate**: Template for the command with variable interpolation
- **scope**: NODE or CLUSTER
- **type**: Action hook type as listed above

### x-omnistrate-compute

`x-omnistrate-compute` allows you to customize the compute parameters of a Resource.

You can customize the following:

- Number of replicas
- Instance type across cloud providers
- Size of the root volume (GiB) across instances

```
services:
  web:
    image: nginx
    x-omnistrate-compute:
      replicaCountAPIParam: numReplicas
      instanceTypes:
        - cloudProvider: aws
          name: t4g.small
        - cloudProvider: gcp
          name: e2-medium
        - cloudProvider: azure
          name: Standard_B2als_v2
      rootVolumeSizeGi: 10
```

In the above example, `replicaCountAPIParam` is configured dynamically using `numReplicas` parameter. The users of your service can decide how many replicas they want. Similarly, `instanceTypes` can also be dynamic if you want your customers to specify the machine type among the list of options. For `rootVolumeSizeGi`, you can specify any integer between 10 and 16384 (up to 16 TB).

#### GPU Accelerator Configuration

GPU accelerator configuration enables you to specify dedicated GPU resources and is only required to attach for some instance types on GCP.

1. **External GPU attachment** using `acceleratorConfiguration` (N1 instances + Tesla GPUs only)
1. **Built-in GPU instances** (G2, A2, A3, A4 families with modern GPUs)

The `acceleratorConfiguration` feature allows declarative specification of GPU accelerators attached to compute instances on GCP.

> **Warning:** External GPU attachment is **only supported** with N1 instances and Tesla series GPUs (T4, V100, P100, P4) on GCP

**Configuration in x-omnistrate-compute:**

```
services:
  gpu-service:
    image: tensorflow/tensorflow:latest-gpu
    x-omnistrate-compute:
      instanceTypes:
        - name: n1-standard-8
          cloudProvider: gcp
          configurationOverrides:
            acceleratorConfiguration:   
              type: "nvidia-tesla-t4"
              count: 1
```

**Multi-GPU Configuration:**

```
services:
  gpu-cluster:
    image: pytorch/pytorch:latest
    x-omnistrate-compute:
      instanceTypes:
        - name: n1-standard-16
          cloudProvider: gcp
          configurationOverrides:
            acceleratorConfiguration:   
              type: "nvidia-tesla-t4"
              count: 4
```

For AWS, Azure and modern GPUs in GCP (L4, A100, H100) , instance families come with GPUs mounted:

**G2 Family - NVIDIA L4 GPUs:**

```
services:
  modern-gpu-service:
    image: nvidia/cuda:12.0-runtime-ubuntu20.04
    x-omnistrate-compute:
      instanceTypes:
        - name: g2-standard-8
          cloudProvider: gcp
          # No acceleratorConfiguration needed - L4 GPU is built-in
```

### x-omnistrate-storage

`x-omnistrate-storage` allows you to customize the storage parameters of a Resource.

```
services:
  database:
    image: postgres
    x-omnistrate-storage:
      aws:
        instanceStorageType: AWS::EBS_GP3
        instanceStorageSizeGi: 100
        instanceStorageIOPSAPIParam: instanceStorageIOPS
        instanceStorageThroughputAPIParam: instanceStorageThroughput
      gcp:
        instanceStorageType: GCP::PD_BALANCED
        instanceStorageSizeGi: 100
      azure:
        instanceStorageType: AZURE::STANDARD_SSD
        instanceStorageSizeGi: 100
```

You can customize the following:

- Storage type from block device to blobs or both
- Size of the volume
- Storage IOPS
- Storage throughput

Like above, the actual value of each of the fields could be static or dynamic.

#### Shared File System

Shared file systems allow you to save data in a persistent volume and share it across different pods within the same deployment cluster. This is ideal for scenarios like AI model training where multiple workers need access to the same dataset, or when you need to scale storage dynamically without downtime.

Define shared file system using volumes in your compose specification:

```
volumes:
  file_system_data:
    driver: sharedFileSystem
    driver_opts:
      efsThroughputMode: provisioned
      efsPerformanceMode: generalPurpose
      efsProvisionedThroughputInMibps: 100
```

**Configuration options:**

- **driver**: Must be set to `sharedFileSystem` to enable Omnistrate shared file system
- **driver_opts**: Customize the shared file system behavior
- **efsThroughputMode**: Throughput mode (provisioned, bursting)
- **efsPerformanceMode**: Performance mode (generalPurpose, maxIO)
- **efsProvisionedThroughputInMibps**: Provisioned throughput in MiB/s (when using provisioned mode)

Mount the shared file system volume to your Resources:

```
services:
  database:
    image: postgres
    volumes:
    volumes:
      - source: file_system_data
        target: /var/lib/redis/data
        type: volume
        x-omnistrate-storage:
          aws:
            clusterStorageType: AWS::EFS
      - source: file_system_data
        target: /var/lib/postgresql/data
        type: volume
        x-omnistrate-storage:
          aws:
            clusterStorageType: AWS::EFS
```

**Mount configuration:**

- **source**: Name of the volume defined in the volumes section
- **target**: Path where you want to mount the volume in the container
- **type**: Must be set to `volume`
- **x-omnistrate-storage**: Storage configuration with `clusterStorageType: AWS::EFS`

> **Note:** Shared file system is currently only available for AWS EFS storage. Contact [support@omnistrate.com](mailto:support@omnistrate.com) for other shared file system options.

#### Blob Storage

Blob storage allows you to define blob storage buckets and their mount paths across Resources. Omnistrate manages the lifecycle of storage volumes (creation and deletion with instance) and mounts them as local volumes.

> **Note:** Blob storage is not available for the Omnistrate hosted model.

Define blob storage in the compose specification:

```
volumes:
  bucket_metadata:
    driver: blob
  bucket_additional_resources:
    driver: blob
```

This configuration provisions two buckets per instance, named based on the Resource name and instance ID.

Each bucket can be mounted to zero, one, or multiple Resources:

```
services:
  app:
    image: myapp
    volumes:
      - source: bucket_metadata
        target: /mnt/blob-metadata
        type: volume
        x-omnistrate-storage:
          gcp:
            clusterStorageType: GCP::GCS
      - source: bucket_additional_resources
        target: /mnt/blob-additional-resources
        type: volume
        x-omnistrate-storage:
          gcp:
            clusterStorageType: GCP::GCS
```

### x-omnistrate-capabilities

`x-omnistrate-capabilities` allows you to add capabilities to your Resources.

```
services:
  web:
    image: nginx
    x-omnistrate-capabilities:
      httpReverseProxy:
        targetPort: 80
      enableMultiZone: true
      enableEndpointPerReplica: true
      customDNS:
        targetPort: 80
      autoscaling:
        maxReplicas: 1
        minReplicas: 1
      serverlessConfiguration:
        enableAutoStop: true
        minimumNodesInPool: 5
        targetPort: 3306
```

#### customDNS

The `customDNS` capability enables endpoint aliases for your Resources, allowing users to assign specific aliases to deployment or resource instance endpoints. This provides enhanced branding, customization, and control over infrastructure.

```
services:
  web:
    image: nginx
    x-omnistrate-capabilities:
      customDNS:
        targetPort: 80
```

**Configuration fields:**

- **targetPort**: The port number where your HTTP service is listening

#### custom sidecars

The `sidecars` capability allows you to enhance your SaaS product's functionality without changing its core service. Sidecars are add-on containers that run alongside your main application container, providing additional features or capabilities while maintaining isolation from the main application.

```
services:
  web:
    image: nginx
    x-omnistrate-capabilities:
      sidecars:
        tooling:
          imageNameWithTag: "busybox:stable"
        monitoring:
          imageNameWithTag: "prometheus/node-exporter:latest"
          securityContext:
            runAsUser: 10
            runAsGroup: 99
            runAsNonRoot: true
            capabilities:
              add:
                - SYS_RESOURCE
          resourceLimits:
            cpu: "250m"
            memory: "256Mi"
          command:
            - "/bin/node_exporter"
          args:
            - "--path.rootfs=/host"
            - "--collector.filesystem.ignored-mount-points"
```

**Configuration fields:**

- **imageNameWithTag**: Container image with tag for the sidecar (required)
- **securityContext**: Security context configuration for the sidecar
- **runAsUser**: User ID to run the container
- **runAsGroup**: Group ID to run the container
- **runAsNonRoot**: Whether to run as non-root user
- **capabilities**: Linux capabilities to add or drop
- **resourceLimits**: Resource constraints for the sidecar
- **cpu**: CPU limit (e.g., "250m" for 250 millicores)
- **memory**: Memory limit (e.g., "256Mi" for 256 MiB)
- **command**: Entry point command for the sidecar container
- **args**: Arguments to pass to the command

#### stableEgressIP

The `stableEgressIP` capability provides a stable egress IP address for outbound traffic from your Resources. This is useful when you need to whitelist your service's IP address with external services or APIs.

```
services:
  web:
    image: nginx
    x-omnistrate-capabilities:
      stableEgressIP: true
```

#### processCoreDump

The `processCoreDump` capability enables process core dump collection for debugging application crashes and analyzing process failures.

```
services:
  database:
    image: postgres
    x-omnistrate-capabilities:
      processCoreDump: /var/lib/data/cores/core.%e.%p.%t
```

**Configuration:**

- Specify the path where core dumps should be stored
- Use format specifiers for core dump file naming:
- `%e`: Executable name
- `%p`: Process ID
- `%t`: Timestamp

#### serviceAccountPolicies

The `serviceAccountPolicies` capability enables your application to securely access cloud-native services by configuring appropriate service account permissions.

```
services:
  app:
    image: myapp
    x-omnistrate-capabilities:
      serviceAccountPolicies:
        aws:
          - MSK_CONNECT
          - SECRETS_MANAGER
          - LAMBDA
          - SQS
        gcp:
          - WORKLOAD_IDENTITY_IAM_BINDING
```

**AWS Policies:**

- **MSK_CONNECT**: Enables AWS MSK Connect access
- **SECRETS_MANAGER**: Enables AWS Secrets Manager access
- **LAMBDA**: Enables AWS Lambda access (including Serverless framework permissions)
- **SQS**: Enables Amazon SQS access

**GCP Policies:**

- **WORKLOAD_IDENTITY_IAM_BINDING**: Binds resource workload identity to IAM service account, granting additional GCP permissions (Logs, Metrics, Secrets)

#### backupConfiguration

The `backupConfiguration` capability enables automatic backup and point-in-time restore for your Resources, providing data protection and recovery capabilities.

```
services:
  database:
    image: postgres
    x-omnistrate-capabilities:
      backupConfiguration:
        backupRetentionInDays: 7
        backupPeriodInHours: 2
```

**Configuration fields:**

- **backupRetentionInDays**: Number of days to retain backups
- **backupPeriodInHours**: Frequency of backup creation in hours

**Storage Volume Backup Control:** You can disable backups for specific storage volumes by adding `disableBackup: true` in the volume configuration:

```
services:
  database:
    image: postgres
    volumes:
      - source: temp_data
        target: /tmp/data
        type: volume
        x-omnistrate-storage:
          aws:
            instanceStorageType: AWS::EBS_GP2
            instanceStorageSizeGi: 50
            disableBackup: true
```

> **Note:** Backup capability is only available for Tenant-Aware Resources and applies to AWS EBS, GCP Persistent Disk, and Azure Disk storage types. When backup retention is updated, changes only apply to new backups.

#### autoscaling

For autoscaling based on custom application metrics instead of default CPU/memory metrics:

```
x-omnistrate-capabilities:
  autoscaling:
    scalingMetric:
      metricEndpoint: "http://localhost:9187/metrics"
      metricLabelName: "application_name"
      metricLabelValue: "psql"
      metricName: "pg_stat_activity_count"
```

#### serverlessConfiguration

For complex serverless scenarios requiring custom metrics, session state preservation, or proxy configuration control, you can use advanced serverless mode:

```
x-omnistrate-capabilities:
  autoscaling:
    maxReplicas: 5
    minReplicas: 1
    idleMinutesBeforeScalingDown: 2
    idleThreshold: 20
    overUtilizedMinutesBeforeScalingUp: 3
    overUtilizedThreshold: 80
  serverlessConfiguration:
    targetPort: 3306
    enableAutoStop: true
    minimumNodesInPool: 5
```

### x-omnistrate-load-balancer

`x-omnistrate-load-balancer` allows you to configure a load balancer for your Resources in the compose specification file adding L7-L4 capabilities.

Adding L7 load balancing capabilities to your Resources allows to route traffic based on the URL path, while L4 routes traffic based on the port number.

```
version: "3"
x-customer-integrations:
  logs: 
  metrics: 
x-omnistrate-load-balancer:
  https:
    - name: PGAdmin
      description: L7 Load Balancer for PGAdmin - New
      paths:
        - associatedResourceKey: admin
          path: /
          backendPort: 80

  tcp:
    - name: Writer
      description: L4 Load Balancer for Writer
      ports:
        - associatedResourceKeys:
            - writer
          ingressPort: 5432
          backendPort: 5432

services:
  admin:
    image: dpage/pgadmin4

  writer:
    image: postgres

  reader:
    image: postgres
```

### x-omnistrate-job-config

`x-omnistrate-job-config` allows you to configure a Resource as a job that runs one time during create, update, or modification operations of a deployment.

When a Resource is configured with job settings, it will execute once during the deployment lifecycle and complete its task. This is useful for initialization scripts, data migrations, setup tasks, or any one-time operations.

```
services:
  hello-world:
    build:
      context: ./jobs/hello-world
      dockerfile: Dockerfile
    environment:
      RAY_ADDRESS: "ray://{{ $var.rayClusterAddress }}:10001"
      SCRIPT_PATH: "submit_job.py"
    deploy:
      resources:
        reservations:
          cpus: "0.1"
          memory: 256M
        limits:
          cpus: "0.5"
          memory: 1G
    privileged: true
    platform: linux/amd64
    volumes:
      - source: ./jobs/hello-world
        target: /app
        type: bind
      - source: ./tmp
        target: /tmp
        type: bind
    x-omnistrate-compute:
      replicaCountApiParam: numReplicas
    x-omnistrate-job-config:
      backoffLimit: 0
      activeDeadlineSeconds: 3600
```

Job-specific configurations:

- **backoffLimit**: Number of retries before considering the job as failed (set to 0 to disable retries)
- **activeDeadlineSeconds**: Maximum duration (in seconds) the job is allowed to run before termination

> **Note:** Jobs are designed to run to completion and then terminate. They are not meant for long-running services.

### x-omnistrate-image-registry-attribute

`x-omnistrate-image-registry-attributes` allows you to configure image registries.

```
x-omnistrate-image-registry-attributes:
  docker.io:
    auth:
      username: username
      password: password
```

> **Note:** For private image registries, username and password are required. Images will only be downloaded in your account (hosted mode) or in your customers' account (BYOA mode).

### x-customer-integrations

`x-customer-integrations` allows you to configure customer-specific integrations such as licensing, logs, and metrics.

```
x-customer-integrations:
  logs:
  metrics:
```

#### Licensing Protection System

The licensing protection system ensures that only authorized subscribed users can access and use your software. This is particularly important for BYOA/BYOC deployments and on-premises installations.

```
x-customer-integrations:
  licensing: 
    licenseExpirationInDays: 7
    productPlanUniqueIdentifier: 'PRODUCT-SAMPLE-SKU-UNIQUE-VALUE'
```

Configuration fields:

- **licensing**: Configure licensing integration
- **licenseExpirationInDays**: Number of days before license expires (default: 7 days)
- **productPlanUniqueIdentifier**: Unique identifier for the product plan SKU (optional, defaults to product tier ID)

#### Customer Observability with Cloud Native Integration

For BYOA (Bring Your Own Account) deployments, you can enable cloud native observability:

```
x-customer-integrations:
  logs:
    provider: native 
  metrics:
    provider: native
```

This integrates with the cloud provider's native observability platform (CloudWatch for AWS, Operations Suite for GCP, Application Insights for Azure).

### x-omnistrate-integrations (deprecated)

`x-omnistrate-integrations` allows you to add integrations to your service. This tag is deprecated, use `x-customer-integrations`.

### x-internal-integrations

`x-internal-integrations` allows you to configure internal integrations that are not exposed to customers.

```
x-internal-integrations:
  logs:
  metrics:
    additionalMetrics:
      postgres:
        prometheusEndpoint: "http://localhost:9187/metrics"
        metrics:
          pg_settings_enable_sort: # shared with customer
          pg_settings_autovacuum: # internal only metric
```

This extension allows you to define metrics and other integrations that are only visible internally and not exposed to your customers.

#### Internal Observability Options

**Omnistrate Native:**

- **logs**: Enable real-time logging for your support team to manage the customer fleet
- **metrics**: Enable real-time infrastructure metrics dashboard

**OpenTelemetry Providers:** You can ship metrics and logs to third-party providers like NewRelic, Signoz, or Datadog:

```
x-internal-integrations:
  metrics:
    provider: newRelic  # or signoz, datadog
    endpoint: https://otlp.nr-data.net
    secretLocators:
      aws: arn:aws:secretsmanager:us-west-2:xxxxxxxxxxx:secret:mySecret123456-abc123
      gcp: projects/xxxxxxxxxxx/secrets/mySecret123456-abc123/versions/latest
      azure: KeyVaultName/SecretName
    serviceComponentsConfiguration:
      postgres:
        prometheusEndpoint: "http://localhost:9187/metrics"
  logs:
    provider: newRelic
    endpoint: https://otlp.nr-data.net
    secretLocators:
      aws: arn:aws:secretsmanager:us-west-2:xxxxxxxxxxx:secret:mySecret123456-abc123
```

**Cloud Native:** Enable integration with cloud provider's native observability platform:

```
x-internal-integrations:
  logs:
    provider: native 
  metrics:
    provider: native
```

#### Custom Metrics Configuration

For additional application metrics, you can specify custom metrics with aggregation functions and label filters:

```
x-internal-integrations:
  metrics:
    additionalMetrics:
      postgres:
        prometheusEndpoint: "http://localhost:9187/metrics"
        metrics:
          pg_stat_activity_count:
            average_active_activity:
              aggregationFunction: avg
              labelFilters:
                state: active
            max_postgres_activity:
              aggregationFunction: max
              labelFilters:
                datname: postgres
```

**Supported aggregation functions:** sum, avg, max, min

#### GPU Slicing and Multi-Tenant GPU

GPU slicing enables multiple workloads to efficiently share GPU resources, maximizing utilization and reducing costs. Omnistrate supports both NVIDIA time-slicing and Multi-Instance GPU (MIG) technologies.

Enable GPU slicing using the `x-internal-integrations` extension:

```
x-internal-integrations: 
  multiTenantGpu: 
    instanceType: g4dn.xlarge
    timeSlicingReplicas: 2
    migProfile: 1g.5gb  # optional: for MIG-capable GPUs
```

**Configuration parameters:**

- **instanceType**: GPU-enabled EC2 instance type (g4dn.xlarge, p3.2xlarge, p4d.24xlarge, etc.)
- **timeSlicingReplicas**: Number of virtual GPU replicas (2, 4, 8, etc.)
- **migProfile**: MIG profile for A100/H100 GPUs (optional)

## Variable Interpolation and System Parameters

### API Parameters

Access API parameters defined in `x-omnistrate-api-params` using `$var.<parameter-key>`:

```
services:
  database:
    image: postgres
    environment:
      - POSTGRES_PASSWORD=$var.postgresqlPassword
      - POSTGRES_DATABASE=$var.postgresqlDatabase
      - POSTGRES_USERNAME=$var.postgresqlUsername
    x-omnistrate-api-params:
      - key: postgresqlPassword
        type: String
        required: true
      - key: postgresqlDatabase
        type: String
        required: true
      - key: postgresqlUsername
        type: String
        required: true
```

### Omnistrate System Parameters

Omnistrate provides system parameters that are replaced with actual values at runtime using the format `$sys.<variable-name>`. These parameters provide contextual information about the deployment environment.

> **Note:** All variables are substituted with their original data type. If you want to use a variable as a string, wrap it in escaped quotes: `\"$sys.deploymentCell.region\"`.

## Fragments

You can use built-in YAML features to make your Compose file neater and more efficient.

### Anchors and Aliases

Anchors are created using the `&` sign, and aliases use the `*` sign:

```
x-common-variables: &common-variables
  POSTGRES_DB: myapp
  POSTGRES_USER: postgres

services:
  db:
    image: postgres
    environment:
      <<: *common-variables
      POSTGRES_PASSWORD: secret

  backup:
    image: postgres
    environment: *common-variables
```
