#!/bin/bash

# Start SQL Server
/opt/mssql/bin/sqlservr & 

# Wait for SQL Server to start
sleep 60

# Run SQL commands with SSL certificate verification disabled
/opt/mssql-tools18/bin/sqlcmd -S localhost -U sa -P "secret123!" -C -t 60 -I -i /usr/config/enable-special-passwords.sql

# Run the setup script to restore database and configure Query Store
/opt/mssql-tools18/bin/sqlcmd -S localhost -U sa -P "secret123!" -C -t 120 -I -i /usr/config/querystore-config.sql

# Keep container running
tail -f /dev/null