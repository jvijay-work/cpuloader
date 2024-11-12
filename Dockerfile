# Use scratch as the base image
FROM scratch

# Set a working directory
WORKDIR /app

# Copy the statically compiled cpuloader binary to the container
COPY cpuloader /app/cpuloader

# Expose the port on which the web server listens
EXPOSE 8080

# Command to run your web server
CMD ["/app/cpuloader"]
