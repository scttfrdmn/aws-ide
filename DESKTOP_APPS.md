# Desktop Applications Architecture Plan

This document outlines a preliminary plan for supporting desktop GUI applications (QGIS, MATLAB, etc.) using AWS Nice DCV for browser-based remote desktop access.

## Overview

### What is Nice DCV?

**NICE DCV** (Desktop Cloud Visualization) is AWS's high-performance remote desktop and application streaming protocol designed for cloud-based desktop environments.

**Key Features**:
- Browser-based access (no client installation required)
- GPU acceleration support (OpenGL, DirectX, CUDA)
- High frame rates for smooth interaction (up to 60 fps)
- Low latency for responsive desktop experience
- Secure WebSocket-based connections
- Multi-monitor support
- USB device redirection
- Audio streaming

**Comparison to Alternatives**:
- **VNC**: Lower performance, no GPU acceleration
- **RDP**: Windows-focused, limited Linux support
- **X11 Forwarding**: Network-intensive, poor for complex GUIs
- **Nice DCV**: Purpose-built for cloud, best performance

### Architecture Comparison

#### Current Web-Based Apps (Jupyter, RStudio, VSCode)

```
┌─────────────────────────────────────────────────────┐
│  User Browser                                       │
│  └─> http://localhost:8888                         │
└─────────────────┬───────────────────────────────────┘
                  │ HTTP/WebSocket
                  │ Port Forward (SSM/SSH)
┌─────────────────▼───────────────────────────────────┐
│  EC2 Instance (Ubuntu Server - No GUI)              │
│  ├─> Web Server (Jupyter/RStudio/code-server)       │
│  ├─> Port 8888/8787/8080                            │
│  └─> Process runs headless                          │
└─────────────────────────────────────────────────────┘
```

**Characteristics**:
- Applications are web-native (serve HTTP)
- No desktop environment required
- Minimal resource requirements (2-4 vCPU, 8-16GB RAM)
- Quick launch times (~2-5 minutes)
- Standard Ubuntu Server AMI sufficient

#### Desktop Apps with Nice DCV (QGIS, MATLAB, etc.)

```
┌─────────────────────────────────────────────────────┐
│  User Browser                                       │
│  └─> http://localhost:8443 (DCV Web Client)        │
└─────────────────┬───────────────────────────────────┘
                  │ HTTPS/WebSocket (DCV Protocol)
                  │ Port Forward (SSM/SSH)
┌─────────────────▼───────────────────────────────────┐
│  EC2 Instance (Ubuntu Desktop)                      │
│  ├─> Nice DCV Server (Port 8443)                    │
│  ├─> Desktop Environment (GNOME/XFCE)               │
│  ├─> X11 Display Server                             │
│  └─> Desktop Apps (QGIS, MATLAB, etc.)              │
│      └─> Render to virtual display                  │
│      └─> Stream via DCV protocol                    │
└─────────────────────────────────────────────────────┘
```

**Characteristics**:
- Full Linux desktop environment required
- Nice DCV server streams desktop to browser
- Higher resource requirements (4-8+ vCPU, 16-32GB+ RAM)
- GPU instances optional (for visualization-heavy work)
- Longer launch times (~5-10 minutes)
- Requires Ubuntu Desktop or specialized DCV AMI

## Technical Requirements

### 1. AMI Requirements

**Current Apps**: Ubuntu Server 22.04 LTS (minimal)

**Desktop Apps**: Need one of:

#### Option A: Ubuntu Desktop AMI
```
Base: Ubuntu 22.04 Desktop
Installed:
- GNOME or XFCE desktop environment
- X11 display server
- Nice DCV server (2023.0+)
- OpenGL libraries
- Desktop app (QGIS/MATLAB/etc.)
```

**Pros**: Full desktop environment, familiar to users
**Cons**: Heavy (~5GB), slower launches, more resource-intensive

#### Option B: AWS-Provided DCV AMI
```
Base: Amazon Linux 2 or Ubuntu with DCV pre-installed
Installed:
- Pre-configured DCV server
- Optimized for streaming
- Desktop environment included
- GPU drivers pre-installed (if GPU AMI)
```

