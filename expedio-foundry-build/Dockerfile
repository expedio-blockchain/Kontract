# Use an ARM-compatible base image
FROM ubuntu:20.04

# Create a non-root user and group
RUN groupadd -r foundrygroup && useradd -r -m -g foundrygroup -s /bin/bash foundryuser

# Install dependencies
RUN apt-get update && apt-get install -y \
    curl \
    git \
    build-essential \
    libusb-1.0-0-dev \
    && rm -rf /var/lib/apt/lists/*

# Set environment variables
ENV XDG_CONFIG_HOME=/home/foundryuser \
    FOUNDRY_DIR=/home/foundryuser/.foundry

# Run the foundry installation script as the non-root user
USER foundryuser
RUN mkdir -p "$FOUNDRY_DIR/bin" \
    && curl -sSf -L https://raw.githubusercontent.com/foundry-rs/foundry/master/foundryup/foundryup -o "$FOUNDRY_DIR/bin/foundryup" \
    && chmod +x "$FOUNDRY_DIR/bin/foundryup" \
    && "$FOUNDRY_DIR/bin/foundryup"

# Add Foundry binaries to the PATH
ENV PATH="$FOUNDRY_DIR/bin:$PATH"

# Copy the pre-downloaded solc version 0.8.26 to the container
USER root
COPY svm/0.8.26 /root/.svm/0.8.26

# Change ownership of the copied files to the non-root user
RUN chown -R foundryuser:foundrygroup /root/.svm

# Create the project directory with the correct permissions
USER foundryuser
RUN mkdir -p /home/foundryuser/expedio-kontract-deployer

# Set the working directory to the project directory
WORKDIR /home/foundryuser/expedio-kontract-deployer

# Initialize a default Foundry project
RUN forge init --no-git .

# Specify the command to run your project
CMD ["bash"]
