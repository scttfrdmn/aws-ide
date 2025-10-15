# Private Subnet Deployment Guide

This guide covers best practices for deploying aws-jupyter instances in private subnets for enhanced security and compliance.

## Table of Contents

- [Overview](#overview)
- [When to Use Private Subnets](#when-to-use-private-subnets)
- [Architecture Options](#architecture-options)
- [NAT Gateway vs No Internet Access](#nat-gateway-vs-no-internet-access)
- [Quick Start](#quick-start)
- [Cost Analysis](#cost-analysis)
- [VPC Endpoints Alternative](#vpc-endpoints-alternative)
- [Security Best Practices](#security-best-practices)
- [Troubleshooting](#troubleshooting)
- [Migration from Public Subnets](#migration-from-public-subnets)

## Overview

Private subnets provide an additional layer of security by isolating instances from direct internet access. This is particularly important for:

- **Compliance requirements** (HIPAA, PCI-DSS, SOC 2)
- **Sensitive data processing** (PII, PHI, financial data)
- **Production environments**
- **Enterprise security policies**

### Key Benefits

✅ **Security**
- No direct internet exposure
- Reduced attack surface
- Easier compliance audits
- Defense in depth

✅ **Network Control**
- All outbound traffic through NAT Gateway or VPC endpoints
- Centralized network monitoring
- Better traffic visibility
- Consistent security policies

✅ **Compliance**
- Meets many regulatory requirements
- Documented network isolation
- Auditable network flows
- Separation of concerns

### Trade-offs

⚠️ **Costs**
- NAT Gateway: ~$45-50/month per AZ
- Data transfer charges (NAT Gateway)
- VPC Endpoints: ~$7-10/month per endpoint

⚠️ **Complexity**
- Additional networking setup
- More moving parts to manage
- Troubleshooting requires more AWS knowledge

## When to Use Private Subnets

### ✅ Use Private Subnets When:

1. **Regulatory Compliance**
   - HIPAA, PCI-DSS, SOC 2, or other compliance frameworks
   - Industry-specific security requirements
   - Corporate security policies mandate it

2. **Sensitive Data**
   - Processing PII (Personally Identifiable Information)
   - Handling PHI (Protected Health Information)
   - Financial or payment data
   - Proprietary research data

3. **Production Environments**
   - Customer-facing applications
   - Critical business workloads
   - Multi-tenant environments

4. **Enterprise Deployments**
   - Corporate network integration
   - Centralized security monitoring
   - Defense-in-depth architecture

### ❌ Public Subnets Are Fine For:

1. **Development and Testing**
   - Personal learning projects
   - Proof of concepts
   - Development environments

2. **Non-Sensitive Data**
   - Public datasets
   - Academic research (non-sensitive)
   - Example notebooks

3. **Cost-Sensitive Workloads**
   - Budget constraints
   - Short-lived instances
   - Infrequent usage

4. **Rapid Prototyping**
   - Quick experiments
   - Fast iteration cycles
   - No compliance requirements

## Architecture Options

### Option 1: Private Subnet with NAT Gateway (Recommended)

**Architecture:**
```
Internet Gateway
       ↓
Public Subnet (NAT Gateway)
       ↓
Private Subnet (Jupyter Instance)
```

**Best For:**
- Production workloads
- Instances need package installations
- Regular updates required
- Jupyter extensions needed

**Command:**
```bash
aws-jupyter launch \
  --connection session-manager \
  --subnet-type private \
  --create-nat-gateway
```

**Pros:**
- ✅ Full internet access for packages
- ✅ No direct inbound access
- ✅ Works with all environments
- ✅ Easy package installation

**Cons:**
- ❌ NAT Gateway costs (~$45/month)
- ❌ Data transfer charges
- ❌ Additional AWS resource

### Option 2: Private Subnet without Internet (Cost-Effective)

**Architecture:**
```
Private Subnet (Jupyter Instance)
       ↓
No Internet Access
```

**Best For:**
- Pre-configured environments
- No additional package needs
- Cost-sensitive deployments
- Air-gapped requirements

**Command:**
```bash
aws-jupyter launch \
  --connection session-manager \
  --subnet-type private
```

**Pros:**
- ✅ Lowest cost (no NAT Gateway)
- ✅ Maximum isolation
- ✅ Simple architecture
- ✅ Fastest launch time

**Cons:**
- ❌ No pip install
- ❌ No apt-get install
- ❌ No Jupyter extensions (unless pre-installed)
- ❌ Requires Session Manager or VPC endpoints

### Option 3: Private Subnet with VPC Endpoints

**Architecture:**
```
Private Subnet (Jupyter Instance)
       ↓
VPC Endpoints (S3, ECR, etc.)
```

**Best For:**
- Cost-conscious production
- Specific AWS service access
- Known package requirements
- S3 data access patterns

**Command:**
```bash
# Step 1: Create VPC endpoints (one-time setup)
aws ec2 create-vpc-endpoint \
  --vpc-id vpc-12345 \
  --service-name com.amazonaws.us-west-2.s3 \
  --route-table-ids rtb-12345

# Step 2: Launch instance
aws-jupyter launch \
  --connection session-manager \
  --subnet-type private
```

**Pros:**
- ✅ Lower cost than NAT Gateway
- ✅ Private connectivity to AWS services
- ✅ Good for S3-heavy workloads
- ✅ No data transfer charges (most endpoints)

**Cons:**
- ❌ Per-endpoint costs (~$7/month each)
- ❌ Requires advance planning
- ❌ Limited to AWS services
- ❌ No PyPI or external package access

## NAT Gateway vs No Internet Access

### Cost Comparison (Monthly)

| Component | NAT Gateway | No Internet | VPC Endpoints |
|-----------|-------------|-------------|---------------|
| **NAT Gateway** | $32.40 | $0 | $0 |
| **Data Transfer (50GB)** | $2.25 | $0 | $0 |
| **VPC Endpoints** | $0 | $0 | $7-14 |
| **Instance (m7g.large)** | ~$60 | ~$60 | ~$60 |
| **Total** | **~$95** | **~$60** | **~$67-74** |

**Calculation Details:**
- NAT Gateway: $0.045/hour = $32.40/month
- Data transfer: $0.045/GB = $2.25 for 50GB
- VPC Endpoints: $0.01/hour per endpoint (~$7/month)
- Instance costs same for all options

### Capability Comparison

| Feature | NAT Gateway | No Internet | VPC Endpoints |
|---------|-------------|-------------|---------------|
| **pip install** | ✅ Yes | ❌ No | ❌ No* |
| **apt-get install** | ✅ Yes | ❌ No | ❌ No* |
| **Jupyter extensions** | ✅ Yes | ⚠️ Pre-installed only | ⚠️ Pre-installed only |
| **S3 access** | ✅ Yes | ✅ Yes (with S3 endpoint) | ✅ Yes |
| **AWS APIs** | ✅ Yes | ⚠️ Limited | ✅ Yes (with endpoints) |
| **External APIs** | ✅ Yes | ❌ No | ❌ No |
| **Session Manager** | ✅ Yes | ✅ Yes (with endpoints) | ✅ Yes |

*With S3 endpoint, you can host internal package repositories

## Quick Start

### 1. Private Subnet with NAT Gateway (Recommended for Production)

```bash
# Preview what will be created
aws-jupyter launch \
  --connection session-manager \
  --subnet-type private \
  --create-nat-gateway \
  --dry-run

# Launch instance
aws-jupyter launch \
  --connection session-manager \
  --subnet-type private \
  --create-nat-gateway \
  --env ml-pytorch \
  --instance-type m7g.xlarge
```

**What happens:**
1. ✅ Creates or selects a private subnet
2. ✅ Creates NAT Gateway in public subnet
3. ✅ Configures route table for internet access
4. ✅ Sets up Session Manager IAM role
5. ✅ Launches instance with no public IP
6. ✅ Instance can install packages via NAT Gateway

### 2. Private Subnet without Internet (Cost-Effective)

```bash
# Preview
aws-jupyter launch \
  --connection session-manager \
  --subnet-type private \
  --dry-run

# Launch
aws-jupyter launch \
  --connection session-manager \
  --subnet-type private \
  --env minimal
```

**What happens:**
1. ✅ Creates or selects a private subnet
2. ✅ Sets up Session Manager IAM role
3. ✅ Launches instance with no public IP
4. ⚠️ No internet access (packages must be pre-installed)

**Important**: Use built-in environments (`minimal`, `data-science`, etc.) that include common packages.

### 3. Connecting to Private Instances

```bash
# List instances
aws-jupyter list

# Connect via Session Manager
aws-jupyter connect i-0abc123def456789

# Port forward Jupyter Lab
aws ssm start-session \
  --target i-0abc123def456789 \
  --document-name AWS-StartPortForwardingSession \
  --parameters '{"portNumber":["8888"],"localPortNumber":["8888"]}'
```

## Cost Analysis

### Monthly Cost Breakdown

**Scenario 1: Small Development (Private + NAT)**
```
Instance: m7g.medium (2 vCPU, 8GB)  = $42/month
NAT Gateway (1 AZ)                   = $32/month
Data Transfer (20GB/month)           = $1/month
EBS Storage (30GB)                   = $3/month
-------------------------------------------------
Total:                               ~$78/month
```

**Scenario 2: Medium Production (Private + NAT)**
```
Instance: m7g.xlarge (4 vCPU, 16GB) = $122/month
NAT Gateway (1 AZ)                   = $32/month
Data Transfer (100GB/month)          = $5/month
EBS Storage (100GB)                  = $10/month
-------------------------------------------------
Total:                               ~$169/month
```

**Scenario 3: Cost-Optimized (Private, No Internet)**
```
Instance: m7g.medium (2 vCPU, 8GB)  = $42/month
NAT Gateway                          = $0/month
Data Transfer                        = $0/month
EBS Storage (30GB)                   = $3/month
-------------------------------------------------
Total:                               ~$45/month
```

**Scenario 4: VPC Endpoints (Middle Ground)**
```
Instance: m7g.medium (2 vCPU, 8GB)  = $42/month
VPC Endpoints (S3, SSM, EC2)         = $21/month
Data Transfer                        = $0/month
EBS Storage (30GB)                   = $3/month
-------------------------------------------------
Total:                               ~$66/month
```

### Cost Optimization Tips

1. **Use Spot Instances** (not yet supported by aws-jupyter, coming in v0.5.0)
   - Save up to 70% on instance costs
   - Good for fault-tolerant workloads

2. **Stop When Not in Use**
   ```bash
   aws-jupyter stop i-0abc123def456789
   ```
   - Only pay for EBS storage when stopped
   - Save ~90% of compute costs

3. **Share NAT Gateway**
   - One NAT Gateway can serve multiple instances
   - aws-jupyter reuses existing NAT Gateways

4. **Right-Size Instances**
   ```bash
   # Start small
   aws-jupyter launch --instance-type m7g.medium

   # Scale up if needed
   # (Manual resize via AWS Console for now)
   ```

5. **Use Minimal Environments**
   - Install only required packages
   - Reduce data transfer costs

## VPC Endpoints Alternative

### When to Use VPC Endpoints

Use VPC endpoints instead of NAT Gateway when:

✅ You primarily access AWS services (S3, DynamoDB, etc.)
✅ You don't need PyPI or external package repositories
✅ You want to minimize costs (endpoints cheaper than NAT)
✅ You have predictable, AWS-focused workloads

### Setting Up VPC Endpoints

#### Required Endpoints for Session Manager

```bash
# Get your VPC and route table IDs
VPC_ID=$(aws ec2 describe-vpcs \
  --filters "Name=tag:Name,Values=aws-jupyter-vpc" \
  --query 'Vpcs[0].VpcId' --output text)

ROUTE_TABLE_ID=$(aws ec2 describe-route-tables \
  --filters "Name=vpc-id,Values=$VPC_ID" "Name=tag:Name,Values=*private*" \
  --query 'RouteTables[0].RouteTableId' --output text)

REGION="us-west-2"  # Change to your region

# 1. SSM Endpoint (required for Session Manager)
aws ec2 create-vpc-endpoint \
  --vpc-id $VPC_ID \
  --service-name com.amazonaws.$REGION.ssm \
  --route-table-ids $ROUTE_TABLE_ID \
  --vpc-endpoint-type Interface

# 2. SSM Messages Endpoint (required for Session Manager)
aws ec2 create-vpc-endpoint \
  --vpc-id $VPC_ID \
  --service-name com.amazonaws.$REGION.ssmmessages \
  --route-table-ids $ROUTE_TABLE_ID \
  --vpc-endpoint-type Interface

# 3. EC2 Messages Endpoint (required for Session Manager)
aws ec2 create-vpc-endpoint \
  --vpc-id $VPC_ID \
  --service-name com.amazonaws.$REGION.ec2messages \
  --route-table-ids $ROUTE_TABLE_ID \
  --vpc-endpoint-type Interface

# 4. S3 Endpoint (optional, for S3 access)
aws ec2 create-vpc-endpoint \
  --vpc-id $VPC_ID \
  --service-name com.amazonaws.$REGION.s3 \
  --route-table-ids $ROUTE_TABLE_ID \
  --vpc-endpoint-type Gateway
```

#### Optional Endpoints for Additional Services

```bash
# CloudWatch Logs (for log shipping)
aws ec2 create-vpc-endpoint \
  --vpc-id $VPC_ID \
  --service-name com.amazonaws.$REGION.logs \
  --vpc-endpoint-type Interface

# ECR (for Docker images)
aws ec2 create-vpc-endpoint \
  --vpc-id $VPC_ID \
  --service-name com.amazonaws.$REGION.ecr.api \
  --vpc-endpoint-type Interface

# DynamoDB (for data access)
aws ec2 create-vpc-endpoint \
  --vpc-id $VPC_ID \
  --service-name com.amazonaws.$REGION.dynamodb \
  --route-table-ids $ROUTE_TABLE_ID \
  --vpc-endpoint-type Gateway
```

### VPC Endpoint Costs

| Endpoint Type | Cost per Hour | Monthly Cost |
|---------------|---------------|--------------|
| **Interface** | $0.01/hour | ~$7/month |
| **Gateway** (S3, DynamoDB) | Free | $0 |
| **Data Transfer** | $0.01/GB | Variable |

**Example Monthly Cost (Minimal Setup):**
- SSM: $7/month
- SSM Messages: $7/month
- EC2 Messages: $7/month
- S3 (Gateway): $0/month
- **Total: ~$21/month**

**vs NAT Gateway: $32-50/month**

## Security Best Practices

### 1. Network Segmentation

```bash
# Development instances in one subnet
aws-jupyter launch \
  --subnet-type private \
  --create-nat-gateway \
  --tags "Environment=dev"

# Production instances in another subnet
aws-jupyter launch \
  --subnet-type private \
  --create-nat-gateway \
  --tags "Environment=prod"
```

### 2. Least Privilege IAM

Use specific IAM policies instead of broad permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject"
      ],
      "Resource": "arn:aws:s3:::my-data-bucket/*"
    }
  ]
}
```

### 3. Enable VPC Flow Logs

```bash
# Create CloudWatch log group
aws logs create-log-group --log-group-name /aws/vpc/flowlogs

# Enable VPC Flow Logs
aws ec2 create-flow-logs \
  --resource-type VPC \
  --resource-ids $VPC_ID \
  --traffic-type ALL \
  --log-destination-type cloud-watch-logs \
  --log-group-name /aws/vpc/flowlogs
```

### 4. Security Group Restrictions

Private instances don't need inbound rules (Session Manager uses outbound only):

```bash
# View security group rules
aws ec2 describe-security-groups \
  --filters "Name=group-name,Values=aws-jupyter-session-manager"

# No inbound rules required for Session Manager!
```

### 5. Regular Security Audits

```bash
# List all instances
aws-jupyter list

# Check instance details
aws-jupyter status i-0abc123def456789

# Review CloudTrail logs
aws cloudtrail lookup-events \
  --lookup-attributes AttributeKey=ResourceType,AttributeValue=AWS::EC2::Instance
```

## Troubleshooting

### Session Manager Can't Connect

**Problem**: `TargetNotConnected` error when trying to connect

**Solutions:**

1. **Check SSM Agent Status**
   ```bash
   aws ssm describe-instance-information \
     --filters "Key=InstanceIds,Values=i-0abc123def456789"
   ```

2. **Verify Internet/Endpoint Access**
   - With NAT Gateway: Check NAT Gateway state
   - Without NAT: Verify VPC endpoints exist
   ```bash
   aws ec2 describe-nat-gateways --filter "Name=vpc-id,Values=$VPC_ID"
   aws ec2 describe-vpc-endpoints --filters "Name=vpc-id,Values=$VPC_ID"
   ```

3. **Check IAM Instance Profile**
   ```bash
   aws-jupyter status i-0abc123def456789
   # Look for "IamInstanceProfile" in output
   ```

### Can't Install Packages

**Problem**: `pip install` or `apt-get install` fails

**Solutions:**

1. **Check Internet Access**
   ```bash
   # Connect to instance
   aws-jupyter connect i-0abc123def456789

   # Test internet connectivity
   curl -I https://pypi.org
   ```

2. **If No Internet Access**
   - Option A: Add NAT Gateway
   - Option B: Use pre-configured environment
   - Option C: Set up internal package mirror with S3 endpoint

3. **Use Pre-Installed Packages**
   ```bash
   # Launch with comprehensive environment
   aws-jupyter launch \
     --subnet-type private \
     --env deep-learning  # Has many packages pre-installed
   ```

### High Costs

**Problem**: NAT Gateway costs too high

**Solutions:**

1. **Switch to VPC Endpoints**
   - If you mainly use AWS services
   - Can save $10-30/month

2. **Stop Instances When Not in Use**
   ```bash
   aws-jupyter stop i-0abc123def456789
   ```
   NAT Gateway charges continue, but instance costs stop

3. **Use No-Internet Option**
   ```bash
   aws-jupyter launch --subnet-type private  # No --create-nat-gateway
   ```

4. **Share NAT Gateway**
   - aws-jupyter automatically reuses existing NAT Gateways
   - One NAT Gateway can serve many instances

## Migration from Public Subnets

### Step 1: Evaluate Current Setup

```bash
# List current instances
aws-jupyter list

# Check instance details
aws-jupyter status i-0abc123def456789
```

### Step 2: Launch New Private Instance

```bash
# Launch replacement in private subnet
aws-jupyter launch \
  --connection session-manager \
  --subnet-type private \
  --create-nat-gateway \
  --env your-environment \
  --instance-type your-instance-type
```

### Step 3: Migrate Data

```bash
# Option A: Use S3 as intermediate
# From old instance:
aws s3 cp /path/to/notebooks s3://my-bucket/backup/ --recursive

# To new instance:
aws s3 cp s3://my-bucket/backup/ /path/to/notebooks --recursive

# Option B: Create AMI and launch from it
aws ec2 create-image --instance-id i-old-instance --name "migration-backup"
```

### Step 4: Test New Instance

```bash
# Connect to new instance
aws-jupyter connect i-new-instance

# Verify notebooks and data
ls -lah /home/ubuntu/notebooks

# Test Jupyter Lab
sudo systemctl status jupyter
```

### Step 5: Terminate Old Instance

```bash
# Once verified, terminate old instance
aws-jupyter terminate i-old-instance
```

## Additional Resources

**AWS Documentation:**
- [VPC Endpoints](https://docs.aws.amazon.com/vpc/latest/privatelink/vpc-endpoints.html)
- [NAT Gateways](https://docs.aws.amazon.com/vpc/latest/userguide/vpc-nat-gateway.html)
- [Private Subnets](https://docs.aws.amazon.com/vpc/latest/userguide/configure-subnets.html)

**aws-jupyter Documentation:**
- [Main README](../README.md)
- [Session Manager Setup](SESSION_MANAGER_SETUP.md)
- [Troubleshooting Guide](TROUBLESHOOTING.md)
- [Examples & Use Cases](EXAMPLES.md)

## Summary

**Quick Decision Guide:**

| Your Need | Recommended Option | Monthly Cost |
|-----------|-------------------|--------------|
| **Production + Packages** | Private + NAT Gateway | ~$95 |
| **Production + AWS Only** | Private + VPC Endpoints | ~$67 |
| **Cost-Sensitive** | Private + No Internet | ~$60 |
| **Development** | Public Subnet | ~$60 |

**Best Practices:**
- ✅ Use private subnets for production
- ✅ Use Session Manager (no SSH keys)
- ✅ Stop instances when not in use
- ✅ Share NAT Gateways across instances
- ✅ Enable VPC Flow Logs
- ✅ Review costs monthly

---

**Next Steps:**
- Set up [Session Manager](SESSION_MANAGER_SETUP.md)
- Explore [real-world examples](EXAMPLES.md)
- Review [troubleshooting tips](TROUBLESHOOTING.md)
