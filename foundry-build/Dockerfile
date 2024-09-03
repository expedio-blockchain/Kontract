# Use an ARM-compatible base image
FROM ubuntu:20.04

# Set environment variable to avoid interactive prompts
ENV DEBIAN_FRONTEND=noninteractive

# Create a non-root user and group
RUN groupadd -r foundrygroup && useradd -r -m -g foundrygroup -s /bin/bash foundryuser

# Install dependencies
RUN apt-get update && apt-get install -y \
    curl \
    git \
    build-essential \
    libusb-1.0-0-dev \
    nodejs \
    npm \
    jq \
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

USER root

# Copy the entrypoint script
COPY entrypoint.sh /home/foundryuser/entrypoint.sh
# Ensure the script has execution permissions
RUN chown -R foundryuser:foundrygroup /home/foundryuser/entrypoint.sh
RUN chmod u+x /home/foundryuser/entrypoint.sh

# Create the project directory with the correct permissions
USER foundryuser
RUN mkdir -p /home/foundryuser/expedio-kontract-deployer

# Set the working directory to the project directory
WORKDIR /home/foundryuser/expedio-kontract-deployer

# Initialize a default Foundry project
RUN forge init --no-git .

# Clean up unnecessary default files
RUN rm -rf script/* test/* src/*

# Specify the command to run your project
CMD ["/home/foundryuser/entrypoint.sh"]
