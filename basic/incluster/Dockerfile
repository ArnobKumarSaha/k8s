FROM debian:bookworm

# Update the package list and install CA certificates
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Copy the application file
COPY ./app /app

# Ensure the application has execute permissions
RUN chmod +x /app

# Set the entrypoint
ENTRYPOINT ["/app"]