**Pros**: Production-ready, optimized, AWS-supported
**Cons**: May require licensing, less control over configuration

#### Option C: Minimal Desktop (Headless + DCV)
```
Base: Ubuntu Server 22.04
Installed:
- Xvfb (virtual framebuffer)
- Lightweight WM (Openbox)
- Nice DCV server
- Only the specific desktop app needed
```

**Pros**: Lighter weight, faster launches, lower cost
**Cons**: More complex setup, may not work for all apps

**Recommendation**: Start with Option B (AWS DCV AMI) for reliability, consider Option C for cost optimization later.

### 2. Instance Type Requirements

**Current Apps**: General-purpose instances (t3/t3a family)
- 2-4 vCPUs
- 8-16GB RAM
- No GPU required
- $0.03-0.15/hour

**Desktop Apps**: Depends on application

#### Minimal Desktop Apps (e.g., simple visualization)
```
Instance Type: t3.xlarge or t3a.xlarge
- 4 vCPUs
- 16GB RAM
- No GPU
- $0.15-0.20/hour
```

#### GPU-Accelerated Apps (QGIS, MATLAB with Simulink, CAD)
```
Instance Type: g4dn.xlarge or g5.xlarge
- 4-8 vCPUs
- 16-24GB RAM
- NVIDIA GPU (T4 or A10G)
- $0.50-1.00/hour
```

#### Heavy Computational Apps (large datasets, complex models)
```
Instance Type: g4dn.2xlarge or higher
- 8+ vCPUs
- 32+ GB RAM
- NVIDIA GPU
- $0.75-2.00+/hour
```

### 3. Nice DCV Server Configuration

**Installation** (on Ubuntu):
```bash
# Download and install DCV server
wget https://d1uj6qtbmh3dt5.cloudfront.net/nice-dcv-ubuntu2204-x86_64.tgz
tar -xvzf nice-dcv-ubuntu2204-x86_64.tgz
cd nice-dcv-*-ubuntu2204-x86_64
sudo apt install ./nice-dcv-server_*.deb
sudo apt install ./nice-dcv-web-viewer_*.deb
sudo apt install ./nice-xdcv_*.deb

# Configure DCV server
sudo vim /etc/dcv/dcv.conf
# Set web-port=8443
# Set authentication=system

# Start DCV server
sudo systemctl enable dcvserver
sudo systemctl start dcvserver

# Create virtual session
dcv create-session --type=virtual --owner ubuntu my-session
```

**Key Configuration**:
- Port: 8443 (default, configurable)
- Authentication: System auth or DCV auth
- Session type: Virtual (headless) or console
- GPU acceleration: Auto-detected if available
- Quality settings: Configurable frame rate, compression

### 4. Network Configuration

**Current Apps**:
- Inbound: Only SSM (no public ports)
- Outbound: Standard (HTTPS, package repos)
- Port forwarding: SSM Session Manager

**Desktop Apps with DCV**:
- Inbound: Port 8443 (DCV HTTPS) - can use SSM forwarding
- Outbound: Standard + GPU driver updates
- Port forwarding: Same SSM/SSH approach

**Security Group** (using SSM forwarding):
```
Inbound Rules:
- None (SSM only, same as current apps)

Outbound Rules:
- All traffic (for package installation)
```

**Security Group** (alternative: direct DCV access):
```
Inbound Rules:
- TCP 8443 from user's IP (DCV HTTPS)
- TCP 22 from user's IP (SSH, optional)

Outbound Rules:
- All traffic
```

**Recommendation**: Use SSM port forwarding (same as current) to avoid exposing ports.

### 5. User Authentication

**Current Apps**: Token-based (Jupyter) or password (RStudio)

**Desktop Apps with DCV**:

#### Option A: System Authentication
```
- Create Linux user (ubuntu)
- Set password
- DCV uses system auth
- User logs in with Linux credentials
```

**Pros**: Simple, familiar
**Cons**: Requires managing Linux passwords

