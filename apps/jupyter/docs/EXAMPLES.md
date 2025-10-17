# Examples & Use Cases

Real-world examples and usage scenarios for aws-jupyter.

## Table of Contents

- [Quick Start Examples](#quick-start-examples)
- [Data Science Workflows](#data-science-workflows)
- [Machine Learning Projects](#machine-learning-projects)
- [Research & Academia](#research--academia)
- [Enterprise & Production](#enterprise--production)
- [Development & Testing](#development--testing)
- [Cost Optimization](#cost-optimization)
- [Team Collaboration](#team-collaboration)
- [Complete Workflows](#complete-workflows)

## Quick Start Examples

### Example 1: Simple Data Analysis

**Scenario:** Quick data exploration with pandas and matplotlib

```bash
# 1. Launch small instance with data science environment
aws-jupyter launch \
  --env data-science \
  --instance-type m7g.medium \
  --connection ssh

# 2. Wait for instance to be ready (2-3 minutes)
aws-jupyter list

# 3. Connect to Jupyter Lab
# SSH tunnel will be shown in output
# Open browser to http://localhost:8888

# 4. When done, stop the instance
aws-jupyter stop i-0abc123def456789

# 5. Later, start it again
aws ec2 start-instances --instance-ids i-0abc123def456789
aws-jupyter connect i-0abc123def456789
```

**Cost:** ~$30-40/month for 8 hours/day usage

### Example 2: Secure ML Training

**Scenario:** Train ML model with maximum security

```bash
# Launch in private subnet with Session Manager
aws-jupyter launch \
  --connection session-manager \
  --subnet-type private \
  --create-nat-gateway \
  --env ml-pytorch \
  --instance-type m7g.xlarge

# Connect securely (no SSH keys, no public IP)
aws-jupyter connect i-0abc123def456789

# Port forward Jupyter
aws ssm start-session \
  --target i-0abc123def456789 \
  --document-name AWS-StartPortForwardingSession \
  --parameters '{"portNumber":["8888"],"localPortNumber":["8888"]}'
```

**Cost:** ~$120/month + $35/month NAT Gateway = ~$155/month

### Example 3: Quick Experiment

**Scenario:** Test an idea quickly, then throw away

```bash
# 1. Launch minimal environment
aws-jupyter launch \
  --env minimal \
  --instance-type m7g.medium

# 2. Do your work in Jupyter

# 3. Terminate when done (not stop - fully delete)
aws-jupyter terminate i-0abc123def456789

# Cost: Pay only for hours used, e.g., 2 hours = $0.10
```

## Data Science Workflows

### Example 4: Large Dataset Processing

**Scenario:** Process 100GB dataset from S3

```bash
# 1. Launch instance with large storage
aws-jupyter launch \
  --env data-science \
  --instance-type m7g.xlarge \
  --volume-size 200  # 200GB EBS volume

# 2. Connect and download data
aws-jupyter connect i-0abc123def456789

# In instance:
aws s3 sync s3://my-bucket/dataset /home/ubuntu/data --region us-west-2

# 3. Process data in Jupyter
# Use pandas, dask, or spark

# 4. Upload results
aws s3 cp /home/ubuntu/results/ s3://my-bucket/results/ --recursive

# 5. Terminate (don't forget!)
exit  # from instance
aws-jupyter terminate i-0abc123def456789
```

**Jupyter Notebook Example:**
```python
import pandas as pd
import dask.dataframe as dd

# Use dask for larger-than-memory datasets
df = dd.read_csv('/home/ubuntu/data/*.csv')

# Process
result = df.groupby('category').agg({'value': 'sum'}).compute()

# Save
result.to_csv('/home/ubuntu/results/summary.csv')
```

### Example 5: Time Series Analysis

**Scenario:** Analyze financial time series data

```bash
# Launch with data science tools
aws-jupyter launch \
  --env data-science \
  --instance-type m7g.large \
  --connection session-manager
```

**Jupyter Notebook:**
```python
import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
from statsmodels.tsa.seasonal import seasonal_decompose
from statsmodels.tsa.arima.model import ARIMA

# Load data from S3
import boto3
s3 = boto3.client('s3')
s3.download_file('my-bucket', 'stock-data.csv', 'data.csv')

# Load and prepare
df = pd.read_csv('data.csv', parse_dates=['date'], index_col='date')

# Visualize
fig, axes = plt.subplots(3, 1, figsize=(12, 10))
df['price'].plot(ax=axes[0], title='Stock Price')
df['price'].rolling(30).mean().plot(ax=axes[1], title='30-day MA')

# Decompose
decomposition = seasonal_decompose(df['price'], model='additive', period=30)
decomposition.plot()

# Forecast
model = ARIMA(df['price'], order=(1,1,1))
model_fit = model.fit()
forecast = model_fit.forecast(steps=30)

print(f"Next 30 days forecast:\n{forecast}")
```

### Example 6: Interactive Dashboard

**Scenario:** Build interactive data dashboard with Plotly

```bash
# Launch with data science environment
aws-jupyter launch \
  --env data-science \
  --instance-type m7g.large
```

**Jupyter Notebook:**
```python
import pandas as pd
import plotly.graph_objects as go
from plotly.subplots import make_subplots
import ipywidgets as widgets
from IPython.display import display

# Load data
df = pd.read_csv('sales-data.csv', parse_dates=['date'])

# Create interactive dashboard
def update_dashboard(region, product):
    # Filter data
    filtered = df[(df['region'] == region) & (df['product'] == product)]

    # Create subplots
    fig = make_subplots(
        rows=2, cols=2,
        subplot_titles=('Sales Over Time', 'Monthly Revenue',
                       'Top Customers', 'Product Mix')
    )

    # Add traces
    fig.add_trace(
        go.Scatter(x=filtered['date'], y=filtered['sales'], mode='lines'),
        row=1, col=1
    )

    # ... more plots ...

    fig.update_layout(height=800, showlegend=False)
    fig.show()

# Create widgets
region_dropdown = widgets.Dropdown(
    options=df['region'].unique(),
    description='Region:'
)
product_dropdown = widgets.Dropdown(
    options=df['product'].unique(),
    description='Product:'
)

# Display
widgets.interact(update_dashboard, region=region_dropdown, product=product_dropdown)
```

## Machine Learning Projects

### Example 7: Train PyTorch Model

**Scenario:** Train computer vision model on custom dataset

```bash
# Launch GPU instance (if available) or large CPU
aws-jupyter launch \
  --env ml-pytorch \
  --instance-type m7g.2xlarge \
  --volume-size 100 \
  --connection session-manager
```

**Jupyter Notebook:**
```python
import torch
import torch.nn as nn
import torchvision
from torchvision import transforms, datasets
from torch.utils.data import DataLoader

# Set device
device = torch.device('cuda' if torch.cuda.is_available() else 'cpu')
print(f"Using device: {device}")

# Load data from S3
import boto3
s3 = boto3.client('s3')
s3.download_file('my-bucket', 'dataset.tar.gz', 'dataset.tar.gz')
!tar -xzf dataset.tar.gz

# Prepare data
transform = transforms.Compose([
    transforms.Resize(224),
    transforms.ToTensor(),
    transforms.Normalize(mean=[0.485, 0.456, 0.406],
                        std=[0.229, 0.224, 0.225])
])

train_dataset = datasets.ImageFolder('dataset/train', transform=transform)
train_loader = DataLoader(train_dataset, batch_size=32, shuffle=True)

# Define model
model = torchvision.models.resnet50(pretrained=True)
model.fc = nn.Linear(model.fc.in_features, len(train_dataset.classes))
model = model.to(device)

# Training loop
criterion = nn.CrossEntropyLoss()
optimizer = torch.optim.Adam(model.parameters(), lr=0.001)

for epoch in range(10):
    model.train()
    running_loss = 0.0

    for images, labels in train_loader:
        images, labels = images.to(device), labels.to(device)

        optimizer.zero_grad()
        outputs = model(images)
        loss = criterion(outputs, labels)
        loss.backward()
        optimizer.step()

        running_loss += loss.item()

    print(f"Epoch {epoch+1}, Loss: {running_loss/len(train_loader):.4f}")

# Save model to S3
torch.save(model.state_dict(), 'model.pth')
s3.upload_file('model.pth', 'my-bucket', 'models/model.pth')
```

### Example 8: Hyperparameter Tuning with Optuna

**Scenario:** Optimize ML model hyperparameters

```bash
# Launch with deep learning environment
aws-jupyter launch \
  --env deep-learning \
  --instance-type m7g.xlarge \
  --connection session-manager
```

**Jupyter Notebook:**
```python
import optuna
from sklearn.datasets import load_breast_cancer
from sklearn.model_selection import cross_val_score
from sklearn.ensemble import RandomForestClassifier
import mlflow
import mlflow.sklearn

# Load data
X, y = load_breast_cancer(return_X_y=True)

# Define objective
def objective(trial):
    # Suggest hyperparameters
    params = {
        'n_estimators': trial.suggest_int('n_estimators', 10, 200),
        'max_depth': trial.suggest_int('max_depth', 2, 32),
        'min_samples_split': trial.suggest_int('min_samples_split', 2, 20),
        'min_samples_leaf': trial.suggest_int('min_samples_leaf', 1, 10),
    }

    # Train and evaluate
    clf = RandomForestClassifier(**params, random_state=42)
    score = cross_val_score(clf, X, y, cv=5, scoring='f1').mean()

    # Log to MLflow
    with mlflow.start_run(nested=True):
        mlflow.log_params(params)
        mlflow.log_metric('f1_score', score)

    return score

# Run optimization
study = optuna.create_study(direction='maximize')
study.optimize(objective, n_trials=100)

# Results
print(f"Best parameters: {study.best_params}")
print(f"Best F1 score: {study.best_value:.4f}")

# Visualize
optuna.visualization.plot_optimization_history(study).show()
optuna.visualization.plot_param_importances(study).show()

# Train final model
final_model = RandomForestClassifier(**study.best_params, random_state=42)
final_model.fit(X, y)

# Save to S3
import joblib
joblib.dump(final_model, 'final_model.pkl')

import boto3
s3 = boto3.client('s3')
s3.upload_file('final_model.pkl', 'my-bucket', 'models/final_model.pkl')
```

### Example 9: NLP with Transformers

**Scenario:** Fine-tune BERT for text classification

```bash
# Launch with ML environment and large instance
aws-jupyter launch \
  --env ml-pytorch \
  --instance-type m7g.2xlarge \
  --volume-size 150 \
  --connection session-manager
```

**Jupyter Notebook:**
```python
from transformers import (
    BertTokenizer, BertForSequenceClassification,
    Trainer, TrainingArguments
)
from datasets import load_dataset
import torch

# Load dataset
dataset = load_dataset('imdb')

# Load tokenizer and model
tokenizer = BertTokenizer.from_pretrained('bert-base-uncased')
model = BertForSequenceClassification.from_pretrained(
    'bert-base-uncased',
    num_labels=2
)

# Tokenize
def tokenize_function(examples):
    return tokenizer(
        examples['text'],
        padding='max_length',
        truncation=True,
        max_length=512
    )

tokenized_datasets = dataset.map(tokenize_function, batched=True)

# Training arguments
training_args = TrainingArguments(
    output_dir='./results',
    evaluation_strategy='epoch',
    learning_rate=2e-5,
    per_device_train_batch_size=8,
    per_device_eval_batch_size=8,
    num_train_epochs=3,
    weight_decay=0.01,
    save_strategy='epoch',
    load_best_model_at_end=True,
)

# Trainer
trainer = Trainer(
    model=model,
    args=training_args,
    train_dataset=tokenized_datasets['train'].select(range(1000)),  # Subset for demo
    eval_dataset=tokenized_datasets['test'].select(range(200)),
)

# Train
trainer.train()

# Evaluate
results = trainer.evaluate()
print(f"Evaluation results: {results}")

# Save model to S3
model.save_pretrained('./fine_tuned_bert')
tokenizer.save_pretrained('./fine_tuned_bert')

import boto3
s3 = boto3.client('s3')
!tar -czf model.tar.gz fine_tuned_bert/
s3.upload_file('model.tar.gz', 'my-bucket', 'models/bert_model.tar.gz')
```

## Research & Academia

### Example 10: Computational Biology Analysis

**Scenario:** Genomics data analysis and visualization

```bash
# Launch with computational biology environment
aws-jupyter launch \
  --env computational-bio \
  --instance-type m7g.xlarge \
  --volume-size 200 \
  --connection session-manager
```

**Jupyter Notebook:**
```python
from Bio import SeqIO
from Bio.Seq import Seq
from Bio.SeqUtils import GC
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns

# Download data from S3
import boto3
s3 = boto3.client('s3')
s3.download_file('my-bucket', 'sequences.fasta', 'sequences.fasta')

# Parse sequences
sequences = list(SeqIO.parse('sequences.fasta', 'fasta'))

# Analyze
results = []
for seq in sequences:
    results.append({
        'id': seq.id,
        'length': len(seq),
        'gc_content': GC(seq.seq),
        'description': seq.description
    })

df = pd.DataFrame(results)

# Visualize
fig, axes = plt.subplots(2, 2, figsize=(15, 10))

# Length distribution
df['length'].hist(bins=50, ax=axes[0,0])
axes[0,0].set_title('Sequence Length Distribution')
axes[0,0].set_xlabel('Length (bp)')

# GC content distribution
df['gc_content'].hist(bins=50, ax=axes[0,1])
axes[0,1].set_title('GC Content Distribution')
axes[0,1].set_xlabel('GC %')

# Scatter plot
axes[1,0].scatter(df['length'], df['gc_content'], alpha=0.5)
axes[1,0].set_title('Length vs GC Content')
axes[1,0].set_xlabel('Length (bp)')
axes[1,0].set_ylabel('GC %')

# Summary statistics
summary = df[['length', 'gc_content']].describe()
axes[1,1].axis('off')
axes[1,1].table(cellText=summary.values,
               rowLabels=summary.index,
               colLabels=summary.columns,
               loc='center')

plt.tight_layout()
plt.savefig('analysis.png')

# Upload results
s3.upload_file('analysis.png', 'my-bucket', 'results/analysis.png')
df.to_csv('results.csv', index=False)
s3.upload_file('results.csv', 'my-bucket', 'results/results.csv')
```

### Example 11: Statistical Analysis with R

**Scenario:** Statistical modeling in R

```bash
# Launch with R environment
aws-jupyter launch \
  --env r-stats \
  --instance-type m7g.large \
  --connection ssh
```

**R Notebook:**
```r
library(tidyverse)
library(broom)
library(patchwork)

# Load data from S3
library(aws.s3)
Sys.setenv("AWS_DEFAULT_REGION" = "us-west-2")

data <- s3read_using(
  FUN = read_csv,
  object = "s3://my-bucket/experiment-data.csv"
)

# Exploratory analysis
summary(data)

# Visualize
p1 <- ggplot(data, aes(x = treatment, y = response)) +
  geom_boxplot() +
  theme_minimal() +
  labs(title = "Response by Treatment")

p2 <- ggplot(data, aes(x = response, fill = treatment)) +
  geom_density(alpha = 0.5) +
  theme_minimal() +
  labs(title = "Response Distribution")

combined_plot <- p1 + p2
ggsave("exploratory.png", combined_plot, width = 12, height = 6)

# Statistical model
model <- lm(response ~ treatment + covariate1 + covariate2, data = data)
summary(model)

# Model diagnostics
par(mfrow = c(2, 2))
plot(model)

# Tidy results
results <- tidy(model) %>%
  mutate(significant = p.value < 0.05)

print(results)

# Save results to S3
write_csv(results, "model_results.csv")
s3write_using(
  results,
  FUN = write_csv,
  object = "s3://my-bucket/results/model_results.csv"
)
```

## Enterprise & Production

### Example 12: Production ML Pipeline

**Scenario:** Automated ML training pipeline for production

```bash
# Launch production instance in private subnet
aws-jupyter launch \
  --connection session-manager \
  --subnet-type private \
  --create-nat-gateway \
  --env deep-learning \
  --instance-type m7g.2xlarge \
  --tags "Environment=production,Project=ml-pipeline"
```

**Pipeline Notebook:**
```python
import mlflow
import mlflow.sklearn
from sklearn.pipeline import Pipeline
from sklearn.preprocessing import StandardScaler
from sklearn.ensemble import RandomForestClassifier
from sklearn.model_selection import train_test_split
from sklearn.metrics import classification_report
import boto3
import json

# Configure MLflow
mlflow.set_tracking_uri('http://mlflow-server:5000')
mlflow.set_experiment('production-model')

# Load data from S3
s3 = boto3.client('s3')
s3.download_file('production-bucket', 'data/training_data.parquet', 'data.parquet')

import pandas as pd
df = pd.read_parquet('data.parquet')

# Prepare data
X = df.drop('target', axis=1)
y = df['target']
X_train, X_test, y_train, y_test = train_test_split(
    X, y, test_size=0.2, random_state=42, stratify=y
)

# Create pipeline
pipeline = Pipeline([
    ('scaler', StandardScaler()),
    ('classifier', RandomForestClassifier(n_estimators=100, random_state=42))
])

# Train with MLflow tracking
with mlflow.start_run():
    # Log parameters
    mlflow.log_param('model_type', 'RandomForest')
    mlflow.log_param('n_estimators', 100)
    mlflow.log_param('train_size', len(X_train))

    # Train
    pipeline.fit(X_train, y_train)

    # Evaluate
    y_pred = pipeline.predict(X_test)
    report = classification_report(y_test, y_pred, output_dict=True)

    # Log metrics
    mlflow.log_metric('accuracy', report['accuracy'])
    mlflow.log_metric('f1_macro', report['macro avg']['f1-score'])
    mlflow.log_metric('precision_macro', report['macro avg']['precision'])
    mlflow.log_metric('recall_macro', report['macro avg']['recall'])

    # Log model
    mlflow.sklearn.log_model(pipeline, 'model')

    # Save classification report
    with open('report.json', 'w') as f:
        json.dump(report, f, indent=2)
    mlflow.log_artifact('report.json')

    # Register model
    model_uri = f"runs:/{mlflow.active_run().info.run_id}/model"
    mlflow.register_model(model_uri, 'production_classifier')

print("Model trained and registered successfully!")

# Deploy model artifacts to S3
import joblib
joblib.dump(pipeline, 'model.pkl')
s3.upload_file('model.pkl', 'production-bucket', 'models/latest/model.pkl')
s3.upload_file('report.json', 'production-bucket', 'models/latest/report.json')

print("Model deployed to S3 successfully!")
```

### Example 13: Data Quality Monitoring

**Scenario:** Monitor data quality for production pipelines

```bash
# Launch monitoring instance
aws-jupyter launch \
  --env data-science \
  --instance-type m7g.large \
  --connection session-manager \
  --subnet-type private \
  --tags "Environment=production,Purpose=monitoring"
```

**Monitoring Notebook:**
```python
import pandas as pd
import great_expectations as ge
from great_expectations.dataset import PandasDataset
import boto3
from datetime import datetime
import json

# Load latest production data
s3 = boto3.client('s3')
s3.download_file('production-bucket', 'data/latest/data.parquet', 'data.parquet')
df = pd.read_parquet('data.parquet')

# Create GE dataset
ge_df = PandasDataset(df)

# Define expectations
expectations = {
    'row_count': ge_df.expect_table_row_count_to_be_between(min_value=1000, max_value=1000000),
    'no_nulls_id': ge_df.expect_column_values_to_not_be_null('id'),
    'valid_age_range': ge_df.expect_column_values_to_be_between('age', min_value=0, max_value=120),
    'valid_categories': ge_df.expect_column_values_to_be_in_set('category', ['A', 'B', 'C']),
    'unique_ids': ge_df.expect_column_values_to_be_unique('id'),
}

# Check expectations
results = {
    'timestamp': datetime.now().isoformat(),
    'row_count': len(df),
    'checks': {}
}

for check_name, expectation in expectations.items():
    results['checks'][check_name] = {
        'success': expectation.success,
        'result': expectation.result
    }

# Alert if failures
failures = [k for k, v in results['checks'].items() if not v['success']]
if failures:
    print(f"❌ Data quality check FAILED: {failures}")

    # Send SNS alert
    sns = boto3.client('sns')
    sns.publish(
        TopicArn='arn:aws:sns:us-west-2:123456789012:data-quality-alerts',
        Subject='Data Quality Check Failed',
        Message=json.dumps(results, indent=2)
    )
else:
    print("✅ All data quality checks passed!")

# Save results
with open('quality_report.json', 'w') as f:
    json.dump(results, f, indent=2)

s3.upload_file(
    'quality_report.json',
    'production-bucket',
    f'monitoring/quality/{datetime.now().strftime("%Y/%m/%d")}/report.json'
)
```

## Development & Testing

### Example 14: Test Development Environment

**Scenario:** Test new packages before adding to production

```bash
# Launch test environment
aws-jupyter launch \
  --env minimal \
  --instance-type m7g.medium \
  --connection ssh \
  --tags "Environment=test,Purpose=package-testing"

# Connect
aws-jupyter connect i-test-instance

# In Jupyter, test new packages
```

**Test Notebook:**
```python
# Test new package versions
!pip install --upgrade new-package==2.0.0

# Run compatibility tests
import new_package
import existing_package

# Test integration
result = new_package.function(existing_package.data)
assert result.shape == (100, 10), "Output shape mismatch!"

# Test performance
import time
start = time.time()
new_package.bulk_operation(data)
duration = time.time() - start
print(f"Duration: {duration:.2f}s")
assert duration < 10, "Too slow!"

# If tests pass, update production requirements.txt
print("✅ All tests passed! Safe to update production.")
```

### Example 15: A/B Test Analysis

**Scenario:** Analyze A/B test results

```bash
aws-jupyter launch \
  --env data-science \
  --instance-type m7g.large \
  --connection session-manager
```

**Analysis Notebook:**
```python
import pandas as pd
import numpy as np
from scipy import stats
import matplotlib.pyplot as plt
import seaborn as sns

# Load A/B test data
import boto3
s3 = boto3.client('s3')
s3.download_file('analytics-bucket', 'ab_test_results.csv', 'data.csv')
df = pd.read_csv('data.csv')

# Split by variant
control = df[df['variant'] == 'A']['conversion_rate']
treatment = df[df['variant'] == 'B']['conversion_rate']

# Descriptive statistics
print("Control Group:")
print(f"  Mean: {control.mean():.4f}")
print(f"  Std:  {control.std():.4f}")
print(f"  N:    {len(control)}")

print("\nTreatment Group:")
print(f"  Mean: {treatment.mean():.4f}")
print(f"  Std:  {treatment.std():.4f}")
print(f"  N:    {len(treatment)}")

# Statistical test
t_stat, p_value = stats.ttest_ind(treatment, control)
print(f"\nT-test Results:")
print(f"  t-statistic: {t_stat:.4f}")
print(f"  p-value:     {p_value:.4f}")

# Effect size (Cohen's d)
pooled_std = np.sqrt((control.std()**2 + treatment.std()**2) / 2)
cohens_d = (treatment.mean() - control.mean()) / pooled_std
print(f"  Cohen's d:   {cohens_d:.4f}")

# Visualization
fig, axes = plt.subplots(1, 2, figsize=(15, 5))

# Distribution comparison
axes[0].hist(control, alpha=0.5, label='Control', bins=30)
axes[0].hist(treatment, alpha=0.5, label='Treatment', bins=30)
axes[0].legend()
axes[0].set_title('Conversion Rate Distribution')
axes[0].set_xlabel('Conversion Rate')

# Box plot
df.boxplot(column='conversion_rate', by='variant', ax=axes[1])
axes[1].set_title('Conversion Rate by Variant')
axes[1].set_xlabel('Variant')
axes[1].set_ylabel('Conversion Rate')

plt.tight_layout()
plt.savefig('ab_test_results.png')

# Recommendation
if p_value < 0.05:
    improvement = ((treatment.mean() - control.mean()) / control.mean()) * 100
    print(f"\n✅ RECOMMENDATION: Deploy variant B (+{improvement:.1f}% improvement, p<0.05)")
else:
    print(f"\n⚠️  RECOMMENDATION: Keep variant A (no significant difference, p={p_value:.4f})")

# Upload results
s3.upload_file('ab_test_results.png', 'analytics-bucket', 'results/ab_test.png')
```

## Cost Optimization

### Example 16: Budget-Conscious Development

**Scenario:** Minimize costs during development

```bash
# 1. Use smallest viable instance
aws-jupyter launch \
  --env minimal \
  --instance-type m7g.medium \
  --connection session-manager \
  --subnet-type private  # No NAT Gateway!

# 2. Stop when not in use (save 90% of costs)
aws-jupyter stop i-0abc123def456789

# 3. Start when needed
aws ec2 start-instances --instance-ids i-0abc123def456789

# 4. Connect
aws-jupyter connect i-0abc123def456789

# 5. Terminate when project done
aws-jupyter terminate i-0abc123def456789
```

**Monthly Cost:**
- Instance: $42/month * 25% usage = $10.50
- Storage: $3/month
- Total: **~$13.50/month**

### Example 17: Spot Instance Alternative (Future)

**Scenario:** Use spot instances for fault-tolerant workloads

```bash
# Not yet supported by aws-jupyter, but planned for v0.5.0
# Future command might look like:
aws-jupyter launch \
  --instance-market spot \
  --spot-max-price 0.05 \
  --env ml-pytorch

# Potential savings: 50-70% off on-demand price
```

## Team Collaboration

### Example 18: Shared Research Environment

**Scenario:** Team of researchers sharing instances

```bash
# Team lead launches instance
aws-jupyter launch \
  --env data-science \
  --instance-type m7g.xlarge \
  --connection session-manager \
  --subnet-type private \
  --create-nat-gateway \
  --tags "Team=research,Project=genomics,Owner=team-lead"

# Share instance ID with team
# Team members connect:
aws-jupyter connect i-shared-instance

# Everyone works in their own notebook directory
# /home/ubuntu/notebooks/alice/
# /home/ubuntu/notebooks/bob/
# /home/ubuntu/notebooks/carol/
```

**Best Practices:**
```bash
# In instance, create user directories
sudo mkdir -p /home/ubuntu/notebooks/{alice,bob,carol}
sudo chown -R ubuntu:ubuntu /home/ubuntu/notebooks/

# Set up Git for collaboration
cd /home/ubuntu/notebooks
git init
git remote add origin https://github.com/team/research-notebooks

# Regular sync
git add .
git commit -m "Daily progress"
git push
```

## Complete Workflows

### Example 19: End-to-End ML Project

**Complete workflow from data to deployment**

```bash
# Phase 1: Data Exploration (2 hours)
aws-jupyter launch \
  --env data-science \
  --instance-type m7g.medium \
  --connection ssh

# Do exploratory data analysis in Jupyter
# Stop when done
aws-jupyter stop i-0abc123def456789

# Phase 2: Model Training (4 hours next day)
aws ec2 start-instances --instance-ids i-0abc123def456789
aws-jupyter connect i-0abc123def456789

# Train models, compare approaches
# Stop when done
aws-jupyter stop i-0abc123def456789

# Phase 3: Hyperparameter Tuning (overnight)
# Upgrade to larger instance for speed
# (Stop instance, modify via console, start)

aws ec2 start-instances --instance-ids i-0abc123def456789
aws-jupyter connect i-0abc123def456789

# Run optimization overnight
# Let it complete

# Phase 4: Production Deployment
# Launch separate production instance
aws-jupyter launch \
  --env ml-pytorch \
  --instance-type m7g.large \
  --connection session-manager \
  --subnet-type private \
  --create-nat-gateway \
  --tags "Environment=production"

# Deploy best model from training
# Set up monitoring
# Terminate training instance

aws-jupyter terminate i-0abc123def456789
```

**Total Cost Estimate:**
- Training (m7g.medium, 6 hours): $0.30
- Hyperparameter tuning (m7g.xlarge, 12 hours): $1.46
- Production (m7g.large, 720 hours/month): $61.32
- NAT Gateway: $32.40
- **Total Month 1: ~$95**
- **Ongoing: ~$94/month**

### Example 20: Research Paper Workflow

**Complete academic research workflow**

```bash
# Step 1: Literature review and data collection
aws-jupyter launch \
  --env minimal \
  --instance-type m7g.medium \
  --connection ssh

# Collect and organize data
# Create initial analysis notebooks
aws-jupyter stop i-0abc123def456789

# Step 2: Analysis and modeling (1 week)
aws ec2 start-instances --instance-ids i-0abc123def456789
# Work 2-3 hours daily
# Stop each day

# Step 3: Generate figures and tables (2 days)
# Run all final analyses
# Create publication-quality figures
# Export results

# Step 4: Write paper
# Download results locally
# Terminate instance when analysis complete
aws-jupyter terminate i-0abc123def456789

# Total cost: ~$15-20 for 2-week project
```

## Additional Resources

**Documentation:**
- [Main README](../README.md)
- [Session Manager Setup](SESSION_MANAGER_SETUP.md)
- [Private Subnet Guide](PRIVATE_SUBNET_GUIDE.md)
- [Troubleshooting](TROUBLESHOOTING.md)
- [Roadmap](../ROADMAP.md)

**AWS Resources:**
- [EC2 Pricing](https://aws.amazon.com/ec2/pricing/)
- [S3 Integration](https://docs.aws.amazon.com/cli/latest/reference/s3/)
- [IAM Best Practices](https://docs.aws.amazon.com/IAM/latest/UserGuide/best-practices.html)

---

**Have your own use case?** Share it in [GitHub Discussions](https://github.com/scttfrdmn/aws-jupyter/discussions) or open a PR to add it to this document!
