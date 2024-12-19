# Installation Guide

## Prerequisites

- A Kubernetes cluster (version 1.18 or higher)
- Helm (version 3 or higher)

## Installation Steps

1. **Clone the Git Repository**

   Clone the Kontract repository:

   ```bash
   git clone https://github.com/expedio-xyz/kontract.git
   cd Kontract
   ```

2. **Install the Kontract Operator**

   Use Helm to install the Kontract operator:

   ```bash
   helm install kontract ./helm-chart --namespace kontract --create-namespace
   ```

   This command installs the Kontract operator in the `kontract` namespace.

## What's Next?

- [Getting Started](getting-started.md)