#### Option B: DCV Authentication File
```
- Generate DCV password file
- Store credentials in state
- Auto-inject on first connection
```

**Pros**: Automated, no user password management
**Cons**: More complex setup

#### Option C: SSO/IAM Integration
```
- Use AWS IAM credentials
- DCV integrates with IAM
- No separate passwords
```

**Pros**: Enterprise-grade, no password management
**Cons**: Complex setup, may require DCV license

**Recommendation**: Start with Option A (system auth) for simplicity.

## CLI Workflow Design

### Proposed Commands

The desktop app CLI would follow the same pattern as current apps but with DCV-specific options:

```bash
# Launch desktop environment with QGIS
aws-qgis launch --profile myprofile --instance-type g4dn.xlarge

# Launch MATLAB environment
aws-matlab launch --profile myprofile --instance-type g4dn.2xlarge --gpu

# Connect to desktop session (opens browser)
aws-qgis connect <instance-id>
# This would:
# 1. Start SSM port forwarding for port 8443
# 2. Open http://localhost:8443 in browser
# 3. User logs in with credentials (auto-filled or prompted)

# List desktop sessions
aws-qgis list

# Stop desktop session (keep instance running)
aws-qgis stop <instance-id>

# Terminate instance
aws-qgis terminate <instance-id>
```

### Launch Workflow

**Current Apps**:
1. Launch EC2 instance (Ubuntu Server)
2. cloud-init installs web application
3. Wait for service readiness (SSM polling)
4. Port forward and open browser
5. User accesses web app immediately

**Desktop Apps**:
1. Launch EC2 instance (Ubuntu Desktop or DCV AMI)
2. cloud-init installs:
   - Desktop environment (if not in AMI)
   - Nice DCV server
   - Desktop application (QGIS, MATLAB, etc.)
   - Configure DCV session
3. Wait for DCV server readiness (SSM polling port 8443)
4. Create DCV virtual session
5. Port forward DCV port (8443) and open browser
6. User logs in to desktop environment
7. User launches desktop application from desktop

**Launch Time Comparison**:
- Current apps: 2-5 minutes
- Desktop apps: 5-10 minutes (desktop environment + DCV setup)
- Desktop apps (with pre-built AMI): 3-6 minutes

### Connect Workflow

**Current Apps**:
```bash
aws-jupyter connect i-xxxxx
# → SSM/SSH port forward localhost:8888 → instance:8888
# → Open http://localhost:8888
# → User sees Jupyter immediately
```

**Desktop Apps**:
```bash
aws-qgis connect i-xxxxx
# → SSM/SSH port forward localhost:8443 → instance:8443
# → Open http://localhost:8443
# → DCV web client loads in browser
# → User logs in (ubuntu / <password>)
# → Full desktop appears in browser
# → User clicks QGIS icon or terminal to launch app
```

**Key Difference**: Desktop apps require login + manual app launch, web apps are immediate.

### Readiness Polling

**Current Apps** (SSM-based):
```bash
# Check if web server responds on port 8888
curl -s -o /dev/null -w "%{http_code}" http://localhost:8888
# Wait for 200/302 response
```

**Desktop Apps** (SSM-based):
```bash
# Check if DCV server responds on port 8443
curl -k -s -o /dev/null -w "%{http_code}" https://localhost:8443
# Wait for 200/302 response (DCV web client)

# Alternative: Check DCV session status
dcv list-sessions
# Wait for session to be "READY"
```

### State Management

**Current Apps**:
```json
{
  "instance_id": "i-xxxxx",
  "instance_type": "t3.medium",
  "service": "jupyter",
  "port": 8888,
  "token": "abc123...",
  "region": "us-west-2"
}
```

**Desktop Apps**:
```json
{
  "instance_id": "i-xxxxx",
  "instance_type": "g4dn.xlarge",
  "service": "qgis",
  "dcv_port": 8443,
  "dcv_session": "my-session",
  "dcv_user": "ubuntu",
  "dcv_password": "temp123",
  "gpu_enabled": true,
  "region": "us-west-2"
}
```

