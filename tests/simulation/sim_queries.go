package simulation

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/microsoft/go-mssqldb"
)

// Helper function to create database connection
func openDB(host string, port int, database, user, password string) (*sqlx.DB, error) {
	query := url.Values{}
	query.Add("database", database)
	query.Add("encrypt", "disable") // Disable encryption for testing

	// Build connection string in MSSQL format
	dsn := fmt.Sprintf("sqlserver://%s:%s@%s:%d?%s",
		url.QueryEscape(user),
		url.QueryEscape(password),
		host,
		port,
		query.Encode(),
	)

	db, err := sqlx.Open("sqlserver", dsn)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to db: %s", err) //nolint:all
	}

	// Test connection
	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("cannot ping db: %s", err) //nolint:all
	}

	return db, nil
}

//nolint:gocyclo
func SimulateDBQueries(t *testing.T, port int, database, user, password string) error {
	t.Helper()

	// Create primary database connection
	db1, err := openDB("localhost", port, database, user, password)
	if err != nil {
		return fmt.Errorf("failed to open first database connection: %v", err) //nolint:all
	}

	// Create secondary connection for blocking sessions
	db2, err := openDB("localhost", port, database, user, password)
	if err != nil {
		db1.Close()
		return fmt.Errorf("failed to open second database connection: %v", err) //nolint:all
	}

	t.Log("Executing simulation queries...")

	// Execute generic queries first
	for _, query := range genericQueries {
		if _, err := db1.Exec(query); err != nil {
			db1.Close()
			db2.Close()
			return fmt.Errorf("error executing generic query: %v", err) //nolint:all
		}
	}

	// Simulate slow queries
	for _, query := range slowQueries {
		if _, err := db1.Exec(query); err != nil {
			db1.Close()
			db2.Close()
			return fmt.Errorf("error executing slow query: %v", err) //nolint:all
		}
	}

	// Simulate query plans
	for _, query := range planQueries {
		if _, err := db1.Exec(query); err != nil {
			db1.Close()
			db2.Close()
			return fmt.Errorf("error executing plan query: %v", err) //nolint:all
		}
	}

	// Start blocking session simulation in a goroutine
	go func() {
		defer db1.Close()
		defer db2.Close()

		// Start first transaction
		tx1, err := db1.Begin()
		if err != nil {
			t.Logf("Error starting blocking transaction: %v", err)
			return
		}

		// Lock a row in the first transaction
		_, err = tx1.Exec(blockingSessionQuery)
		if err != nil {
			tx1.Rollback() //nolint:all
			t.Logf("Error in first blocking query: %v", err)
			return
		}

		// Start a goroutine for the blocked query
		go func() {
			// Try to update the same row in second connection
			_, err := db2.Exec(blockingSessionQuery)
			if err != nil {
				t.Logf("Expected blocking error in second connection: %v", err)
			}
		}()

		// Hold the lock briefly
		time.Sleep(5 * time.Second) //nolint:all
		tx1.Rollback()              //nolint:all
	}()

	// Brief sleep to ensure blocking is established
	// time.Sleep(2 * time.Second) //nolint:all
	return nil
}

var genericQueries = []string{
	// Query 1: Basic select from Customer table
	`SELECT TOP 100 CustomerID, FirstName, LastName, ModifiedDate
     FROM SalesLT.Customer
     WHERE LastName LIKE 'A%'`,

	// Query 2: Simple join between Product and ProductCategory
	`SELECT TOP 100 p.ProductID, p.Name, pc.Name as CategoryName
     FROM SalesLT.Product p
     JOIN SalesLT.ProductCategory pc ON p.ProductCategoryID = pc.ProductCategoryID`,

	// Query 3: Basic aggregation on SalesOrderDetail
	`SELECT ProductID, COUNT(*) as OrderCount
     FROM SalesLT.SalesOrderDetail
     GROUP BY ProductID
     ORDER BY OrderCount DESC`,

	// Query 4: Simple date filtering
	`SELECT ProductID, Name, ModifiedDate
     FROM SalesLT.Product
     WHERE ModifiedDate > DATEADD(year, -5, GETDATE())`,

	// Query 5: Basic subquery
	`SELECT c.CustomerID, c.FirstName, c.LastName
     FROM SalesLT.Customer c
     WHERE EXISTS (
         SELECT 1
         FROM SalesLT.SalesOrderHeader soh
         WHERE soh.CustomerID = c.CustomerID
     )`,
}

