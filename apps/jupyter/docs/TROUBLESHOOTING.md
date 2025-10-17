# Troubleshooting Guide

Comprehensive troubleshooting guide for common issues with aws-jupyter.

## Table of Contents

- [Installation Issues](#installation-issues)
- [AWS Authentication](#aws-authentication)
- [Instance Launch Problems](#instance-launch-problems)
- [Connection Issues](#connection-issues)
- [Session Manager Problems](#session-manager-problems)
- [SSH Connection Issues](#ssh-connection-issues)
- [Networking Problems](#networking-problems)
- [Jupyter Lab Issues](#jupyter-lab-issues)
- [Permission Errors](#permission-errors)
- [Cost and Billing](#cost-and-billing)
- [Performance Issues](#performance-issues)
- [Getting Help](#getting-help)

## Installation Issues

### Command Not Found: aws-jupyter

**Problem:**
```bash
$ aws-jupyter launch
bash: aws-jupyter: command not found
```

**Solutions:**

1. **Check if Go is installed:**
   ```bash
   go version
   # Should show: go version go1.22 or higher
   ```

2. **Verify GOPATH/GOBIN is in PATH:**
   ```bash
   echo $PATH | grep go

   # Add to ~/.bashrc or ~/.zshrc if missing:
   export PATH=$PATH:$(go env GOPATH)/bin
   ```

3. **Reinstall aws-jupyter:**
   ```bash
   go install github.com/scttfrdmn/aws-jupyter@latest
   ```

4. **Check installation location:**
   ```bash
   which aws-jupyter
   # Should show: /Users/yourname/go/bin/aws-jupyter or similar
   ```

### Go Version Too Old

**Problem:**
```
go: github.com/scttfrdmn/aws-jupyter requires go >= 1.22
```

**Solution:**
```bash
# macOS
brew upgrade go

# Linux (download from https://go.dev/dl/)
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz

# Verify
go version
```

### Module Download Issues

**Problem:**
```
go: github.com/scttfrdmn/aws-jupyter@latest: reading https://proxy.golang.org/...
dial tcp: lookup proxy.golang.org: no such host
```

**Solution:**
```bash
# Check internet connectivity
ping 8.8.8.8

# Try setting Go proxy
export GOPROXY=https://proxy.golang.org,direct

# Or use direct mode (slower)
export GOPROXY=direct
```

## AWS Authentication

### No AWS Credentials Found

**Problem:**
```
Error: NoCredentialProviders: no valid providers in chain
```

**Solutions:**

1. **Configure AWS CLI:**
   ```bash
   aws configure --profile myprofile
   # Enter: Access Key ID, Secret Access Key, Region, Output format
   ```

2. **Verify credentials work:**
   ```bash
   aws sts get-caller-identity --profile myprofile
   ```

3. **Use environment variables:**
   ```bash
   export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
   export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
   export AWS_REGION=us-west-2
   ```

4. **Check AWS config files:**
   ```bash
   cat ~/.aws/credentials
   cat ~/.aws/config
   ```

### Invalid AWS Profile

**Problem:**
```
Error: SharedConfigProfileNotExist: failed to get profile, myprofile
```

**Solution:**
```bash
# List available profiles
aws configure list-profiles

# Create the profile
aws configure --profile myprofile

# Or edit config file directly
vim ~/.aws/config
```

### Expired AWS Session Token

**Problem:**
```
Error: ExpiredToken: The security token included in the request is expired
```

**Solution:**
```bash
# For AWS SSO:
aws sso login --profile myprofile

# For temporary credentials:
# Re-run your credential generation command (e.g., assume-role)

# Verify new credentials
aws sts get-caller-identity --profile myprofile
```

### MFA Required

**Problem:**
```
Error: AccessDenied: MultiFactorAuthentication required
```

**Solution:**
```bash
# Use AWS CLI with MFA
aws sts get-session-token \
  --serial-number arn:aws:iam::123456789012:mfa/username \
  --token-code 123456

# Export the temporary credentials from the output
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
export AWS_SESSION_TOKEN=...
```

## Instance Launch Problems

### Insufficient Capacity

**Problem:**
```
Error: InsufficientInstanceCapacity: We currently do not have sufficient m7g.xlarge capacity
```

**Solutions:**

1. **Try different instance type:**
   ```bash
   aws-jupyter launch --instance-type m7g.large
   ```

2. **Try different availability zone:**
   ```bash
   aws-jupyter launch --region us-west-2
   # Or wait and retry in current region
   ```

3. **Use different generation:**
   ```bash
   # Instead of m7g, try m6g
   aws-jupyter launch --instance-type m6g.xlarge
   ```

### AMI Not Found

**Problem:**
```
Error: InvalidAMIID.NotFound: The image id '[ami-xxxxx]' does not exist
```

**Solutions:**

1. **Check region:**
   ```bash
   # AMI IDs are region-specific
   aws-jupyter launch --region us-west-2
   ```

2. **Update aws-jupyter:**
   ```bash
   go install github.com/scttfrdmn/aws-jupyter@latest
   ```

3. **Find correct AMI:**
   ```bash
   aws ec2 describe-images \
     --owners amazon \
     --filters "Name=name,Values=ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-arm64-*" \
     --query 'Images | sort_by(@, &CreationDate) | [-1].ImageId'
   ```

### VPC Not Found

**Problem:**
```
Error: VPCIdNotSpecified: No default VPC for this user
```

**Solution:**

1. **Create default VPC:**
   ```bash
   aws ec2 create-default-vpc
   ```

2. **Or specify existing VPC:**
   ```bash
   # Find available VPCs
   aws ec2 describe-vpcs

   # aws-jupyter will auto-detect VPCs
   # Or create one manually and it will be used
   ```

### Subnet Exhaustion

**Problem:**
```
Error: InsufficientFreeAddressesInSubnet: There are not enough free addresses
```

**Solution:**

1. **Use different availability zone:**
   ```bash
   # aws-jupyter will try another subnet automatically
   aws-jupyter launch
   ```

2. **Create new subnet:**
   ```bash
   aws ec2 create-subnet \
     --vpc-id vpc-12345 \
     --cidr-block 10.0.3.0/24 \
     --availability-zone us-west-2c
   ```

### Quota Exceeded

**Problem:**
```
Error: InstanceLimitExceeded: You have requested more instances than your current instance limit
```

**Solution:**

1. **Check current limits:**
   ```bash
   aws service-quotas get-service-quota \
     --service-code ec2 \
     --quota-code L-1216C47A
   ```

2. **Request limit increase:**
   - Go to AWS Service Quotas console
   - Search for "EC2 instances"
   - Request increase

3. **Terminate unused instances:**
   ```bash
   aws-jupyter list
   aws-jupyter terminate i-old-instance
   ```

## Connection Issues

### Cannot Connect to Instance

**Problem:** Can't connect to instance after launch

**Checklist:**

1. **Verify instance is running:**
   ```bash
   aws-jupyter list
   aws-jupyter status i-0abc123def456789
   ```

2. **Check connection method:**
   ```bash
   # For Session Manager:
   aws-jupyter connect i-0abc123def456789

   # For SSH:
   ssh -i ~/.aws-jupyter/keys/aws-jupyter-us-west-2.pem ubuntu@<public-ip>
   ```

3. **Wait for initialization:**
   ```bash
   # Instances need 2-3 minutes after launch
   # for SSM agent to start and user data to run
   ```

4. **Check security groups:**
   ```bash
   aws ec2 describe-instances --instance-ids i-0abc123def456789 \
     --query 'Reservations[0].Instances[0].SecurityGroups'
   ```

## Session Manager Problems

### Session Manager Plugin Not Found

**Problem:**
```
SessionManagerPlugin is not found. Please refer to SessionManager Documentation
```

**Solution:**

1. **Install plugin (macOS):**
   ```bash
   brew install --cask session-manager-plugin
   ```

2. **Install plugin (Linux):**
   ```bash
   curl "https://s3.amazonaws.com/session-manager-downloads/plugin/latest/ubuntu_64bit/session-manager-plugin.deb" -o "session-manager-plugin.deb"
   sudo dpkg -i session-manager-plugin.deb
   ```

3. **Verify installation:**
   ```bash
   session-manager-plugin
   # Should show: "The Session Manager plugin was installed successfully..."
   ```

4. **Add to PATH (if needed):**
   ```bash
   export PATH=$PATH:/usr/local/sessionmanagerplugin/bin
   ```

### Target Not Connected

**Problem:**
```
Error: TargetNotConnected: i-0abc123def456789 is not connected
```

**Solutions:**

1. **Wait for SSM agent:**
   ```bash
   # Wait 2-3 minutes after launch, then check:
   aws ssm describe-instance-information \
     --filters "Key=InstanceIds,Values=i-0abc123def456789"
   ```

2. **Check instance has internet access:**
   ```bash
   # For Session Manager, instance needs:
   # - NAT Gateway (private subnet), OR
   # - Internet Gateway (public subnet), OR
   # - VPC endpoints (ssm, ssmmessages, ec2messages)
   ```

3. **Verify IAM instance profile:**
   ```bash
   aws-jupyter status i-0abc123def456789
   # Look for IamInstanceProfile
   ```

4. **Check SSM agent logs (if you can SSH):**
   ```bash
   ssh -i ~/.aws-jupyter/keys/... ubuntu@<ip>
   sudo journalctl -u amazon-ssm-agent -f
   ```

### Access Denied (Session Manager)

**Problem:**
```
Error: AccessDeniedException: User is not authorized to perform: ssm:StartSession
```

**Solution:**

Add IAM permissions:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ssm:StartSession",
        "ssm:TerminateSession"
      ],
      "Resource": "*"
    }
  ]
}
```

## SSH Connection Issues

### Permission Denied (publickey)

**Problem:**
```
Permission denied (publickey)
```

**Solutions:**

1. **Check key permissions:**
   ```bash
   aws-jupyter key validate

   # Or manually:
   chmod 600 ~/.aws-jupyter/keys/*.pem
   ```

2. **Verify correct key:**
   ```bash
   aws-jupyter key list
   aws-jupyter key show
   ```

3. **Use correct username:**
   ```bash
   # For Ubuntu AMIs:
   ssh -i ~/.aws-jupyter/keys/aws-jupyter-us-west-2.pem ubuntu@<ip>

   # NOT: ec2-user, admin, or root
   ```

4. **Check key is registered in AWS:**
   ```bash
   aws ec2 describe-key-pairs --region us-west-2
   ```

### Connection Timeout

**Problem:**
```
ssh: connect to host <ip> port 22: Operation timed out
```

**Solutions:**

1. **Check security group allows SSH:**
   ```bash
   aws ec2 describe-security-groups \
     --group-ids sg-12345 \
     --query 'SecurityGroups[0].IpPermissions'

   # Should show port 22 open to your IP
   ```

2. **Verify your public IP:**
   ```bash
   curl https://checkip.amazonaws.com

   # Compare with security group rules
   ```

3. **Check instance has public IP:**
   ```bash
   aws-jupyter status i-0abc123def456789
   # Look for PublicIpAddress
   ```

4. **Try Session Manager instead:**
   ```bash
   aws-jupyter connect i-0abc123def456789
   ```

### Key Not Found

**Problem:**
```
Error: Unable to load AWS SSH key from /Users/name/.aws-jupyter/keys/aws-jupyter-us-west-2.pem
```

**Solutions:**

1. **Check key exists:**
   ```bash
   aws-jupyter key list
   ls -l ~/.aws-jupyter/keys/
   ```

2. **Download key from AWS:**
   ```bash
   # Keys can't be re-downloaded from AWS
   # You'll need to create a new one
   ```

3. **Launch with new key:**
   ```bash
   # aws-jupyter will create new key automatically
   aws-jupyter launch --connection ssh
   ```

4. **Specify existing key:**
   ```bash
   aws-jupyter launch --key-name my-existing-key
   ```

## Networking Problems

### NAT Gateway Not Working

**Problem:** Private instance can't access internet despite NAT Gateway

**Solutions:**

1. **Verify NAT Gateway state:**
   ```bash
   aws ec2 describe-nat-gateways \
     --filter "Name=state,Values=available"
   ```

2. **Check route table:**
   ```bash
   # Private subnet route table should have:
   # 0.0.0.0/0 -> nat-gateway-id

   aws ec2 describe-route-tables \
     --filters "Name=vpc-id,Values=vpc-12345"
   ```

3. **Test from instance:**
   ```bash
   aws-jupyter connect i-0abc123def456789

   # In instance:
   curl -I https://pypi.org
   # Should succeed if NAT Gateway working
   ```

4. **Verify subnet association:**
   ```bash
   aws ec2 describe-subnets --subnet-ids subnet-12345 \
     --query 'Subnets[0].RouteTableId'
   ```

### VPC Endpoint Issues

**Problem:** Can't connect via Session Manager in private subnet

**Solutions:**

1. **Check required endpoints exist:**
   ```bash
   aws ec2 describe-vpc-endpoints \
     --filters "Name=vpc-id,Values=vpc-12345" \
     --query 'VpcEndpoints[*].[ServiceName,State]'

   # Required for Session Manager:
   # - com.amazonaws.REGION.ssm
   # - com.amazonaws.REGION.ssmmessages
   # - com.amazonaws.REGION.ec2messages
   ```

2. **Create missing endpoints:**
   ```bash
   # See PRIVATE_SUBNET_GUIDE.md for complete instructions
   aws ec2 create-vpc-endpoint \
     --vpc-id vpc-12345 \
     --service-name com.amazonaws.us-west-2.ssm \
     --vpc-endpoint-type Interface
   ```

3. **Verify security group allows HTTPS:**
   ```bash
   # VPC endpoints need port 443 outbound
   ```

### Cannot Install Packages (Private Subnet)

**Problem:**
```bash
pip install numpy
# ERROR: Could not find a version that satisfies the requirement
```

**Solutions:**

1. **Add NAT Gateway:**
   ```bash
   aws-jupyter launch \
     --subnet-type private \
     --create-nat-gateway
   ```

2. **Use pre-configured environment:**
   ```bash
   # Launch with environment that has packages
   aws-jupyter launch \
     --subnet-type private \
     --env ml-pytorch  # Has PyTorch, numpy, etc.
   ```

3. **Set up S3 package repository:**
   ```bash
   # Host PyPI mirror in S3
   # Configure pip to use S3 endpoint
   pip install --index-url https://my-bucket.s3.amazonaws.com/pypi/simple numpy
   ```

## Jupyter Lab Issues

### Cannot Access Jupyter Lab

**Problem:** Browser can't connect to `http://localhost:8888`

**Solutions:**

1. **Verify Jupyter is running:**
   ```bash
   aws-jupyter connect i-0abc123def456789

   # In instance:
   sudo systemctl status jupyter
   sudo journalctl -u jupyter -f
   ```

2. **Check port forwarding:**
   ```bash
   # For Session Manager:
   aws ssm start-session \
     --target i-0abc123def456789 \
     --document-name AWS-StartPortForwardingSession \
     --parameters '{"portNumber":["8888"],"localPortNumber":["8888"]}'

   # For SSH:
   ssh -i ~/.aws-jupyter/keys/aws-jupyter-us-west-2.pem \
     -L 8888:localhost:8888 \
     ubuntu@<public-ip>
   ```

3. **Get Jupyter token:**
   ```bash
   aws-jupyter connect i-0abc123def456789
   sudo jupyter server list
   # Copy the token from output
   ```

4. **Check Jupyter config:**
   ```bash
   cat ~/.jupyter/jupyter_server_config.py
   # Should have: c.ServerApp.ip = '0.0.0.0'
   ```

### Jupyter Lab Won't Start

**Problem:** Jupyter service fails to start

**Solutions:**

1. **Check service status:**
   ```bash
   sudo systemctl status jupyter
   sudo journalctl -u jupyter -n 50
   ```

2. **Check Python environment:**
   ```bash
   which jupyter
   jupyter --version
   python3 --version
   ```

3. **Restart service:**
   ```bash
   sudo systemctl restart jupyter
   ```

4. **Start manually for debugging:**
   ```bash
   sudo systemctl stop jupyter
   jupyter lab --ip=0.0.0.0 --port=8888 --no-browser
   ```

5. **Check for port conflicts:**
   ```bash
   sudo lsof -i :8888
   # If something else is using port 8888, kill it or use different port
   ```

### Jupyter Kernel Died

**Problem:** "Kernel died, restarting" message in Jupyter

**Solutions:**

1. **Check memory usage:**
   ```bash
   free -h
   htop
   # May need larger instance type
   ```

2. **Check kernel logs:**
   ```bash
   # In Jupyter, go to: Kernel -> Kernel died -> View kernel logs
   ```

3. **Install missing packages:**
   ```bash
   pip install --upgrade ipykernel
   python -m ipykernel install --user
   ```

4. **Increase instance size:**
   ```bash
   # Stop instance, change type via console, start again
   aws-jupyter stop i-0abc123def456789
   # Change via AWS Console
   aws ec2 start-instances --instance-ids i-0abc123def456789
   ```

## Permission Errors

### IAM Permission Denied

**Problem:**
```
Error: UnauthorizedOperation: You are not authorized to perform this operation
```

**Solutions:**

1. **Check IAM permissions:**
   ```bash
   aws iam get-user
   aws iam list-attached-user-policies --user-name youruser
   aws iam list-user-policies --user-name youruser
   ```

2. **Required permissions:**
   - EC2: `RunInstances`, `DescribeInstances`, `TerminateInstances`
   - IAM: `CreateRole`, `AttachRolePolicy`, `PassRole`
   - SSM: `StartSession` (for Session Manager)

3. **Use PowerUserAccess (development):**
   ```bash
   aws iam attach-user-policy \
     --user-name youruser \
     --policy-arn arn:aws:iam::aws:policy/PowerUserAccess
   ```

4. **See AWS_AUTHENTICATION.md** for detailed permission requirements

### PassRole Permission Denied

**Problem:**
```
Error: User is not authorized to perform: iam:PassRole on resource
```

**Solution:**

Add PassRole permission:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "iam:PassRole",
      "Resource": "arn:aws:iam::123456789012:role/aws-jupyter-*"
    }
  ]
}
```

### Key Pair Permission Issues

**Problem:** Can't create or use key pairs

**Solution:**

Add EC2 key pair permissions:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:CreateKeyPair",
        "ec2:DescribeKeyPairs",
        "ec2:DeleteKeyPair"
      ],
      "Resource": "*"
    }
  ]
}
```

## Cost and Billing

### Unexpected NAT Gateway Charges

**Problem:** High AWS bill from NAT Gateway

**Understanding Costs:**
```
NAT Gateway charges:
- Per hour: $0.045/hour = $32.40/month
- Data transfer: $0.045/GB

Example monthly costs:
- Base: $32.40
- 50GB data: $2.25
- Total: ~$35/month per NAT Gateway
```

**Solutions:**

1. **Check NAT Gateway usage:**
   ```bash
   aws ec2 describe-nat-gateways --filters "Name=state,Values=available"
   ```

2. **Delete unused NAT Gateways:**
   ```bash
   aws ec2 delete-nat-gateway --nat-gateway-id nat-12345
   ```

3. **Use VPC Endpoints instead:**
   - See PRIVATE_SUBNET_GUIDE.md
   - Can save $10-20/month

4. **Stop instances when not in use:**
   ```bash
   aws-jupyter stop i-0abc123def456789
   # NAT Gateway continues but instance stops
   ```

### Forgot to Stop Instance

**Problem:** Ran instance for a month by accident

**Prevention:**

1. **Set up billing alerts:**
   ```bash
   # AWS Console -> Billing -> Billing Preferences
   # Enable billing alerts
   ```

2. **Use AWS Budgets:**
   - Set monthly budget
   - Get alerts at 80%, 100%

3. **Tag instances:**
   ```bash
   aws-jupyter launch --tags "Owner=yourname,Project=research"
   ```

4. **Regular cleanup:**
   ```bash
   # Weekly check
   aws-jupyter list
   ```

5. **Use instance scheduler:**
   - AWS Instance Scheduler solution
   - Auto-stop nights/weekends

### EBS Volume Costs

**Problem:** Paying for EBS volumes from terminated instances

**Solution:**

1. **Check for orphaned volumes:**
   ```bash
   aws ec2 describe-volumes \
     --filters "Name=status,Values=available"
   ```

2. **Delete unused volumes:**
   ```bash
   aws ec2 delete-volume --volume-id vol-12345
   ```

3. **Verify delete-on-termination:**
   ```bash
   aws ec2 describe-instances --instance-ids i-0abc123def456789 \
     --query 'Reservations[0].Instances[0].BlockDeviceMappings[0].Ebs.DeleteOnTermination'
   ```

## Performance Issues

### Slow Instance Performance

**Problem:** Jupyter notebooks running slowly

**Solutions:**

1. **Check instance size:**
   ```bash
   aws-jupyter status i-0abc123def456789
   # Check InstanceType

   # Upgrade to larger instance (requires stop/start)
   ```

2. **Monitor resource usage:**
   ```bash
   aws-jupyter connect i-0abc123def456789

   # Check CPU/memory
   htop

   # Check disk I/O
   iostat -x 1
   ```

3. **Check EBS volume type:**
   ```bash
   # gp3 is faster than gp2
   aws ec2 describe-volumes --volume-ids vol-12345
   ```

4. **Profile your code:**
   ```python
   # In Jupyter:
   %load_ext line_profiler
   %lprun -f your_function your_function(args)
   ```

### Slow Package Installation

**Problem:** `pip install` taking forever

**Solutions:**

1. **Check internet speed:**
   ```bash
   curl -o /dev/null https://pypi.org/simple/
   ```

2. **Use NAT Gateway (private subnet):**
   ```bash
   # Faster than VPC endpoints for PyPI
   aws-jupyter launch --create-nat-gateway
   ```

3. **Use pip cache:**
   ```bash
   pip install --cache-dir ~/.cache/pip package-name
   ```

4. **Install from requirements.txt:**
   ```bash
   # Faster than individual installs
   pip install -r requirements.txt
   ```

### Network Latency

**Problem:** High latency to instance

**Solutions:**

1. **Use closer region:**
   ```bash
   # If in US West, use us-west-2 not eu-west-1
   aws-jupyter launch --region us-west-2
   ```

2. **Check your internet:**
   ```bash
   ping 8.8.8.8
   speedtest-cli
   ```

3. **Use Session Manager (can be faster):**
   ```bash
   aws-jupyter connect i-0abc123def456789
   ```

## Getting Help

### Debug Information

When reporting issues, include:

```bash
# 1. aws-jupyter version
aws-jupyter version

# 2. Instance status
aws-jupyter status i-0abc123def456789

# 3. AWS CLI version
aws --version

# 4. Go version
go version

# 5. Operating system
uname -a

# 6. Error messages (full output)
aws-jupyter launch --dry-run 2>&1 | tee error.log
```

### Common Commands for Debugging

```bash
# Check AWS connectivity
aws sts get-caller-identity

# List all resources
aws-jupyter list
aws ec2 describe-instances
aws ec2 describe-key-pairs
aws ec2 describe-security-groups

# Check logs (on instance)
sudo journalctl -u jupyter -f
sudo journalctl -u amazon-ssm-agent -f
dmesg | tail -n 50

# Network diagnostics
aws ec2 describe-vpc-endpoints
aws ec2 describe-nat-gateways
aws ec2 describe-route-tables

# IAM diagnostics
aws iam get-user
aws iam list-attached-user-policies --user-name $(aws iam get-user --query 'User.UserName' --output text)
```

### Reporting Issues

1. **Check existing issues:**
   - https://github.com/scttfrdmn/aws-jupyter/issues

2. **Search documentation:**
   - [README](../README.md)
   - [Session Manager Setup](SESSION_MANAGER_SETUP.md)
   - [Private Subnet Guide](PRIVATE_SUBNET_GUIDE.md)
   - [Examples](EXAMPLES.md)

3. **Open new issue:**
   - Provide debug information above
   - Include steps to reproduce
   - Describe expected vs actual behavior

### Community Support

- **GitHub Discussions:** https://github.com/scttfrdmn/aws-jupyter/discussions
- **Issue Tracker:** https://github.com/scttfrdmn/aws-jupyter/issues

### AWS Support

For AWS-specific issues:
- **AWS Documentation:** https://docs.aws.amazon.com/
- **AWS Support:** (If you have support plan)
- **AWS re:Post:** https://repost.aws/

## Additional Resources

**aws-jupyter Documentation:**
- [Main README](../README.md)
- [Session Manager Setup](SESSION_MANAGER_SETUP.md)
- [Private Subnet Guide](PRIVATE_SUBNET_GUIDE.md)
- [Examples & Use Cases](EXAMPLES.md)
- [Roadmap](../ROADMAP.md)

**AWS Documentation:**
- [EC2 User Guide](https://docs.aws.amazon.com/ec2/)
- [Session Manager](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager.html)
- [VPC Guide](https://docs.aws.amazon.com/vpc/)

---

**Still having issues?** Open an issue on [GitHub](https://github.com/scttfrdmn/aws-jupyter/issues) with detailed information about your problem.