## Candidate Desktop Applications

### High Priority

#### 1. QGIS ⭐⭐⭐⭐⭐

**What**: Open-source Geographic Information System

**Use Case**:
- Spatial data analysis
- Map creation and visualization
- Geospatial research (geography, environmental science, urban planning)
- GIS coursework and teaching

**Why Desktop App**:
- Complex GUI with multiple panels
- Requires OpenGL for rendering
- Interactive map manipulation
- Plugin ecosystem designed for desktop

**Technical Requirements**:
- Instance: t3.xlarge or g4dn.xlarge (if large datasets)
- Memory: 16GB minimum
- GPU: Optional but recommended for large rasters
- Storage: 20-50GB (for GIS datasets)

**Installation**:
```bash
sudo apt-get update
sudo apt-get install qgis qgis-plugin-grass
```

**References**:
- Website: https://qgis.org/
- Documentation: https://docs.qgis.org/

---

#### 2. MATLAB ⭐⭐⭐⭐⭐

**What**: Commercial numerical computing environment

**Use Case**:
- Engineering simulations
- Signal processing
- Control systems
- Academic coursework (widespread in universities)
- Alternative to Octave (when license available)

**Why Desktop App**:
- Rich desktop IDE (editor, debugger, profiler)
- Simulink requires GUI for block diagrams
- Visualization tools (3D plots, animations)
- Toolbox interfaces are desktop-native

**Technical Requirements**:
- Instance: g4dn.xlarge or higher (Simulink benefits from GPU)
- Memory: 16-32GB
- GPU: Recommended for Simulink, required for Parallel Computing Toolbox
- Storage: 30-50GB (MATLAB + toolboxes)
- License: Requires valid MATLAB license (user responsibility)

**Installation**:
```bash
# User must provide MATLAB installer and license file
# Installation via silent install
./install -mode silent -inputFile installer_input.txt
```

**Licensing Considerations**:
- Users must have institutional or personal MATLAB license
- License can be network-based (FlexNet) or individual
- AWS Marketplace MATLAB AMIs available (pay-per-use)

**References**:
- Website: https://www.mathworks.com/products/matlab.html
- AWS Marketplace: https://aws.amazon.com/marketplace/pp/prodview-vya5gvtpvkuns

---

#### 3. RStudio Desktop ⭐⭐⭐⭐

**What**: Desktop version of RStudio (more features than Server)

**Use Case**:
- Same as RStudio Server, but with desktop features
- Users who prefer desktop experience
- Access to desktop-only plugins/extensions

**Why Desktop App**:
- Some R packages require desktop environment
- Desktop version has features not in Server version
- Familiar experience for desktop RStudio users

**Technical Requirements**:
- Instance: t3.xlarge
- Memory: 16GB
- GPU: Not required
- Storage: 20GB

**Installation**:
```bash
wget https://download1.rstudio.org/desktop/jammy/amd64/rstudio-2023.12.0-daily-amd64.deb
sudo apt install ./rstudio-2023.12.0-daily-amd64.deb
```

**Note**: This would be an alternative to the existing aws-rstudio (Server version). May not be worth the complexity since RStudio Server is excellent.

---

### Medium Priority

#### 4. ParaView ⭐⭐⭐⭐

**What**: Open-source scientific visualization application

**Use Case**:
- Visualize large scientific datasets
- CFD (Computational Fluid Dynamics) result analysis
- 3D rendering of simulation outputs
- Climate modeling visualization
- Medical imaging

**Why Desktop App**:
- Complex 3D visualization requires GPU
- Interactive manipulation of large datasets
- Pipeline-based workflow in GUI

**Technical Requirements**:
- Instance: g4dn.xlarge or g5.xlarge
- Memory: 16-32GB
- GPU: Required for large datasets
- Storage: 50-100GB (large visualization datasets)

**Installation**:
```bash
sudo apt-get install paraview
# Or download latest from paraview.org
```

**References**:
- Website: https://www.paraview.org/
- Documentation: https://docs.paraview.org/

---