// Slow queries - intentionally complex operations
var slowQueries = []string{
	// Query 1: Complex joins with window functions
	`SELECT 
        c.CustomerID,
        c.FirstName,
        c.LastName,
        COUNT(DISTINCT soh.SalesOrderID) TotalOrders,
        SUM(soh.TotalDue) TotalSpent,
        AVG(soh.TotalDue) AvgOrderAmount,
        a.AddressLine1,
        a.City,
        a.StateProvince,
        DENSE_RANK() OVER (ORDER BY SUM(soh.TotalDue) DESC) SpendRank
    FROM SalesLT.Customer c
    LEFT JOIN SalesLT.SalesOrderHeader soh ON c.CustomerID = soh.CustomerID
    LEFT JOIN SalesLT.CustomerAddress ca ON c.CustomerID = ca.CustomerID
    LEFT JOIN SalesLT.Address a ON ca.AddressID = a.AddressID
    GROUP BY 
        c.CustomerID, 
        c.FirstName, 
        c.LastName,
        a.AddressLine1,
        a.City,
        a.StateProvince
    ORDER BY TotalSpent DESC`,

	// Query 2: Multiple joins with aggregations
	`SELECT 
        p.ProductID,
        p.Name,
        p.ListPrice,
        COUNT(sod.SalesOrderDetailID) TimesOrdered,
        AVG(sod.UnitPrice) AvgPrice,
        MAX(sod.UnitPrice) MaxPrice,
        SUM(sod.OrderQty) TotalQty
    FROM SalesLT.Product p
    LEFT JOIN SalesLT.SalesOrderDetail sod ON p.ProductID = sod.ProductID
    WHERE p.ListPrice > 0
    GROUP BY 
        p.ProductID,
        p.Name,
        p.ListPrice
    ORDER BY TimesOrdered DESC, p.ListPrice DESC`,

	// Query 3: Complex joins and aggregations
	`SELECT 
        pc.Name CategoryName,
        p.Name ProductName,
        COUNT(sod.SalesOrderDetailID) OrderCount,
        SUM(sod.OrderQty) TotalQuantity,
        SUM(sod.UnitPrice * sod.OrderQty) TotalRevenue,
        AVG(sod.UnitPrice) AvgUnitPrice,
        MIN(soh.OrderDate) FirstOrder,
        MAX(soh.OrderDate) LastOrder
    FROM SalesLT.ProductCategory pc
    JOIN SalesLT.Product p ON pc.ProductCategoryID = p.ProductCategoryID
    LEFT JOIN SalesLT.SalesOrderDetail sod ON p.ProductID = sod.ProductID
    LEFT JOIN SalesLT.SalesOrderHeader soh ON sod.SalesOrderID = soh.SalesOrderID
    GROUP BY pc.Name, p.Name
    HAVING COUNT(sod.SalesOrderDetailID) > 0
    ORDER BY TotalRevenue DESC`,
}

// Queries that force specific execution plans
var planQueries = []string{
	`SELECT CustomerID, FirstName, LastName, ModifiedDate
     FROM SalesLT.Customer WITH (FORCESCAN)
     WHERE LastName LIKE 'S%'`,

	`SELECT p.ProductID, p.Name, COUNT(*) as OrderCount
     FROM SalesLT.Product p WITH (FORCESEEK)
     JOIN SalesLT.SalesOrderDetail sod ON p.ProductID = sod.ProductID
     GROUP BY p.ProductID, p.Name
     HAVING COUNT(*) > 10`,
}

// Query to create blocking sessions
var blockingSessionQuery = `
    UPDATE SalesLT.SalesOrderHeader 
    SET SubTotal = SubTotal * 1.1 
    WHERE SalesOrderID = (
        SELECT TOP 1 SalesOrderID 
        FROM SalesLT.SalesOrderHeader 
        ORDER BY ModifiedDate DESC
    )
`
