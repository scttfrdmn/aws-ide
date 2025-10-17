# Future Applications for Academic Researchers

This document tracks potential applications to add to the aws-ide project, focusing on academic research workflows.

## Priority Applications

### 1. OpenRefine ⭐⭐⭐⭐⭐

**What**: Data cleaning and transformation tool with GUI interface

**Use Case**:
- Clean messy survey data
- Transform scraped/collected data
- Reconcile data against external sources
- Prepare data before analysis in Jupyter/R

**Technical Details**:
- Language: Java
- Port: 3333
- Memory: ~512MB-1GB
- Install: Download JAR or use Docker
- Web-based interface

**User Persona**:
- Social scientists with survey data
- Digital humanities researchers
- Any researcher with messy CSV/Excel files
- Non-programmers who need data cleanup

**Disciplines**: Social science, digital humanities, qualitative research

**Implementation Complexity**: Medium
- Java-based, single JAR file
- Well-documented
- No database required
- Similar pattern to RStudio

**References**:
- Website: https://openrefine.org/
- GitHub: https://github.com/OpenRefine/OpenRefine
- Docker: https://hub.docker.com/r/felixlohmeier/openrefine

---

### 2. GNU Octave ⭐⭐⭐⭐

**What**: MATLAB-compatible numerical computing environment

**Use Case**:
- Run MATLAB code without license
- Numerical analysis and simulation
- Signal processing
- Linear algebra and differential equations
- Legacy .m files from other researchers

**Technical Details**:
- Language: C++
- Port: Multiple options
  - Option A: Jupyter kernel (port 8888, reuse jupyter)
  - Option B: Octave Online Server (port 8089, standalone)
  - Option C: VSCode extension (port 8080, reuse vscode)
- Memory: ~256MB-512MB
- Install: apt-get install octave

**User Persona**:
- Engineering researchers (mechanical, electrical, aerospace)
- Physics/mathematics researchers
- Signal processing researchers
- Researchers migrating from MATLAB

**Disciplines**: Engineering, physics, mathematics, computational science

**Implementation Complexity**:
- Low (Jupyter kernel) - Just add kernel to existing jupyter app
- Medium (Standalone) - octave-online-server requires Node.js + Docker

**Recommended Approach**: Start with Jupyter kernel, upgrade to standalone if demand warrants

**References**:
- Website: https://www.gnu.org/software/octave/
- Jupyter kernel: https://github.com/Calysto/octave_kernel
- Octave Online: https://github.com/octave-online/octave-online-server

---

### 3. Shiny Server ⭐⭐⭐⭐

**What**: Publish interactive R applications and dashboards

**Use Case**:
- Share R analysis as interactive web apps
- Create dashboards for research results
- Build data exploration tools
- Publish reproducible research

**Technical Details**:
- Language: R + Node.js
- Port: 3838
- Memory: ~512MB base + per-app memory
- Install: R package + shiny-server binary
- Integrates with RStudio

**User Persona**:
- R users who want to share their work
- Researchers creating interactive visualizations
- Labs publishing tools for their community
- Bioinformatics researchers

**Disciplines**: Bioinformatics, statistics, social science, any R users

**Implementation Complexity**: Medium
- Open-source version available
- Good documentation
- Natural companion to RStudio
- Could be combined with RStudio in same AMI

**Note**: RStudio already has Shiny app support, but Shiny Server allows publishing/hosting multiple apps

**References**:
- Website: https://posit.co/products/open-source/shiny-server/
- GitHub: https://github.com/rstudio/shiny-server
- Documentation: https://docs.posit.co/shiny-server/

---

### 4. Label Studio ⭐⭐⭐⭐

**What**: Data labeling and annotation platform

**Use Case**:
- Label images for computer vision research
- Annotate text for NLP research
- Code qualitative data (interviews, documents)
- Create training datasets for ML
- Collaborative annotation with research assistants

**Technical Details**:
- Language: Python + React
- Port: 8080 (configurable, would need different port)
- Memory: ~512MB-1GB
- Install: pip install label-studio
- Database: SQLite or PostgreSQL
- Web-based interface

**User Persona**:
- ML/AI researchers building datasets
- Qualitative researchers coding interviews
- Computer vision researchers
- NLP researchers
- Social science researchers with text data

**Disciplines**: Computer science, linguistics, sociology, psychology, communication

**Implementation Complexity**: Medium
- Python-based (similar to Jupyter)
- Good documentation
- Requires persistent storage for projects
- Multi-user support built-in

**References**:
- Website: https://labelstud.io/
- GitHub: https://github.com/heartexlabs/label-studio
- Documentation: https://labelstud.io/guide/

---

### 5. Datasette ⭐⭐⭐

**What**: Instant JSON API and web interface for SQLite/CSV datasets

**Use Case**:
- Publish research datasets
- Explore CSV/Excel files without coding
- Share data with collaborators
- Create queryable data repositories
- Reproducible research data access

**Technical Details**:
- Language: Python
- Port: 8001 (default, configurable)
- Memory: ~128MB-256MB + dataset size
- Install: pip install datasette
- Works with SQLite databases and CSV files
- Web-based interface with SQL query builder

**User Persona**:
- Researchers publishing data alongside papers
- Data librarians
- Researchers exploring others' datasets
- Anyone who needs to share structured data
- Non-programmers who need to query data

**Disciplines**: All disciplines (universal data need)

**Implementation Complexity**: Low
- Very simple Python application
- Single command to start
- No database setup required (uses SQLite)
- Plugin ecosystem for extensions

**Note**: Extremely lightweight, could potentially be bundled with another app

**References**:
- Website: https://datasette.io/
- GitHub: https://github.com/simonw/datasette
- Documentation: https://docs.datasette.io/

---

## Implementation Priority for v0.7.0+

**Recommended Order**:

1. **OpenRefine** (v0.7.0)
   - Fills biggest gap (data cleaning)
   - Universal need across disciplines
   - Medium complexity

2. **Octave Jupyter Kernel** (v0.7.0 or v0.7.1)
   - Quick win (extend existing jupyter)
   - Test demand before building standalone

3. **Datasette** (v0.7.0 or v0.7.1)
   - Very easy to implement
   - Could bundle with another app
   - Good for quick wins

4. **Shiny Server** (v0.8.0)
   - Natural companion to RStudio
   - Medium complexity
   - High value for R users

5. **Label Studio** (v0.8.0 or v0.9.0)
   - More specialized use case
   - Medium complexity
   - High value for ML/qualitative research

## Notes

- All applications follow the same pattern: web-based, single main port, Linux-compatible
- All can run headless on Ubuntu AMI
- All are open-source with active communities
- All fit academic research workflows
- Consider AMI combinations (e.g., RStudio + Shiny Server together)

## Alternative Considerations

**If implementation proves difficult:**
- **OpenRefine**: No good alternative
- **Octave**: Could just document manual setup in VSCode/Jupyter
- **Shiny Server**: RStudio already supports Shiny development
- **Label Studio**: Several alternatives (Prodigy, Doccano, LabelImg)
- **Datasette**: Could use Jupyter + pandas, but less user-friendly

## User Feedback Needed

Before implementing, consider gathering feedback on:
- Which applications would you use?
- Current painful workflows we could address?
- Existing tools you wish were easier to deploy?
- Collaboration features needed?