#### 5. Inkscape / GIMP ⭐⭐⭐

**What**: Vector graphics (Inkscape) and raster graphics (GIMP) editors

**Use Case**:
- Create figures for academic papers
- Edit images for presentations
- Design diagrams and illustrations
- Prepare graphics for publication

**Why Desktop App**:
- Requires GUI for visual editing
- Complex toolbars and panels

**Technical Requirements**:
- Instance: t3.xlarge
- Memory: 16GB
- GPU: Not required
- Storage: 20GB

**Installation**:
```bash
sudo apt-get install inkscape gimp
```

**Note**: Lower priority as many researchers use local tools for graphics editing.

---

### Lower Priority

#### 6. PyMOL ⭐⭐⭐

**What**: Molecular visualization system

**Use Case**:
- Protein structure visualization
- Molecular biology research
- Structural biology
- Drug design

**Why Desktop App**:
- 3D molecular rendering
- Interactive structure manipulation

**Technical Requirements**:
- Instance: t3.xlarge or g4dn.xlarge
- Memory: 16GB
- GPU: Optional
- Storage: 20GB

**Installation**:
```bash
sudo apt-get install pymol
```

---

#### 7. Blender ⭐⭐

**What**: 3D modeling and rendering software

**Use Case**:
- Scientific visualization
- 3D animation for presentations
- Architectural visualization
- Limited academic use

**Why Desktop App**:
- Complex 3D interface
- GPU-accelerated rendering

**Technical Requirements**:
- Instance: g4dn.xlarge or higher
- Memory: 32GB
- GPU: Required for rendering
- Storage: 50GB

**Installation**:
```bash
sudo apt-get install blender
```

---

## Implementation Strategy

### Phase 1: Proof of Concept (v0.8.0)

**Goal**: Validate Nice DCV approach with single application

**Tasks**:
1. Create DCV-enabled AMI (Ubuntu Desktop + DCV + QGIS)
2. Build `aws-qgis` CLI application
3. Implement DCV-specific launch/connect workflow
4. Test SSM port forwarding for DCV (port 8443)
5. Document user workflow

**Success Criteria**:
- User can launch QGIS environment in <10 minutes
- DCV connection works reliably via SSM port forwarding
- User can interact with QGIS smoothly (acceptable frame rate)
- Cost is reasonable (<$0.50/hour for t3.xlarge)

**Deliverables**:
- `apps/qgis/` application
- QGIS AMI (private, for testing)
- Updated documentation

### Phase 2: Expand to MATLAB (v0.9.0)

**Goal**: Support licensed commercial software

**Tasks**:
1. Create MATLAB-compatible DCV AMI (with GPU support)
2. Handle license file/key input from user
3. Build `aws-matlab` CLI application
4. Test GPU-accelerated workflows (Simulink)
5. Document licensing requirements

**Success Criteria**:
- User can provide MATLAB license and launch environment
- GPU acceleration works for Simulink
- Frame rate is acceptable for interactive work
- Documentation clearly explains licensing requirements

**Deliverables**:
- `apps/matlab/` application
- MATLAB AMI (user brings license)
- GPU instance support

### Phase 3: Generalize Desktop Framework (v1.0.0)

**Goal**: Create reusable framework for desktop apps

**Tasks**:
1. Extract common DCV code into `pkg/dcv/`
2. Create generic `pkg/desktop/` launcher
3. Standardize AMI build process
4. Support multiple desktop apps from single AMI
5. Add multi-session support (multiple apps simultaneously)

**Success Criteria**:
- Adding new desktop app requires minimal code
- Common AMI can support multiple apps
- Users can run QGIS + ParaView + RStudio Desktop simultaneously
- Framework is well-documented

**Deliverables**:
- `pkg/dcv/` and `pkg/desktop/` libraries
- Multi-app AMI
- Framework documentation

## Cost Analysis

### Current Web-Based Apps

**Typical Usage** (8 hours/day, 20 days/month):

