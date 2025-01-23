FROM mcr.microsoft.com/mssql/server:2022-latest

# Switch to root to create directories and set permissions
USER root

# Create directories for our custom setup scripts and database files
RUN mkdir -p /usr/config
RUN mkdir -p /var/opt/mssql/backup

# Set up SQL Server directories and permissions
RUN mkdir -p /var/opt/mssql && \
    chown -R mssql /var/opt/mssql

# Install wget
RUN apt-get update && apt-get install -y wget && rm -rf /var/lib/apt/lists/*

# Download AdventureWorks backup during build
RUN wget -q https://github.com/Microsoft/sql-server-samples/releases/download/adventureworks/AdventureWorksLT2022.bak -O /var/opt/mssql/backup/adventureworks-light.bak

# Ensure proper permissions on all SQL Server directories and files
RUN chown -R mssql:root /var/opt/mssql/backup && \
    chmod -R 775 /var/opt/mssql/backup && \
    chown mssql:root /var/opt/mssql/backup/adventureworks-light.bak && \
    chmod 664 /var/opt/mssql/backup/adventureworks-light.bak

# Copy our configuration files
COPY querystore-config.sql /usr/config/
COPY enable-special-passwords.sql /usr/config/
RUN chmod +r /usr/config/*.sql

COPY entrypoint.sh /usr/config/
RUN chmod +x /usr/config/entrypoint.sh

# Switch back to mssql user
# USER mssql

ENTRYPOINT ["/usr/config/entrypoint.sh"]