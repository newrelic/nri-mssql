FROM mcr.microsoft.com/mssql/server:2017-latest

USER root

# Create directories for our custom setup scripts and database files
RUN mkdir -p /usr/config && \
    mkdir -p /var/opt/mssql/backup && \
    mkdir -p /var/opt/mssql/data

# Install wget
RUN apt-get update && apt-get install -y wget && rm -rf /var/lib/apt/lists/*

# Download AdventureWorks backup during build
RUN wget -q https://github.com/Microsoft/sql-server-samples/releases/download/adventureworks/AdventureWorksLT2017.bak -O /var/opt/mssql/backup/adventureworks-light.bak

# Copy our configuration files
COPY querystore-config.sql /usr/config/
COPY enable-special-passwords.sql /usr/config/
COPY entrypoint.sh /usr/config/

# Set permissions after all files are in place
RUN chmod +r /usr/config/*.sql && \
    chmod +x /usr/config/entrypoint.sh && \
    chown -R root /usr/config && \
    chown -R root /var/opt/mssql && \
    chmod -R 775 /var/opt/mssql/backup && \
    chmod 664 /var/opt/mssql/backup/adventureworks-light.bak

# USER 10001

ENTRYPOINT ["/usr/config/entrypoint.sh"]