| App | Instance | $/hour | Monthly Cost |
|-----|----------|--------|--------------|
| Jupyter | t3.medium | $0.042 | $6.72 |
| RStudio | t3.medium | $0.042 | $6.72 |
| VSCode | t3.medium | $0.042 | $6.72 |

**Total**: ~$7/month per app

### Desktop Apps with Nice DCV

**Typical Usage** (8 hours/day, 20 days/month):

| App | Instance | $/hour | Monthly Cost | Notes |
|-----|----------|--------|--------------|-------|
| QGIS (basic) | t3.xlarge | $0.166 | $26.56 | No GPU |
| QGIS (large datasets) | g4dn.xlarge | $0.526 | $84.16 | With GPU |
| MATLAB | g4dn.xlarge | $0.526 | $84.16 | GPU recommended |
| ParaView | g4dn.2xlarge | $0.752 | $120.32 | Large viz |
| RStudio Desktop | t3.xlarge | $0.166 | $26.56 | No GPU needed |

**Cost Comparison**:
- Desktop apps: 4-12x more expensive than web apps
- GPU instances: 10-15x more expensive than web apps
- Still cost-effective vs. purchasing desktop workstation + GPU

**Cost Optimization Strategies**:
1. **Auto-stop on idle**: More important for desktop apps
2. **Spot instances**: 60-70% discount for non-critical work
3. **Right-sizing**: Start with t3.xlarge, upgrade only if needed
4. **Scheduled instances**: Reserve instances for regular heavy users
5. **Multi-app AMIs**: Share single instance for multiple light apps

## Challenges and Considerations

### 1. Launch Time

**Challenge**: Desktop environments take longer to initialize (5-10 min vs 2-5 min)

**Mitigation**:
- Pre-bake AMIs with desktop + DCV + applications
- Use faster instance types during launch (e.g., c5.2xlarge for setup)
- Parallel installation in cloud-init where possible
- Provide clear progress indication (DCV server is 60% of launch time)

### 2. User Experience

**Challenge**: Desktop apps require login + manual app launch (less streamlined than web apps)

**Mitigation**:
- Auto-login to desktop environment
- Auto-launch application on desktop startup
- Provide desktop shortcuts and clear instructions
- Consider kiosk mode (full-screen app, no desktop shown)

### 3. Licensing

**Challenge**: Commercial software (MATLAB) requires user to provide license

**Mitigation**:
- Clear documentation on license requirements
- Support multiple license methods (file, key, network)
- Consider AWS Marketplace MATLAB (pay-per-use) as alternative
- Provide license verification step in CLI

### 4. GPU Availability

**Challenge**: GPU instances may not be available in all regions or have limits

**Mitigation**:
- Default to non-GPU instances where possible
- Provide clear error messages if GPU unavailable
- Document how to request limit increases
- Support multiple GPU instance families (g4dn, g5)

### 5. Network Performance

**Challenge**: DCV streaming requires good network connection

**Mitigation**:
- Use SSM Session Manager (more reliable than direct connection)
- Support quality settings (frame rate, compression)
- Document bandwidth requirements (~5-10 Mbps)
- Provide fallback to lower quality settings

### 6. Storage

**Challenge**: Desktop apps + datasets require more storage than web apps

**Mitigation**:
- Default to larger root volumes (50-100GB vs 20GB)
- Support EBS volume attachment for large datasets
- Document cleanup procedures
- Consider EFS for shared datasets across sessions

### 7. Cost

**Challenge**: Desktop apps cost 4-12x more than web apps due to GPU/larger instances

**Mitigation**:
- Prominent cost warnings in CLI and documentation
- Aggressive auto-stop on idle (15 min vs 60 min)
- Recommend Spot instances for non-critical work
- Provide cost estimation before launch
- Support scheduled start/stop

## Architectural Differences Summary

