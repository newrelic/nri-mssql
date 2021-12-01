package queryplan

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/clbanning/mxj/v2"
	"io/ioutil"
	"reflect"
	"regexp"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-mssql/src/args"
	"github.com/newrelic/nri-mssql/src/connection"
	"gopkg.in/yaml.v2"
)

/**
Trying to report the query plan data through Infra as metrics runs into a number of issues related to number of attributes and value size so we need to send them back
directly via the LogEntry API so we can send the data back as actual JSON rather than a String containing a huge chuck of stringified JSON. The JSON approach is also easier
on the visualizer.
*/
type customQuery struct {
	Query    string
	Prefix   string
	Name     string `yaml:"metric_name"`
	Type     string `yaml:"metric_type"`
	Database string
}

var arguments args.ArgumentList

func PopulateQueryPlan(connection *connection.SQLConnection, al args.ArgumentList) {
	arguments = al
	log.Debug("PopulateQueryPlan: enter")
	queries, err := parseCustomQueries()
	if err != nil {
		log.Error("Failed to parse query plan queries: %s", err)
	}
	var wg sync.WaitGroup
	for _, query := range queries {
		wg.Add(1)
		go func(query customQuery) {
			defer wg.Done()
			populateLogEvent(connection, query)
		}(query)
	}
	wg.Wait()
	log.Debug("PopulateQueryPlan: exit")
}

func parseCustomQueries() ([]customQuery, error) {
	log.Debug("parseCustomQueries: enter")
	// load YAML config file
	b, err := ioutil.ReadFile(arguments.QueryPlanConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to read custom_metrics_config: %s", err)
	}
	// parse
	var c struct{ Queries []customQuery }
	err = yaml.Unmarshal(b, &c)
	if err != nil {
		return nil, fmt.Errorf("failed to parse custom_metrics_config: %s", err)
	}

	log.Debug("parseCustomQueries: exit")
	return c.Queries, nil
}

type LogEntry struct {
	Timestamp  int64                  `json:"timestamp"`
	Attributes map[string]interface{} `json:"attributes"`
	Message    string                 `json:"message"`
	Source     string                 `json:"source"`
}

type LogEvent struct {
	LogEntries []LogEntry `json:"logs"`
}

var logEvent LogEvent
var eventSize int

func populateLogEvent(connection *connection.SQLConnection, query customQuery) {
	log.Debug("populateLogEvent: enter")

	var prefix string
	if len(query.Database) > 0 {
		prefix = "USE " + query.Database + "; "
	}

	rows, err := connection.Queryx(prefix + query.Query)
	if err != nil {
		log.Error("Could not execute custom query: %s", err)
		return
	}

	defer func() {
		_ = rows.Close()
	}()

RowLoop:
	for rows.Next() {
		row := make(map[string]interface{})
		err := rows.MapScan(row)
		if err != nil {
			log.Error("Failed to scan custom query row: %s", err)
			return
		}

		var logEntry LogEntry
		logEntry.Timestamp = time.Now().UnixMilli()
		logEntry.Message = "Query Plan"
		logEntry.Attributes = make(map[string]interface{})
		logEntry.Source = "SQL Server"

		for k, v := range row {
			if k == "query_plan" && (v == nil || v == "") {
				continue RowLoop
			}
			if k == "query_plan"{
				start := time.Now()
				log.Debug("Query plan original length: %d", len(v.(string)))

				// Remove extra white space from the original XML String
				spaces := regexp.MustCompile(`\s+`)
				xml := spaces.ReplaceAllString(v.(string), " ")
				log.Debug("\t trimmed XML Length: %d", len(xml))
				//v = xml

				// Turn XML into Go map
				queryPlan, err := mxj.NewMapXml([]byte(xml))
				if err != nil {
					log.Error("Error unmarshaling XML: %s", err)
					continue RowLoop
				}

				// Turn Go Map into JSON
				qpJson, err := json.Marshal(queryPlan)
				if err != nil {
					log.Error("Error marshaling json: %s", err)
					continue RowLoop
				}
				log.Debug("\t JSON length: %d", len(qpJson))

				// Compress JSON
				var buf bytes.Buffer
				zw := gzip.NewWriter(&buf)
				_, err = zw.Write(qpJson)
				if err != nil {
					log.Fatal(err)
				}
				if err := zw.Close(); err != nil {
					log.Fatal(err)
				}
				log.Debug("\t GZIP'd length: %d", buf.Len())

				// Base64 encode compressed JSON
				encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
				log.Debug("\t Base64 encoded length: %d", len(encoded))
				if len(encoded) > 128000{
					log.Warn("Query plan compressed and encoded is greater than 128k and will be truncated upon ingestion")
				}
				v = encoded
				log.Debug("\t Total time: %v", time.Since(start))
			}
			// Get a rough idea of the size of the data
			value := reflect.ValueOf(v)
			kind := value.Kind()
			if kind == reflect.String {
				eventSize += len(value.String())
			}
			logEntry.Attributes[k] = v
		}
		addLog(logEntry)
	}
	publishEvent()
	log.Debug("populateCustomMetrics: exit")
}

func addLog(entry LogEntry) {
	if eventSize >= 1000000 {
		publishEvent()
		eventSize = 0
		logEvent.LogEntries = nil
	}
	logEvent.LogEntries = append(logEvent.LogEntries, entry)
}

var client = resty.New()

func publishEvent() {
	//headers := map[string]string{"Content-Type": "application/json",  "Api-Key": arguments.LicenseKey}
	headers := map[string]string{"Content-Type": "application/json", "Content-Encoding": "gzip", "Api-Key": arguments.LicenseKey}
	type PostResult interface {
	}
	type PostError interface {
	}
	var postResult PostResult
	var postError PostError

	// Marshal the body
	body, err := json.Marshal([]LogEvent{logEvent})
	if err != nil {
		log.Error("Error marshaling json: %s", err)
	}

	// Compress the body
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err = zw.Write(body)
	zw.Close()
	if err != nil {
		log.Error("Error compressing Event", err)
	}
	if err := zw.Close(); err != nil {
		log.Error("Error closing gzip writer", err)
	}

	resp, err := client.R().
		//SetBody([]LogEvent{logEvent}).
		SetBody(buf.Bytes()).
		SetHeaders(headers).
		SetResult(&postResult).
		SetError(&postError).
		Post(arguments.LogApiEndpoint)

	if err != nil {
		log.Error("Error POSTing query plan", err)
	}
	if resp.StatusCode() >= 300 {
		log.Error("Bad status code POSTing query plan", resp.Status())
	} else {
		log.Debug("Status code POSTing query plan", resp.Status())
	}
}
