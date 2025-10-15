# AWS Session Manager Setup Guide

This guide covers everything you need to know about using AWS Session Manager with aws-jupyter for secure, SSH-key-free access to your Jupyter Lab instances.

## Table of Contents

- [What is Session Manager?](#what-is-session-manager)
- [Prerequisites](#prerequisites)
- [Installing Session Manager Plugin](#installing-session-manager-plugin)
- [IAM Permissions](#iam-permissions)
- [Quick Start](#quick-start)
- [Connection Methods](#connection-methods)
- [Troubleshooting](#troubleshooting)
- [Advanced Configuration](#advanced-configuration)
- [Security Considerations](#security-considerations)

## What is Session Manager?

AWS Systems Manager Session Manager is a fully managed AWS service that provides secure and auditable instance management without the need to:
- Manage SSH keys
- Open inbound ports
- Maintain bastion hosts
- Configure VPN connections

### Benefits

**Security**
- No SSH keys to manage or rotate
- No inbound security group rules required
- All connections encrypted with TLS 1.2+
- Complete audit trail in CloudTrail

**Convenience**
- Works from anywhere with AWS API access
- No direct internet connectivity required on instances
- Browser-based or CLI access
- Port forwarding for Jupyter Lab

**Compliance**
- Centralized access control via IAM
- Session recording and logging
- Integration with AWS CloudWatch
- MFA support for additional security

## Prerequisites

### 1. AWS CLI

Session Manager requires AWS CLI v2 or higher:

```bash
# Check AWS CLI version
aws --version

# Should show: aws-cli/2.x.x or higher
```

If you need to install or upgrade:
- **macOS**: `brew install awscli`
- **Linux**: [AWS CLI Installation Guide](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)
- **Windows**: [AWS CLI MSI Installer](https://awscli.amazonaws.com/AWSCLIV2.msi)

### 2. AWS Credentials

Configure AWS credentials with appropriate permissions:

```bash
# Configure default profile
aws configure

# Or configure a named profile
aws configure --profile myprofile

# Verify access
aws sts get-caller-identity --profile myprofile
```

### 3. IAM Permissions

Your IAM user/role needs these permissions:

**For launching instances with Session Manager**:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:RunInstances",
        "ec2:DescribeInstances",
        "iam:CreateRole",
        "iam:AttachRolePolicy",
        "iam:CreateInstanceProfile",
        "iam:AddRoleToInstanceProfile",
        "iam:GetRole",
        "iam:GetInstanceProfile",
        "iam:PassRole"
      ],
      "Resource": "*"
    }
  ]
}
```

**For connecting to instances**:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ssm:StartSession",
        "ssm:TerminateSession",
        "ssm:ResumeSession",
        "ssm:DescribeSessions",
        "ssm:GetConnectionStatus"
      ],
      "Resource": "*"
    }
  ]
}
```

**Recommended**: Use the AWS managed policy `AmazonSSMManagedInstanceCore` for the instance role.

## Installing Session Manager Plugin

The Session Manager plugin is required for CLI-based connections.

### macOS

```bash
# Using Homebrew
brew install --cask session-manager-plugin

# Verify installation
session-manager-plugin --version
```

### Linux

```bash
# Download the plugin (64-bit)
curl "https://s3.amazonaws.com/session-manager-downloads/plugin/latest/ubuntu_64bit/session-manager-plugin.deb" -o "session-manager-plugin.deb"

# Install
sudo dpkg -i session-manager-plugin.deb

# Verify installation
session-manager-plugin --version
```

For other Linux distributions:
- **Amazon Linux/RHEL/CentOS**: Use `.rpm` instead of `.deb`
- **Ubuntu/Debian**: Use the `.deb` package as shown above

### Windows

```powershell
# Download the installer
Invoke-WebRequest -Uri "https://s3.amazonaws.com/session-manager-downloads/plugin/latest/windows/SessionManagerPluginSetup.exe" -OutFile "SessionManagerPluginSetup.exe"

# Run the installer
.\SessionManagerPluginSetup.exe

# Verify installation
session-manager-plugin
```

### Verification

After installation, verify the plugin works:

```bash
session-manager-plugin

# Expected output:
# The Session Manager plugin was installed successfully. Use the AWS CLI to start a session.
```

## Quick Start

### 1. Launch Instance with Session Manager

```bash
# Basic launch with Session Manager
aws-jupyter launch --connection session-manager

# Preview what will be created
aws-jupyter launch --connection session-manager --dry-run

# With custom environment and instance type
aws-jupyter launch \
  --connection session-manager \
  --env ml-pytorch \
  --instance-type m7g.xlarge
```

### 2. Check Instance Status

```bash
# List all instances
aws-jupyter list

# Get detailed status
aws-jupyter status i-0abc123def456789
```

### 3. Connect to Instance

```bash
# Using aws-jupyter (recommended)
aws-jupyter connect i-0abc123def456789

# Using AWS CLI directly
aws ssm start-session --target i-0abc123def456789 --profile myprofile
```

## Connection Methods

### Interactive Shell

Start an interactive shell session:

```bash
# Using aws-jupyter
aws-jupyter connect i-0abc123def456789

# Using AWS CLI
aws ssm start-session --target i-0abc123def456789
```

Once connected, you can:
```bash
# Check Jupyter Lab status
sudo systemctl status jupyter

# View Jupyter logs
sudo journalctl -u jupyter -f

# Access Jupyter token
sudo jupyter server list
```

### Port Forwarding

Forward Jupyter Lab port (8888) to your local machine:

```bash
# Forward port 8888 to localhost:8888
aws ssm start-session \
  --target i-0abc123def456789 \
  --document-name AWS-StartPortForwardingSession \
  --parameters '{"portNumber":["8888"],"localPortNumber":["8888"]}'

# Now access Jupyter Lab at: http://localhost:8888
```

**Tip**: Get the Jupyter token after connecting via interactive shell:
```bash
aws-jupyter connect i-0abc123def456789
# In the instance shell:
sudo jupyter server list
```

### Port Forwarding to Remote Host

Forward through the instance to another host:

```bash
# Forward to database on another host
aws ssm start-session \
  --target i-0abc123def456789 \
  --document-name AWS-StartPortForwardingSessionToRemoteHost \
  --parameters '{"host":["db.internal.example.com"],"portNumber":["5432"],"localPortNumber":["5432"]}'
```

## Troubleshooting

### Plugin Not Found Error

**Error**: `SessionManagerPlugin is not found. Please refer to SessionManager Documentation here`

**Solution**:
```bash
# Verify plugin is installed
which session-manager-plugin

# If not found, install using instructions above

# macOS: May need to add to PATH
export PATH=$PATH:/usr/local/sessionmanagerplugin/bin
```

### Permission Denied

**Error**: `An error occurred (AccessDeniedException) when calling the StartSession operation`

**Solution**:
1. Verify your IAM permissions include `ssm:StartSession`
2. Check the instance has the Session Manager role attached:
   ```bash
   aws-jupyter status i-0abc123def456789
   ```
3. Ensure the instance is in the "running" state

### Target Not Connected

**Error**: `TargetNotConnected: i-0abc123def is not connected`

**Possible Causes**:
1. **SSM Agent not running**: Wait a few minutes after instance launch
2. **No internet access**: Instance needs internet to reach SSM endpoints
3. **IAM role missing**: Instance needs the Session Manager role

**Solution**:
```bash
# Check instance status
aws-jupyter status i-0abc123def456789

# Verify SSM agent connectivity
aws ssm describe-instance-information \
  --filters "Key=InstanceIds,Values=i-0abc123def456789"

# If using private subnet without NAT gateway:
# The instance needs VPC endpoints for SSM, or internet access
```

### SSM Agent Not Running

**Symptoms**: Instance shows "running" but Session Manager can't connect

**Solution**:
1. Wait 2-3 minutes after launch for SSM agent to initialize
2. Connect via SSH (if available) and check agent:
   ```bash
   sudo systemctl status amazon-ssm-agent
   ```
3. Restart agent if needed:
   ```bash
   sudo systemctl restart amazon-ssm-agent
   ```

### VPC Endpoint Issues (Private Subnets)

If using private subnets without NAT gateway, you need VPC endpoints:

```bash
# Required endpoints for Session Manager:
# - com.amazonaws.region.ssm
# - com.amazonaws.region.ssmmessages
# - com.amazonaws.region.ec2messages

# Create endpoints using AWS CLI:
aws ec2 create-vpc-endpoint \
  --vpc-id vpc-12345 \
  --service-name com.amazonaws.us-west-2.ssm \
  --route-table-ids rtb-12345
```

**Tip**: aws-jupyter automatically configures Session Manager when you use `--create-nat-gateway`, which is easier than managing VPC endpoints.

## Advanced Configuration

### Session Preferences

Configure session preferences for logging and encryption:

```bash
# Create Session Manager preferences (optional)
aws ssm create-document \
  --content file://session-prefs.json \
  --name "SessionPreferences" \
  --document-type "Session"
```

Example `session-prefs.json`:
```json
{
  "schemaVersion": "1.0",
  "description": "Document to hold regional settings for Session Manager",
  "sessionType": "Standard_Stream",
  "inputs": {
    "s3BucketName": "my-session-logs-bucket",
    "s3KeyPrefix": "session-logs/",
    "s3EncryptionEnabled": true,
    "cloudWatchLogGroupName": "/aws/ssm/sessions",
    "cloudWatchEncryptionEnabled": true,
    "kmsKeyId": "alias/MySessionEncryptionKey"
  }
}
```

### MFA Enforcement

Require MFA for Session Manager access:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "ssm:StartSession",
      "Resource": "*",
      "Condition": {
        "BoolIfExists": {
          "aws:MultiFactorAuthPresent": "true"
        }
      }
    }
  ]
}
```

### Restrict to Specific Instances

Limit Session Manager access to specific instances:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "ssm:StartSession",
      "Resource": [
        "arn:aws:ec2:us-west-2:123456789012:instance/i-0abc123def456789",
        "arn:aws:ssm:us-west-2:123456789012:document/AWS-StartPortForwardingSession"
      ]
    }
  ]
}
```

### Custom Shell Profiles

Create custom shell settings for Session Manager sessions:

```bash
# On the instance, create /etc/profile.d/session-manager.sh
sudo tee /etc/profile.d/session-manager.sh > /dev/null <<EOF
# Custom Session Manager profile
export PS1='\[\033[01;32m\]\u@\h\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]\$ '
alias ll='ls -lah'
alias jupyter-logs='sudo journalctl -u jupyter -f'
alias jupyter-status='sudo systemctl status jupyter'

echo "=== AWS Jupyter Lab Instance ==="
echo "Jupyter Lab: http://localhost:8888"
echo "Get token: sudo jupyter server list"
echo "=========================="
EOF
```

## Security Considerations

### Network Security

**Session Manager doesn't require**:
- Open inbound ports (no port 22/SSH or port 8888/Jupyter)
- Public IP addresses (works in private subnets)
- Bastion hosts or jump boxes

**What it does require**:
- Outbound HTTPS (443) access to AWS SSM endpoints
- IAM-based authentication
- Instance IAM role with Session Manager permissions

### Audit and Compliance

**CloudTrail Logging**: All Session Manager activity is logged:
```bash
# View Session Manager events
aws cloudtrail lookup-events \
  --lookup-attributes AttributeKey=EventName,AttributeValue=StartSession
```

**Session History**:
```bash
# List all sessions
aws ssm describe-sessions --state History

# Get specific session details
aws ssm describe-sessions --filters "key=SessionId,value=session-id"
```

### Best Practices

1. **Use IAM Policies**: Grant minimal required permissions
2. **Enable MFA**: Require MFA for production access
3. **Enable Logging**: Send session logs to S3 and CloudWatch
4. **Tag Resources**: Use tags for access control policies
5. **Regular Audits**: Review Session Manager usage in CloudTrail
6. **Instance Isolation**: Use private subnets for sensitive workloads
7. **Encryption**: Enable encryption for session logs

### Comparison with SSH

| Feature | Session Manager | SSH |
|---------|----------------|-----|
| **Key Management** | Not required | Required |
| **Inbound Ports** | Not required | Port 22 must be open |
| **Bastion Hosts** | Not required | Often required |
| **Audit Logging** | Built-in CloudTrail | Requires setup |
| **MFA Support** | Native IAM MFA | Requires additional setup |
| **Port Forwarding** | Supported | Supported |
| **Browser Access** | Yes (AWS Console) | No |
| **Works in Private Subnet** | Yes | Requires bastion or VPN |

## Additional Resources

**AWS Documentation**:
- [Session Manager User Guide](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager.html)
- [Session Manager Prerequisites](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-prerequisites.html)
- [Port Forwarding Guide](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-sessions-start.html#sessions-start-port-forwarding)

**aws-jupyter Documentation**:
- [Main README](../README.md)
- [Private Subnet Guide](PRIVATE_SUBNET_GUIDE.md)
- [Troubleshooting Guide](TROUBLESHOOTING.md)
- [Examples & Use Cases](EXAMPLES.md)

## Getting Help

If you encounter issues:

1. Check the [Troubleshooting Guide](TROUBLESHOOTING.md)
2. Review [AWS Session Manager documentation](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager.html)
3. Open an issue on [GitHub](https://github.com/scttfrdmn/aws-jupyter/issues)

---

**Next Steps**:
- Learn about [Private Subnet deployments](PRIVATE_SUBNET_GUIDE.md)
- Explore [real-world examples](EXAMPLES.md)
- Review [troubleshooting tips](TROUBLESHOOTING.md)