| Aspect | Web-Based Apps | Desktop Apps (DCV) |
|--------|----------------|-------------------|
| **AMI** | Ubuntu Server | Ubuntu Desktop or DCV AMI |
| **Desktop Env** | None | GNOME/XFCE/Lightweight |
| **Streaming** | Native HTTP | DCV Protocol (HTTPS/WebSocket) |
| **Port** | 8888/8787/8080 | 8443 (DCV) |
| **Instance** | t3.medium (2vCPU, 8GB) | t3.xlarge+ (4vCPU, 16GB+) |
| **GPU** | Not required | Optional/Required |
| **Launch Time** | 2-5 minutes | 5-10 minutes |
| **Cost** | ~$7/month | ~$27-120/month |
| **User Flow** | Open URL → immediate access | Open URL → login → launch app |
| **Readiness Check** | HTTP check on app port | HTTP check on DCV port |
| **Authentication** | Token/password (app-level) | Linux user (system-level) |
| **Use Cases** | Web-native coding/analysis | GUI-heavy visualization/simulation |

## Recommended Applications for v0.8.0

Based on demand, complexity, and value:

1. **QGIS** (v0.8.0)
   - High demand in GIS research community
   - Open-source (no licensing issues)
   - Can work without GPU (lower cost)
   - Good proof-of-concept for DCV approach

2. **MATLAB** (v0.9.0)
   - Extremely high demand in engineering/academia
   - Licensing complexity (good to solve early)
   - Requires GPU for best experience
   - High value for institutions with licenses

3. **ParaView** (v1.0.0)
   - Moderate demand in visualization-heavy fields
   - Open-source
   - Requires GPU
   - Good complement to QGIS

## Next Steps

1. **Research Phase** (Current)
   - Review Nice DCV documentation thoroughly
   - Test DCV on EC2 manually (validate approach)
   - Estimate actual costs with real workloads
   - Gather user feedback on desktop app priorities

2. **Prototype Phase** (Next)
   - Build basic DCV-enabled AMI with QGIS
   - Create minimal `aws-qgis` CLI
   - Test end-to-end workflow
   - Measure performance and costs

3. **Implementation Phase** (v0.8.0)
   - Build production-quality QGIS application
   - Create comprehensive documentation
   - Add auto-stop and cost controls
   - Gather user feedback

4. **Expansion Phase** (v0.9.0+)
   - Add MATLAB support
   - Generalize DCV framework
   - Consider additional apps based on demand

## Resources

### Nice DCV Documentation
- Overview: https://aws.amazon.com/hpc/dcv/
- User Guide: https://docs.aws.amazon.com/dcv/latest/userguide/
- Admin Guide: https://docs.aws.amazon.com/dcv/latest/adminguide/
- Installation: https://docs.aws.amazon.com/dcv/latest/adminguide/setting-up-installing.html

### Nice DCV Pricing
- Nice DCV is free for up to 4 concurrent sessions
- Enterprise licensing available for >4 sessions: https://aws.amazon.com/hpc/dcv/pricing/

### Alternative Solutions
- Apache Guacamole (open-source, but less performant): https://guacamole.apache.org/
- X2Go (open-source, NX protocol): https://wiki.x2go.org/

### AWS Resources
- DCV AMIs: https://aws.amazon.com/marketplace/search/results?searchTerms=nice+dcv
- GPU Instance Types: https://aws.amazon.com/ec2/instance-types/#Accelerated_Computing
- Session Manager Port Forwarding: https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-sessions-start.html#sessions-start-port-forwarding

## Conclusion

Supporting desktop applications with Nice DCV represents a significant architectural shift from the current web-based approach, but it's feasible and would unlock a new category of high-value applications for academic researchers.

**Key Takeaways**:
- Nice DCV is the right technology for browser-based desktop streaming
- Implementation complexity is medium (similar to adding RStudio)
- Costs are 4-12x higher due to GPU/larger instances, but still cost-effective
- Launch times are longer (5-10 min vs 2-5 min)
- User experience requires one extra step (login to desktop)
- QGIS is the ideal first application for v0.8.0
- MATLAB is high-value but requires licensing complexity
- Framework can be generalized for multiple desktop apps

**Recommendation**: Proceed with QGIS proof-of-concept in v0.8.0 development cycle after completing v0.7.0 web-based apps (Streamlit, OpenRefine, etc.).
