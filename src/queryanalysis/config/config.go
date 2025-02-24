package config

// Documentation: https:https://newrelic.atlassian.net/wiki/x/SYFq6g
// The above link contains all the queries, data models, and query details for QueryAnalysis.

// We need to use this limit of long strings that we are injesting because the logs datastore in New Relic limits the field length to 4,094 characters. Any data longer than that is truncated during ingestion.
const TextTruncateLimit = 4094

const (
	// QueryResponseTimeThresholdDefault defines the default threshold in milliseconds
	// for determining if a query is considered slow based on its response time.
	QueryResponseTimeThresholdDefault = 500

	// SlowQueryCountThresholdDefault sets the default maximum number of slow queries
	// that is ingested in an analysis cycle/interval.
	SlowQueryCountThresholdDefault = 20

	// IndividualQueryCountMax represents the maximum number of individual queries
	// that is ingested at one time for any grouped query in detailed analysis.
	IndividualQueryCountMax = 10

	// GroupedQueryCountMax specifies the maximum number of grouped queries
	// that is ingested in  an analysis cycle/interval.
	GroupedQueryCountMax = 30

	// MaxSystemDatabaseID indicates the highest database ID value considered
	// a system database, used to filter out system databases from certain operations.
	MaxSystemDatabaseID = 4
	BatchSize           = 600 // New Relic's Integration SDK imposes a limit of 1000 metrics per ingestion.To handle metric sets exceeding this limit, we process and ingest metrics in smaller chunks to ensure all data is successfully reported without exceeding the limit.

)
