# Installation Guide

## Prerequisites

- Kubernetes cluster (version 1.18+)
- Helm (version 3+)

## Installation Steps

1. **Clone the Git Repository**

   ```bash
   git clone https://github.com/expedio-xyz/kontract.git
   ```

2. **Install the Kontract Operator**

   ```bash
   helm install kontract ./helm-chart --namespace kontract --create-namespace
   ```

   This command installs the Kontract operator in the `kontract` namespace.
